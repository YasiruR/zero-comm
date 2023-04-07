package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/YasiruR/didcomm-prober/pubsub/transport"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/tryfix/log"
)

// processor implements the handlers functions for incoming messages
type processor struct {
	myLabel     string
	pubEndpoint string
	outChan     chan string
	probr       servicesPkg.Agent
	log         log.Logger
	// sync map may not be required as rarely updated by multiple go-routines
	// and always read only by one go-routine
	ackChans map[string]chan string
	*internals
}

func newProcessor(label, pubEndpoint string, c *container.Container, in *internals) *processor {
	return &processor{
		myLabel:     label,
		pubEndpoint: pubEndpoint,
		probr:       c.Prober,
		internals:   in,
		log:         c.Log,
		outChan:     c.OutChan,
		ackChans:    map[string]chan string{},
	}
}

func (p *processor) joinReqs(msg *models.Message) error {
	_, body, err := p.probr.ReadMessage(*msg)
	if err != nil {
		return fmt.Errorf(`reading group-join authcrypt request failed - %v`, err)
	}

	var req messages.ReqGroupJoin
	if err = json.Unmarshal([]byte(body), &req); err != nil {
		return fmt.Errorf(`unmarshalling group-join request failed - %v`, err)
	}

	if len(p.gs.Membrs(req.Topic)) == 0 {
		return fmt.Errorf(`acceptor is not a member of the requested group (%s)`, req.Topic)
	}

	if !p.validJoiner(req.Label) {
		return fmt.Errorf(`group-join request denied to member (%s)`, req.Label)
	}

	byts, err := json.Marshal(messages.ResGroupJoin{
		Id:   uuid.New().String(),
		Type: messages.JoinResponseV1,
		Params: models.GroupParams{
			OrderEnabled:   p.gs.OrderEnabled(req.Topic),
			JoinConsistent: p.gs.JoinConsistent(req.Topic),
			Mode:           p.gs.Mode(req.Topic),
		},
		Members: p.gs.Membrs(req.Topic),
		//Members: p.addIntruder(req.Topic),
	})
	if err != nil {
		return fmt.Errorf(`marshalling group-join response failed - %v`, err)
	}

	packedMsg, err := p.packr.pack(req.Label, nil, byts)
	if err != nil {
		return fmt.Errorf(`packing group-join response failed - %v`, err)
	}

	// no response is sent if process failed
	msg.Reply <- packedMsg
	p.log.Debug(fmt.Sprintf(`shared group state upon join request by %s`, req.Label), string(byts))

	return nil
}

func (p *processor) subscriptions(msg *models.Message) error {
	_, unpackedMsg, err := p.probr.ReadMessage(*msg)
	if err != nil {
		return fmt.Errorf(`reading subscribe message failed - %v`, err)
	}

	var sm messages.Subscribe
	if err = json.Unmarshal([]byte(unpackedMsg), &sm); err != nil {
		return fmt.Errorf(`unmarshalling subscribe message failed - %v`, err)
	}

	if !sm.Subscribe {
		p.subs.Delete(sm.Topic, sm.Member.Label)
		return nil
	}

	if !p.validJoiner(sm.Member.Label) {
		return fmt.Errorf(`requester (%s) is not eligible`, sm.Member.Label)
	}

	if err = p.sendAuth(sm.Member.Label, sm.Transport.ServrPubKey, sm.Transport.ClientPubKey, sm.Member.Publisher); err != nil {
		return fmt.Errorf(`sending internal auth message failed - %v`, err)
	}

	// send response back to subscriber along with zmq server pub-key of this node
	if err = p.sendSubscribeRes(sm.Topic, sm.Member, msg); err != nil {
		return fmt.Errorf(`sending subscribe response failed - %v`, err)
	}

	if err = p.sendConnect(true, domain.RoleSubscriber, p.myLabel, sm.Topic, sm.Member); err != nil {
		return fmt.Errorf(`sending connect message failed - %v`, err)
	}

	sk := base58.Decode(sm.PubKey)
	p.subs.Add(sm.Topic, sm.Member.Label, sk)
	p.log.Debug(`processed subscription request`, sm)

	return nil
}

