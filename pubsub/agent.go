package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"strings"
	"sync"
)

type subKey map[string][]byte // subscriber to public key map

// performance may be improved by using granular locks and
// trading off with complexity and memory utilization
type topicStore struct {
	*sync.RWMutex
	subs map[string]subKey // can extend to multiple keys per peer
}

type sockets struct {
	sktPubState *zmq.Socket
	sktSubState *zmq.Socket
	sktPubMsgs  *zmq.Socket
	sktSubMsgs  *zmq.Socket
}

type Agent struct {
	myLabel     string
	myInv       string
	myPubEndpnt string
	groups      map[string][]models.Member
	prb         services.Agent
	client      services.Client
	km          services.KeyManager
	log         log.Logger
	ts          *topicStore
	*sockets
}

func (a *Agent) Init() {
	// add handler for subscribe messages
	// add handler for group-join requests as sync
	//  - upon request, check if eligible
	//  - if eligible
	//	  - A sends group-info
	// create PUB and SUB sockets for msgs and statuses
}

func (a *Agent) Join(topic, acceptor string, publisher bool) error {
	// check if already joined to the topic
	if _, ok := a.groups[topic]; ok {
		return fmt.Errorf(`already connected to group %s`, topic)
	}
	a.groups[topic] = []models.Member{}

	// check if B is already connected with acceptor (A)
	// - only needs to check if ever connected (in agent's map), since disconnect is not implemented
	// if not, return
	p, err := a.prb.Peer(acceptor)
	if err != nil {
		return fmt.Errorf(`fetching acceptor failed - %v`, err)
	}

	// if connected, check if A's DID Doc has join-endpoint
	//  - save services in prober's peers map upon connection
	//  - open a func to get peer data
	var srvcJoin models.Service
	for _, s := range p.Services {
		if s.Type == domain.ServcGroupJoin {
			srvcJoin = s
			break
		}
	}

	// if join service is not present, return
	if srvcJoin.Type == `` {
		return fmt.Errorf(`acceptor does not provide group-join service`)
	}

	// call A's group-join/<topic> endpoint
	byts, err := json.Marshal(messages.ReqGroupJoin{Topic: topic, RequesterInv: a.myInv})
	if err != nil {
		return fmt.Errorf(`marshalling group-join request failed - %v`, err)
	}

	res, err := a.client.Send(domain.MsgTypGroupJoin, byts, srvcJoin.Endpoint)
	if err != nil {
		return fmt.Errorf(`group-join request failed - %v`, err)
	}

	// save received group info in-memory (map with topics?)
	var resGroup messages.ResGroupJoin
	if err = json.Unmarshal([]byte(res), &resGroup); err != nil {
		return fmt.Errorf(`unmarshalling group-join response failed - %v`, err)
	}

	// for each pub in group info
	// - connect
	// - stores/updates pub in-memory
	for _, m := range resGroup.Members {
		if !m.Active {
			continue
		}

		if err = a.connect(m); err != nil {
			return fmt.Errorf(`connecting to %s failed - %v`, m.Label, err)
		}

		if err = a.subscribe(topic, m); err != nil {
			return fmt.Errorf(`subscribing to topic %s with %s failed - %v`, topic, m.Label, err)
		}

		a.groups[topic] = append(a.groups[topic], m)
	}

	// init conn listener
	// init status listener
	// init message listener

	// publish status
	if err = a.notifyAll(topic, true); err != nil {
		return fmt.Errorf(`publishing status active failed - %v`, err)
	}

	return nil
}

