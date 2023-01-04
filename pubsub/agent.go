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
type keyStore struct {
	*sync.RWMutex
	subs map[string]subKey // can extend to multiple keys per peer
}

type sockets struct {
	sktPub   *zmq.Socket
	sktState *zmq.Socket
	sktMsgs  *zmq.Socket
}

type Agent struct {
	myLabel     string
	pubEndpoint string
	invs        map[string]string // invitation per each topic
	groups      map[string][]models.Member
	probr       services.Agent
	client      services.Client
	km          services.KeyManager
	packer      services.Packer
	ts          *keyStore
	log         log.Logger
	outChan     chan string
	*sockets
}

func NewAgent(zmqCtx *zmq.Context, c *domain.Container) (*Agent, error) {
	// create PUB and SUB sockets for msgs and statuses
	sktPub, err := zmqCtx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	if err = sktPub.Bind(c.Cfg.PubEndpoint); err != nil {
		return nil, fmt.Errorf(`binding zmq pub socket to %s failed - %v`, c.Cfg.PubEndpoint, err)
	}

	sktStates, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for %s topics failed - %v`, domain.PubTopicSuffix, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	a := &Agent{
		myLabel:     c.Cfg.Args.Name,
		pubEndpoint: c.Cfg.PubEndpoint,
		invs:        make(map[string]string),
		groups:      make(map[string][]models.Member),
		probr:       c.Prober,
		client:      c.Client,
		km:          c.KeyManager,
		packer:      c.Packer,
		log:         c.Log,
		outChan:     c.OutChan,
		ts:          &keyStore{RWMutex: &sync.RWMutex{}, subs: map[string]subKey{}},
		sockets: &sockets{
			sktPub:   sktPub,
			sktState: sktStates,
			sktMsgs:  sktMsgs,
		},
	}

	a.init(c.Server)
	return a, nil
}

func (a *Agent) init(s services.Server) {
	// add handler for subscribe messages
	subChan := make(chan models.Message)
	s.AddHandler(domain.MsgTypSubscribe, subChan, true)

	// add handler for group-join requests as sync
	//  - upon request, check if eligible
	//  - if eligible
	//	  - A sends group-info
	joinChan := make(chan models.Message)
	s.AddHandler(domain.MsgTypGroupJoin, joinChan, false)

	go a.joinReqListnr(joinChan)
	go a.subscrptionListnr(subChan)
	go a.statusListnr()
	go a.msgListnr()
}

func (a *Agent) Create(topic string, publisher bool) error {
	inv, err := a.probr.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	m := models.Member{
		Active:      true,
		Publisher:   publisher,
		Label:       a.myLabel,
		Inv:         inv,
		PubEndpoint: a.pubEndpoint,
	}

	a.invs[topic] = inv
	a.groups[topic] = []models.Member{m}

	return nil
}

func (a *Agent) Join(topic, acceptor string, publisher bool) error {
	// check if already joined to the topic
	if _, ok := a.groups[topic]; ok {
		return fmt.Errorf(`already connected to group %s`, topic)
	}

	inv, err := a.probr.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	a.groups[topic] = []models.Member{}
	a.invs[topic] = inv

	// check if B is already connected with acceptor (A)
	// - only needs to check if ever connected (in agent's map), since disconnect is not implemented
	// if not, return
	p, err := a.probr.Peer(acceptor)
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
	byts, err := json.Marshal(messages.ReqGroupJoin{Label: a.myLabel, Topic: topic, RequesterInv: inv})
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

	// publish status
	if err = a.notifyAll(topic, true, publisher); err != nil {
		return fmt.Errorf(`publishing status active failed - %v`, err)
	}

	return nil
}

func (a *Agent) Publish(topic, msg string) error {
	subs, err := a.subsByTopic(topic)
	if err != nil {
		return fmt.Errorf(`fetching subscribers for topic %s failed - %v`, topic, err)
	}

	var published bool
	for sub, key := range subs {
		ownPubKey, err := a.km.PublicKey(sub)
		if err != nil {
			return fmt.Errorf(`getting public key for connection with %s failed - %v`, sub, err)
		}

		ownPrvKey, err := a.km.PrivateKey(sub)
		if err != nil {
			return fmt.Errorf(`getting private key for connection with %s failed - %v`, sub, err)
		}

		encryptdMsg, err := a.packer.Pack([]byte(msg), key, ownPubKey, ownPrvKey)
		if err != nil {
			a.log.Error(err)
			return err
		}

		data, err := json.Marshal(encryptdMsg)
		if err != nil {
			a.log.Error(err)
			return err
		}

		subTopic := topic + `_` + a.myLabel + `_` + sub
		if _, err = a.sktPub.SendMessage(fmt.Sprintf(`%s %s`, subTopic, string(data))); err != nil {
			return fmt.Errorf(`publishing message (%s) failed for %s - %v`, msg, sub, err)
		}

		published = true
		a.log.Trace(fmt.Sprintf(`published %s to %s`, msg, subTopic))
	}

	if published {
		a.outChan <- `Published '` + msg + `' to '` + topic + `'`
	}
	return nil
}

func (a *Agent) connect(m models.Member) error {
	// if not already connected
	// - sets up DIDComm connection via inv
	_, err := a.probr.Peer(m.Label)
	if err != nil {
		if err = a.probr.SyncAccept(m.Inv); err != nil {
			return fmt.Errorf(`accepting group-member invitation failed - %v`, err)
		}
	}

	// B connects to member via SUB for statuses and msgs
	if err = a.sktState.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher state socket failed - %v`, err)
	}

	if m.Publisher {
		if err = a.sktMsgs.Connect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
		}
	}

	return nil
}