func (p *processor) state(_, msg string) error {
	status, err := p.extractStatus(msg)
	if err != nil {
		return fmt.Errorf(`extracting status message failed - %v`, err)
	}

	var validMsg string
	for exchId, encMsg := range status.AuthMsgs {
		_, ok := p.probr.ValidConn(exchId)
		if ok {
			validMsg = encMsg
			break
		}
	}

	if validMsg == `` {
		return fmt.Errorf(`status update is not intended to this member`)
	}

	sender, strAuthMsg, err := p.probr.ReadMessage(models.Message{Type: models.TypGroupStatus, Data: []byte(validMsg)})
	if err != nil {
		return fmt.Errorf(`reading status didcomm message failed - %v`, err)
	}

	// return ack if hello protocol
	if strAuthMsg == domain.HelloPrefix {
		if err = p.probr.SendMessage(models.TypStatusAck, sender, p.pubEndpoint); err != nil {
			return fmt.Errorf(`sending hello ack failed - %v`, err)
		}
		p.log.Debug(fmt.Sprintf(`sent ack to hello protocol of %s`, sender))
		return nil
	}

	var m models.Member
	if err = json.Unmarshal([]byte(strAuthMsg), &m); err != nil {
		return fmt.Errorf(`unmarshalling member message failed - %v`, err)
	}

	if !m.Active {
		if err = p.removeMember(m, status); err != nil {
			return fmt.Errorf(`removing member failed - %v`, err)
		}
		p.outChan <- m.Label + ` left group ` + status.Topic
		return nil
	}

	if err = p.gs.AddMembrs(status.Topic, m); err != nil {
		return fmt.Errorf(`adding member failed - %v`, err)
	}

	p.log.Debug(fmt.Sprintf(`group state updated for member %s in topic %s`, m.Label, status.Topic))
	return nil
}

func (p *processor) data(zmqTopic, msg string) error {
	topic := p.zmq.GroupNameByDataTopic(zmqTopic)
	sender, data, err := p.probr.ReadMessage(models.Message{Type: models.TypGroupMsg, Data: []byte(msg)})
	if err != nil {
		if p.gs.Mode(topic) == domain.SingleQueueMode {
			//p.log.Debug(fmt.Sprintf(`message may not be intended to this member - %v`, err))
			return nil
		}
		return fmt.Errorf(`reading subscribed message failed - %v`, err)
	}

	data, err = p.syncr.parse(topic, data)
	if err != nil {
		return fmt.Errorf(`parsing data message via syncer failed - %v`, err)
	}

	if p.ackChans[sender] != nil {
		p.ackChans[sender] <- data
	}

	p.outChan <- fmt.Sprintf(`%s sent in group '%s': %s`, sender, topic, data)
	return nil
}

func (p *processor) extractStatus(msg string) (*messages.Status, error) {
	out, err := p.compactr.zDecodr.DecodeAll([]byte(msg), nil)
	if err != nil {
		return nil, fmt.Errorf(`decode error - %v`, err)
	}

	var sm messages.Status
	if err = json.Unmarshal(out, &sm); err != nil {
		return nil, fmt.Errorf(`unmarshal error - %v`, err)
	}

	return &sm, nil
}

func (p *processor) sendSubscribeRes(topic string, m models.Member, msg *models.Message) error {
	// to fetch if current node is a publisher of the topic
	curntMembr := p.gs.Membr(topic, p.myLabel)
	if curntMembr == nil {
		return fmt.Errorf(`current member or topic does not exist in group store`)
	}

	resByts, err := json.Marshal(messages.ResSubscribe{
		Transport: messages.Transport{
			ServrPubKey:  p.zmq.ServrPubKey(),
			ClientPubKey: p.zmq.ClientPubKey(),
		},
		Publisher: curntMembr.Publisher,
		Checksum:  p.gs.Checksum(topic),
	})
	if err != nil {
		return fmt.Errorf(`marshalling subscribe response failed - %v`, err)
	}

	packedMsg, err := p.packr.pack(m.Label, nil, resByts)
	if err != nil {
		return fmt.Errorf(`packing subscribe response failed - %v`, err)
	}

	msg.Reply <- packedMsg
	return nil
}

func (p *processor) removeMember(m models.Member, status *messages.Status) error {
	if err := p.sendConnect(false, domain.RoleNull, ``, ``, m); err != nil {
		return fmt.Errorf(`sending connect message failed - %v`, err)
	}

	if m.Publisher {
		if err := p.sendSubscribe(false, false, false, true, status.Topic, p.myLabel, m); err != nil {
			return fmt.Errorf(`sending internal subscribe message failed - %v`, err)
		}
	}

	p.subs.Delete(status.Topic, m.Label)
	if err := p.gs.DeleteMembr(status.Topic, m.Label); err != nil {
		return fmt.Errorf(`deleting member failed - %v`, err)
	}

	if err := p.zmq.RemvKeys(m.Label); err != nil {
		return fmt.Errorf(`removing zmq transport keys failed - %v`, err)
	}

	return nil
}