func (a *Agent) connect(m models.Member) error {
	// if not already connected
	// - sets up DIDComm connection via inv
	_, err := a.prb.Peer(m.Label)
	if err != nil {
		if err = a.prb.SyncAccept(m.Inv); err != nil {
			return fmt.Errorf(`accepting group-member invitation failed - %v`, err)
		}
	}

	// B connects to pub via SUB for statuses and msgs
	if err = a.sktSubState.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher state socket failed - %v`, err)
	}

	if err = a.sktSubMsgs.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
	}

	return nil
}

func (a *Agent) subscribe(topic string, m models.Member) error {
	// get my public key corresponding to this member
	subPublcKey, err := a.km.PublicKey(m.Label)
	if err != nil {
		return fmt.Errorf(`fetching public key for the connection failed - %v`, err)
	}

	// B sends agent subscribe msg to pub
	subTopic := topic + `_` + m.Label + `_` + a.myLabel
	sm := messages.SubscribeMsg{
		Id:          uuid.New().String(),
		Type:        messages.SubscribeV1,
		Subscribe:   true,
		Peer:        a.myLabel,
		PubKey:      base58.Encode(subPublcKey),
		Topics:      []string{subTopic},
		PubEndpoint: a.myPubEndpnt,
	}

	byts, err := json.Marshal(sm)
	if err != nil {
		return fmt.Errorf(`marshalling subscribe message failed - %v`, err)
	}

	if err = a.prb.SendMessage(domain.MsgTypSubscribe, m.Label, string(byts)); err != nil {
		return fmt.Errorf(`sending subscribe message failed for topic %s - %v`, subTopic, err)
	}

	// B subscribes via zmq
	if err = a.sktSubMsgs.SetSubscribe(subTopic); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, subTopic, err)
	}

	return nil
}

func (a *Agent) initConnListener(subChan chan models.Message) {
	// if B is a publisher
	// - if didcomm connection is established
	//   - if sub msg is received
	//     - B connects to sub via SUB for statuses

	for {
		msg := <-subChan
		unpackedMsg, err := a.prb.ReadMessage(msg)
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading subscribe message failed - %v`, err))
			continue
		}

		var sm messages.SubscribeMsg
		if err = json.Unmarshal([]byte(unpackedMsg), &sm); err != nil {
			a.log.Error(fmt.Sprintf(`unmarshalling subscribe message failed - %v`, err))
			continue
		}

		if !sm.Subscribe {
			a.deleteSub(sm)
			continue
		}

		if !a.validJoiner(sm.Peer) {
			a.log.Error(fmt.Sprintf(`requester (%s) is not eligible`, sm.Peer))
			continue
		}

		sk := base58.Decode(sm.PubKey)
		for _, t := range sm.Topics {
			a.addSub(t, sm.Peer, sk)
		}

		// connect to subscriber's publishing endpoint for further status changes
		if err = a.sktSubState.Connect(sm.PubEndpoint); err != nil {
			a.log.Error(fmt.Sprintf(`connecting to subscriber (%s) endpoint for state changes failed - %v`, sm.Peer, err))
		}
	}
}

func (a *Agent) initStatusListener() {
	// if status received,
	// - store/update in-memory
	// - if active
	//  - if B is a sub
	//    - if sender is a pub
	//      - if not already connected
	//		  - connect
	// - if not active, remove/disconnect

	for {
		msg, err := a.sktSubState.Recv(0)
		if err != nil {
			a.log.Error(fmt.Sprintf(`receiving zmq message for member status failed - %v`, err))
			continue
		}

		ms, err := a.parseMembrStatus(msg)
		if err != nil {
			a.log.Error(fmt.Sprintf(`parsing member status message failed - %v`, err))
			continue
		}
	}
}

func (a *Agent) initMsgListener() {

}

func (a *Agent) notifyAll(topic string, active bool) error {
	byts, err := json.Marshal(messages.MemberStatus{Label: a.myLabel, Active: active, Inv: a.myInv, Topic: topic})
	if err != nil {
		return fmt.Errorf(`marshalling publisher status failed - %v`, err)
	}

	if _, err = a.sktPubState.SendMessage(fmt.Sprintf(`%s%s %s`, topic, domain.PubTopicSuffix, string(byts))); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
	}

	return nil
}

func (a *Agent) parseMembrStatus(msg string) (*messages.MemberStatus, error) {
	frames := strings.Split(msg, " ")
	if len(frames) != 2 {
		return nil, fmt.Errorf(`received a message (%v) with an invalid format - frame count should be 2`, msg)
	}

	var ms messages.MemberStatus
	if err := json.Unmarshal([]byte(frames[1]), &ms); err != nil {
		return nil, fmt.Errorf(`unmarshalling publisher status failed (msg: %s) - %v`, frames[1], err)
	}

	return &ms, nil
}

// dummy validation for PoC
func (a *Agent) validJoiner(label string) bool {
	return true
}

// addSub replaces the key if already exists for the subscriber
func (a *Agent) addSub(topic, sub string, key []byte) {
	a.ts.Lock()
	defer a.ts.Unlock()
	if a.ts.subs[topic] == nil {
		a.ts.subs[topic] = subKey{}
	}
	a.ts.subs[topic][sub] = key
}

func (a *Agent) deleteSub(sm messages.SubscribeMsg) {
	a.ts.Lock()
	defer a.ts.Unlock()
	for _, t := range sm.Topics {
		delete(a.ts.subs[t], sm.Peer)
	}
}