// subscribe sets subscriptions via zmq for status topic of the member.
// If the member is a publisher, it proceeds with sending a subscription
// didcomm message and subscribing to message topic via zmq.
func (a *Agent) subscribe(topic string, m models.Member) error {
	if err := a.sktMsgs.SetSubscribe(topic + domain.PubTopicSuffix); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, topic+domain.PubTopicSuffix, err)
	}

	if !m.Publisher {
		return nil
	}

	// get my public key corresponding to this member
	subPublcKey, err := a.km.PublicKey(m.Label)
	if err != nil {
		return fmt.Errorf(`fetching public key for the connection failed - %v`, err)
	}

	// B sends agent subscribe msg to pub
	subTopic := topic + `_` + m.Label + `_` + a.myLabel
	sm := messages.SubscribeMsg{
		Id:        uuid.New().String(),
		Type:      messages.SubscribeV1,
		Subscribe: true,
		Peer:      a.myLabel,
		PubKey:    base58.Encode(subPublcKey),
		Topics:    []string{topic},
		//PubEndpoint: a.myPubEndpnt,
	}

	byts, err := json.Marshal(sm)
	if err != nil {
		return fmt.Errorf(`marshalling subscribe message failed - %v`, err)
	}

	if err = a.probr.SendMessage(domain.MsgTypSubscribe, m.Label, string(byts)); err != nil {
		return fmt.Errorf(`sending subscribe message failed for topic %s - %v`, subTopic, err)
	}

	// B subscribes via zmq
	if err = a.sktMsgs.SetSubscribe(subTopic); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, subTopic, err)
	}

	return nil
}

func (a *Agent) joinReqListnr(joinChan chan models.Message) {
	//  - upon request, check if eligible
	//  - if eligible
	//	  - A sends group-info
	for {
		msg := <-joinChan
		//unpackedMsg, err := a.probr.ReadMessage(msg)
		//if err != nil {
		//	a.log.Error(fmt.Sprintf(`reading group-join request failed - %v`, err))
		//	continue
		//}

		var req messages.ReqGroupJoin
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			a.log.Error(fmt.Sprintf(`unmarshalling group-join request failed - %v`, err))
			continue
		}

		if !a.validJoiner(req.Label) {
			a.log.Error(fmt.Sprintf(`group-join request denied to member (%s)`, req.Label))
			continue
		}

		res := messages.ResGroupJoin{Members: []models.Member{}}
		for _, membr := range a.groups[req.Topic] {
			res.Members = append(res.Members, membr)
		}

		byts, err := json.Marshal(res)
		if err != nil {
			a.log.Error(fmt.Sprintf(`marshalling group-join response failed - %v`, err))
		}

		// a null response is sent if an error occurred
		msg.Reply <- byts
	}
}