// dummy validation for PoC
func (p *processor) validJoiner(label string) bool {
	return true
}

/* internal channel functions */

func (p *processor) sendPublish(topic string, data []byte) error {
	rep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	p.zmq.PubChan <- transport.PublishMsg{Topic: topic, Data: data, Reply: rep}
	err := <-rep.Chan
	if err != nil {
		return fmt.Errorf(`zmq publish failed - %v`, err)
	}

	return nil
}

func (p *processor) sendConnect(connect bool, initRole domain.Role, initLabel, topic string, m models.Member) error {
	stateRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	dataRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	p.zmq.ConChan <- transport.ConnectMsg{
		Connect:   connect,
		Initiator: transport.Initiator{Role: initRole, Label: initLabel},
		Topic:     topic,
		Peer:      m,
		Reply: struct {
			State transport.Reply `json:"state"`
			Data  transport.Reply `json:"data"`
		}{State: stateRep, Data: dataRep},
	}

	err := <-stateRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq state connection failed - %v`, err)
	}

	err = <-dataRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq data connection failed - %v`, err)
	}

	return nil
}

func (p *processor) sendSubscribe(subscribe, unsubAll, state, data bool, topic, label string, m models.Member) error {
	stateRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	dataRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	p.zmq.SubChan <- transport.SubscribeMsg{
		Subscribe: subscribe,
		UnsubAll:  unsubAll,
		MyLabel:   label,
		Topic:     topic,
		State:     state,
		Data:      data,
		Peer:      m,
		Reply: struct {
			State transport.Reply `json:"state"`
			Data  transport.Reply `json:"data"`
		}{State: stateRep, Data: dataRep},
	}

	err := <-stateRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq state subscribe failed - %v`, err)
	}

	err = <-dataRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq data subscribe failed - %v`, err)
	}

	return nil
}

func (p *processor) sendAuth(label, srvrPubK, clientPubK string, publisher bool) error {
	var dataAuth bool
	if publisher {
		dataAuth = true
	}

	authStateRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	authDataRep := transport.Reply{Id: uuid.New().String(), Chan: make(chan error)}
	p.zmq.AuthChan <- transport.AuthMsg{
		Label:        label,
		ServrPubKey:  srvrPubK,
		ClientPubKey: clientPubK,
		Data:         dataAuth,
		Reply: struct {
			State transport.Reply `json:"state"`
			Data  transport.Reply `json:"data"`
		}{State: authStateRep, Data: authDataRep},
	}

	err := <-authStateRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq state authenticate failed - %v`, err)
	}

	err = <-authDataRep.Chan
	if err != nil {
		return fmt.Errorf(`zmq data authenticate failed - %v`, err)
	}

	return nil
}

func (p *processor) acker(register bool, label string, ackr chan string) {
	if register {
		p.ackChans[label] = ackr
		return
	}
	delete(p.ackChans, label)
}

//func (p *processor) addIntruder(topic string) []models.Member {
//	return append(p.gs.Membrs(topic),
//		models.Member{
//			Active:      true,
//			Publisher:   true,
//			Label:       "zack",
//			Inv:         "tcp://127.0.1.1:?oob=eyJpZCI6ImEwNWZlY2M1LTI0MWItNDcxMy1iMGIwLTM4MjllNDQxNDgxNyIsInR5cGUiOiJodHRwczovL2RpZGNvbW0ub3JnL291dC1vZi1iYW5kLzEuMC9pbnZpdGF0aW9uIiwiZnJvbSI6IiIsImxhYmVsIjoiemFjayIsImJvZHkiOnsiZ29hbF9jb2RlIjoiIiwiZ29hbCI6IiIsImFjY2VwdCI6bnVsbH0sIkF0dGFjaG1lbnRzIjpudWxsLCJzZXJ2aWNlcyI6W3siaWQiOiJjNTM5ODkzYS04ODBlLTQ3NzUtOTBmZS00MzVhZWZkMmNhNjYiLCJ0eXBlIjoiZGlkLWV4Y2hhbmdlLXNlcnZpY2UiLCJyZWNpcGllbnRLZXlzIjpbIlBkbnpGa2VjbysvcUxic0Nrd3JOQ2tJRlJoSnRSK3RmaVYzWGl5RFViblU9Il0sInJvdXRpbmdLZXlzIjpudWxsLCJzZXJ2aWNlRW5kcG9pbnQiOiJ0Y3A6Ly8xMjcuMC4xLjE6OTA5MCIsImFjY2VwdCI6bnVsbH1dfQ==",
//			PubEndpoint: "tcp://127.0.1.1:9091",
//		},
//	)
//}
