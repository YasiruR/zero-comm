package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	zmqPkg "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

// processor implements the handlers functions for incoming messages
type processor struct {
	myLabel string
	probr   servicesPkg.Agent
	log     log.Logger
	outChan chan string
	*internals
}

func newProcessor(label string, c *domain.Container, in *internals) *processor {
	return &processor{
		myLabel:   label,
		probr:     c.Prober,
		internals: in,
		log:       c.Log,
		outChan:   c.OutChan,
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

	if len(p.gs.membrs(req.Topic)) == 0 {
		return fmt.Errorf(`acceptor is not a member of the requested group (%s)`, req.Topic)
	}

	if !p.validJoiner(req.Label) {
		return fmt.Errorf(`group-join request denied to member (%s)`, req.Label)
	}

	byts, err := json.Marshal(messages.ResGroupJoin{
		Id:      uuid.New().String(),
		Type:    messages.JoinResponseV1,
		Members: p.gs.membrs(req.Topic),
		//Members: p.addIntruder(req.Topic),
	})
	if err != nil {
		return fmt.Errorf(`marshalling group-join response failed - %v`, err)
	}

	packedMsg, err := p.packr.pack(false, req.Label, nil, byts)
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
		p.subs.delete(sm.Topic, sm.Member.Label)
		return nil
	}

	if !p.validJoiner(sm.Member.Label) {
		return fmt.Errorf(`requester (%s) is not eligible`, sm.Member.Label)
	}

	var sktMsgs *zmqPkg.Socket = nil
	if sm.Member.Publisher {
		sktMsgs = p.zmq.msgs
	}

	if err = p.auth.setPeerAuthn(sm.Member.Label, sm.Transport.ServrPubKey, sm.Transport.ClientPubKey, p.zmq.state, sktMsgs); err != nil {
		return fmt.Errorf(`setting zmq transport authentication failed - %v`, err)
	}

	// send response back to subscriber along with zmq server pub-key of this node
	if err = p.sendSubscribeRes(sm.Topic, sm.Member, msg); err != nil {
		return fmt.Errorf(`sending subscribe response failed - %v`, err)
	}

	if err = p.zmq.connect(domain.RoleSubscriber, p.myLabel, sm.Topic, sm.Member); err != nil {
		return fmt.Errorf(`zmq connection failed - %v`, err)
	}

	sk := base58.Decode(sm.PubKey)
	p.subs.add(sm.Topic, sm.Member.Label, sk)
	p.log.Debug(`processed subscription request`, sm)

	return nil
}

func (p *processor) states(_, msg string) error {
	sm, err := p.extractStatus(msg)
	if err != nil {
		return fmt.Errorf(`extracting status message failed - %v`, err)
	}

	var validMsg string
	for exchId, encMsg := range sm.AuthMsgs {
		ok, _ := p.probr.ValidConn(exchId)
		if ok {
			validMsg = encMsg
			break
		}
	}

	if validMsg == `` {
		return fmt.Errorf(`status update is not intended to this member`)
	}

	defer func(topic string) {
		if err = p.valdtr.updateHash(topic, p.gs.membrs(topic)); err != nil {
			p.log.Error(fmt.Sprintf(`updating checksum for the group failed - %v`, err))
		}
	}(sm.Topic)

	_, strAuthMsg, err := p.probr.ReadMessage(models.Message{Type: models.TypGroupStatus, Data: []byte(validMsg)})
	if err != nil {
		return fmt.Errorf(`reading status didcomm message failed - %v`, err)
	}

	var member models.Member
	if err = json.Unmarshal([]byte(strAuthMsg), &member); err != nil {
		return fmt.Errorf(`unmarshalling member message failed - %v`, err)
	}

	if !member.Active {
		if member.Publisher {
			if err = p.zmq.unsubscribeData(p.myLabel, sm.Topic, member.Label); err != nil {
				return fmt.Errorf(`unsubscribing data topic failed - %v`, err)
			}
		}

		p.subs.delete(sm.Topic, member.Label)
		p.gs.deleteMembr(sm.Topic, member.Label)
		if err = p.auth.remvKeys(member.Label); err != nil {
			return fmt.Errorf(`removing zmq transport keys failed - %v`, err)
		}
		p.outChan <- member.Label + ` left group ` + sm.Topic
		return nil
	}

	p.gs.addMembr(sm.Topic, member)
	p.log.Debug(fmt.Sprintf(`group state updated for member %s in topic %s`, member.Label, sm.Topic))
	return nil
}