func (a *Agent) subscrptionListnr(subChan chan models.Message) {
	// if B is a publisher
	// - if didcomm connection is established
	//   - if sub msg is received
	//     - B connects to sub via SUB for statuses

	for {
		msg := <-subChan
		unpackedMsg, err := a.probr.ReadMessage(msg)
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
	}
}

func (a *Agent) statusListnr() {
	// if status received,
	// - store/update in-memory
	// - if active
	//  - if B is a sub
	//    - if sender is a pub
	//      - if not already connected
	//		  - connect
	// - if not active, remove/disconnect

	for {
		msg, err := a.sktState.Recv(0)
		if err != nil {
			a.log.Error(fmt.Sprintf(`receiving zmq message for member status failed - %v`, err))
			continue
		}

		ms, err := a.parseMembrStatus(msg)
		if err != nil {
			a.log.Error(fmt.Sprintf(`parsing member status message failed - %v`, err))
			continue
		}

		if !ms.Member.Active {
			// todo remove member from group
			continue
		}

		if err = a.connect(ms.Member); err != nil {
			a.log.Error(fmt.Sprintf(`connecting to %s failed in listener - %v`, ms.Member.Label, err))
		}

		if err = a.subscribe(ms.Topic, ms.Member); err != nil {
			a.log.Error(fmt.Sprintf(`subscribing to topic %s with %s failed in listener - %v`, ms.Topic, ms.Member.Label, err))
		}

		// todo add validation to check if already added
		a.groups[ms.Topic] = append(a.groups[ms.Topic], ms.Member)
	}
}

func (a *Agent) msgListnr() {
	for {
		msg, err := a.sktMsgs.Recv(0)
		if err != nil {
			a.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
			continue
		}

		frames := strings.Split(msg, ` `)
		if len(frames) != 2 {
			a.log.Error(fmt.Sprintf(`received an invalid subscribed message (%v) - %v`, frames, err))
			continue
		}

		_, err = a.probr.ReadMessage(models.Message{Type: domain.MsgTypData, Data: []byte(frames[1])})
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading subscribed message failed - %v`, err))
		}
	}
}

func (a *Agent) notifyAll(topic string, active, publisher bool) error {
	byts, err := json.Marshal(messages.Status{
		Member: models.Member{Label: a.myLabel, Active: active, Inv: a.invs[topic], Publisher: publisher},
		Topic:  topic,
	})
	if err != nil {
		return fmt.Errorf(`marshalling publisher status failed - %v`, err)
	}

	if _, err = a.sktPub.SendMessage(fmt.Sprintf(`%s%s %s`, topic, domain.PubTopicSuffix, string(byts))); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
	}

	return nil
}

func (a *Agent) parseMembrStatus(msg string) (*messages.Status, error) {
	frames := strings.Split(msg, " ")
	if len(frames) != 2 {
		return nil, fmt.Errorf(`received a message (%v) with an invalid format - frame count should be 2`, msg)
	}

	var ms messages.Status
	if err := json.Unmarshal([]byte(frames[1]), &ms); err != nil {
		return nil, fmt.Errorf(`unmarshalling publisher status failed (msg: %s) - %v`, frames[1], err)
	}

	return &ms, nil
}

// dummy validation for PoC
func (a *Agent) validJoiner(label string) bool {
	return true
}

func (a *Agent) subsByTopic(topic string) (subKey, error) {
	a.ts.RLock()
	defer a.ts.RUnlock()
	subs, ok := a.ts.subs[topic]
	if !ok {
		return nil, fmt.Errorf(`topic (%s) is not registered`, topic)
	}

	return subs, nil
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
