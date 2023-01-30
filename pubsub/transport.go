package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmqLib "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

type sktType int

const (
	typStateSkt sktType = iota
	typMsgSkt
)

// todo remove group store
type zmq struct {
	pub   *zmqLib.Socket
	state *zmqLib.Socket
	msgs  *zmqLib.Socket
	gs    *groupStore
	log   log.Logger
}

func newZmqTransport(zmqCtx *zmqLib.Context, gs *groupStore, l log.Logger) (*zmq, error) {
	pubSkt, err := zmqCtx.NewSocket(zmqLib.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	sktStates, err := zmqCtx.NewSocket(zmqLib.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for status topic failed - %v`, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmqLib.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	return &zmq{
		pub:   pubSkt,
		state: sktStates,
		msgs:  sktMsgs,
		gs:    gs,
		log:   l,
	}, nil
}

func (z *zmq) start(pubEndpoint string) error {
	if err := z.pub.Bind(pubEndpoint); err != nil {
		return fmt.Errorf(`binding zmq pub socket to %s failed - %v`, pubEndpoint, err)
	}
	return nil
}

func (z *zmq) connect(initRole domain.Role, initLabel, topic string, m models.Member) error {
	var newTopic bool
	if len(z.gs.membrs(topic)) < 2 {
		newTopic = true
	}

	// B connects to member via SUB for statuses and msgs
	if err := z.state.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher (%s) state socket failed - %v`, m.PubEndpoint, err)
	}

	if newTopic {
		if err := z.subscribeStatus(topic); err != nil {
			return fmt.Errorf(`subscribing to status topic of %s failed - %v`, topic, err)
		}
	}

	if m.Publisher {
		if err := z.msgs.Connect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
		}

		// B subscribes via zmq
		var it string
		switch initRole {
		case domain.RolePublisher:
			it = z.internalTopic(topic, initLabel, m.Label)
		case domain.RoleSubscriber:
			it = z.internalTopic(topic, m.Label, initLabel)
		default:
			return fmt.Errorf(`incompatible role (%s) for initiator of the connection`, initRole)
		}

		if err := z.msgs.SetSubscribe(it); err != nil {
			return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, it, err)
		}
	}

	return nil
}

func (z *zmq) publish(topic string, data []byte) error {
	if _, err := z.pub.SendMessage(fmt.Sprintf(`%s %s`, topic, string(data))); err != nil {
		return fmt.Errorf(`publishing message (%s) failed - %v`, string(data), err)
	}
	return nil
}

func (z *zmq) subscribeStatus(topic string) error {
	if err := z.state.SetSubscribe(z.stateTopic(topic)); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, z.stateTopic(topic), err)
	}

	return nil
}

func (z *zmq) unsubscribe(label, topic string) error {
	if err := z.state.SetUnsubscribe(z.stateTopic(topic)); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, z.stateTopic(topic), err)
	}

	for _, m := range z.gs.membrs(topic) {
		if !m.Publisher || m.Label == label {
			continue
		}

		it := z.internalTopic(topic, m.Label, label)
		if err := z.msgs.SetUnsubscribe(it); err != nil {
			return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, it, err)
		}
	}

	return nil
}

// constructs the URN in the format of 'urn:<NID>:<NSS>' (https://www.rfc-editor.org/rfc/rfc2141#section-2)
func (z *zmq) internalTopic(topic, pub, sub string) string {
	return `urn:didcomm-queue:` + topic + `:data:` + pub + `:` + sub
}

func (z *zmq) stateTopic(topic string) string {
	return `urn:didcomm-queue:` + topic + `:state`
}

func (z *zmq) listen(st sktType, handlerFunc func(msg string) error) {
	var skt *zmqLib.Socket
	switch st {
	case typStateSkt:
		skt = z.state
	case typMsgSkt:
		skt = z.msgs
	}

	go func() {
		for {
			msg, err := skt.Recv(0)
			if err != nil {
				z.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
				continue
			}

			if err = handlerFunc(msg); err != nil {
				z.log.Error(fmt.Sprintf(`processing received message failed - %v`, err))
			}
		}
	}()
}