func (p *processor) data(topic, msg string) error {
	sender, data, err := p.probr.ReadMessage(models.Message{Type: models.TypGroupMsg, Data: []byte(msg)})
	if err != nil {
		return fmt.Errorf(`reading subscribed message failed - %v`, err)
	}

	if p.syncr != nil {
		data, err = p.syncr.parse(data)
		if err != nil {
			return fmt.Errorf(`parsing ordered data message failed - %v`, err)
		}
	}

	p.outChan <- fmt.Sprintf(`%s sent in group '%s': %s`, sender, p.zmq.groupNameByDataTopic(topic), data)
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
	curntMembr := p.gs.membr(topic, p.myLabel)
	if curntMembr == nil {
		return fmt.Errorf(`current member or topic does not exist in group store`)
	}

	checksum, err := p.valdtr.hash(topic)
	if err != nil {
		return fmt.Errorf(`fetching group checksum failed - %v`, err)
	}

	resByts, err := json.Marshal(messages.ResSubscribe{
		Transport: messages.Transport{
			ServrPubKey:  p.auth.servr.pub,
			ClientPubKey: p.auth.client.pub,
		},
		Publisher: curntMembr.Publisher,
		Checksum:  checksum,
	})
	if err != nil {
		return fmt.Errorf(`marshalling subscribe response failed - %v`, err)
	}

	packedMsg, err := p.packr.pack(false, m.Label, nil, resByts)
	if err != nil {
		return fmt.Errorf(`packing subscribe response failed - %v`, err)
	}

	msg.Reply <- packedMsg
	return nil
}

// dummy validation for PoC
func (p *processor) validJoiner(label string) bool {
	return true
}

//func (p *processor) addIntruder(topic string) []models.Member {
//	return append(p.gs.membrs(topic),
//		models.Member{
//			Active:      true,
//			Publisher:   true,
//			Label:       "zack",
//			Inv:         "tcp://127.0.1.1:?oob=eyJpZCI6ImEwNWZlY2M1LTI0MWItNDcxMy1iMGIwLTM4MjllNDQxNDgxNyIsInR5cGUiOiJodHRwczovL2RpZGNvbW0ub3JnL291dC1vZi1iYW5kLzEuMC9pbnZpdGF0aW9uIiwiZnJvbSI6IiIsImxhYmVsIjoiemFjayIsImJvZHkiOnsiZ29hbF9jb2RlIjoiIiwiZ29hbCI6IiIsImFjY2VwdCI6bnVsbH0sIkF0dGFjaG1lbnRzIjpudWxsLCJzZXJ2aWNlcyI6W3siaWQiOiJjNTM5ODkzYS04ODBlLTQ3NzUtOTBmZS00MzVhZWZkMmNhNjYiLCJ0eXBlIjoiZGlkLWV4Y2hhbmdlLXNlcnZpY2UiLCJyZWNpcGllbnRLZXlzIjpbIlBkbnpGa2VjbysvcUxic0Nrd3JOQ2tJRlJoSnRSK3RmaVYzWGl5RFViblU9Il0sInJvdXRpbmdLZXlzIjpudWxsLCJzZXJ2aWNlRW5kcG9pbnQiOiJ0Y3A6Ly8xMjcuMC4xLjE6OTA5MCIsImFjY2VwdCI6bnVsbH1dfQ==",
//			PubEndpoint: "tcp://127.0.1.1:9091",
//		},
//	)
//}
