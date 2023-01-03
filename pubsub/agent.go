package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	zmq "github.com/pebbe/zmq4"
)

type Agent struct {
	myLabel     string
	myInv       string
	groups      map[string][]models.Member
	prb         services.Agent
	client      services.Client
	sktPubState *zmq.Socket
	sktSubState *zmq.Socket
	sktPubMsgs  *zmq.Socket
	sktSubMsgs  *zmq.Socket
}

func (a *Agent) Init() {
	// add handler for group-join requests
	//  - upon request, check if eligible
	//  - if eligible
	//	  - A sends group-info
	// create PUB and SUB sockets for msgs and statuses
}

func (a *Agent) Join(topic, acceptor string) error {
	// check if already joined to the topic
	if _, ok := a.groups[topic]; ok {
		return fmt.Errorf(`already connected to group %s`, topic)
	}

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
		if err = a.connect(m); err != nil {
			return fmt.Errorf(`connecting to %s failed - %v`, m.Label, err)
		}

		// wait till connection is done via conn channel

		// subscribe
	}

	// store acceptor
}

func (a *Agent) connect(m models.Member) error {
	// if not already connected
	// - sets up DIDComm connection via inv
	p, err := a.prb.Peer(m.Label)
	if err != nil {
		if _, err = a.prb.Accept(m.Inv); err != nil {
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
	// B sends agent subscribe msg to pub

	// B subscribes via zmq
}

func (a *Agent) initConnListener() {
	// if B is a publisher
	// - if didcomm connection is established
	//   - if sub msg is received
	//     - B connects to sub via SUB for statuses
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
}

func (a *Agent) initMsgListener() {

}

// dummy validation for PoC
func (a *Agent) validJoiner() bool {
	return true
}
