package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmqPkg "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

type sktType int

const (
	typStateSkt sktType = iota
	typMsgSkt
)

type zmq struct {
	singlQ bool
	pub    *zmqPkg.Socket
	state  *zmqPkg.Socket
	msgs   *zmqPkg.Socket
	gs     *groupStore
	log    log.Logger
}

func newZmqTransport(zmqCtx *zmqPkg.Context, gs *groupStore, c *domain.Container) (*zmq, error) {
	pubSkt, err := zmqCtx.NewSocket(zmqPkg.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	sktStates, err := zmqCtx.NewSocket(zmqPkg.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for status topic failed - %v`, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmqPkg.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	return &zmq{
		singlQ: c.Cfg.SingleQ,
		pub:    pubSkt,
		state:  sktStates,
		msgs:   sktMsgs,
		gs:     gs,
		log:    c.Log,
	}, nil
}

func (z *zmq) start(pubEndpoint string) error {
	if err := z.pub.Bind(pubEndpoint); err != nil {
		return fmt.Errorf(`binding zmq pub socket to %s failed - %v`, pubEndpoint, err)
	}
	return nil
}

func (z *zmq) connect(initRole domain.Role, initLabel, topic string, m models.Member) error {
	// B connects to member via SUB for statuses and msgs
	if err := z.state.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher (%s) state socket failed - %v`, m.PubEndpoint, err)
	}

	if err := z.subscribeStatus(topic); err != nil {
		return fmt.Errorf(`subscribing to status topic of %s failed - %v`, topic, err)
	}

	if m.Publisher {
		if err := z.msgs.Connect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
		}

		var pub, sub string
		switch initRole {
		case domain.RolePublisher:
			pub, sub = initLabel, m.Label
		case domain.RoleSubscriber:
			pub, sub = m.Label, initLabel
		default:
			return fmt.Errorf(`incompatible role (%v) for initiator of the connection`, initRole)
		}

		if err := z.subscribeData(topic, pub, sub); err != nil {
			return fmt.Errorf(`subscribing data failed - %v`, err)
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
	if z.topicExists(topic) {
		z.log.Debug(fmt.Sprintf(`subscription already exists with topic %s for states`, topic))
		return nil
	}

	if err := z.state.SetSubscribe(z.stateTopic(topic)); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, z.stateTopic(topic), err)
	}

	return nil
}

// subscribeData may subscribe the same topic multiple times in the single
// queue mode but all identical subscriptions will be unsubscribed upon leaving
func (z *zmq) subscribeData(topic, pub, sub string) error {
	dt := z.dataTopic(topic, pub, sub)
	if err := z.msgs.SetSubscribe(dt); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, dt, err)
	}

	return nil
}

func (z *zmq) unsubscribeAll(label, topic string) error {
	if err := z.state.SetUnsubscribe(z.stateTopic(topic)); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, z.stateTopic(topic), err)
	}

	for _, m := range z.gs.membrs(topic) {
		if !m.Publisher || m.Label == label {
			continue
		}

		dt := z.dataTopic(topic, m.Label, label)
		if err := z.msgs.SetUnsubscribe(dt); err != nil {
			return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, dt, err)
		}
	}

	return nil
}

func (z *zmq) unsubscribeData(label, topic, peer string) error {
	dt := z.dataTopic(topic, peer, label)
	if err := z.msgs.SetUnsubscribe(dt); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, dt, err)
	}

	return nil
}

func (z *zmq) topicExists(topic string) bool {
	if len(z.gs.membrs(topic)) < 2 {
		return false
	}
	return true
}

// constructs the URN in the format of 'urn:<NID>:<NSS>' (https://www.rfc-editor.org/rfc/rfc2141#section-2)
func (z *zmq) dataTopic(topic, pub, sub string) string {
	if z.singlQ {
		return `urn:didcomm-queue:` + topic + `:data`
	}
	return `urn:didcomm-queue:` + topic + `:data:` + pub + `:` + sub
}

func (z *zmq) stateTopic(topic string) string {
	return `urn:didcomm-queue:` + topic + `:state`
}

func (z *zmq) listen(st sktType, handlerFunc func(msg string) error) {
	var skt *zmqPkg.Socket
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
				if z.singlQ && st == typMsgSkt {
					z.log.Debug(fmt.Sprintf(`message may not be intended to this member - %v`, err))
					continue
				}
				z.log.Error(fmt.Sprintf(`processing received message failed - %v`, err))
			}
		}
	}()
}

func (z *zmq) close() error {
	if err := z.pub.Close(); err != nil {
		return fmt.Errorf(`closing publisher socket failed - %v`, err)
	}

	if err := z.state.Close(); err != nil {
		return fmt.Errorf(`closing state socket failed - %v`, err)
	}

	if err := z.msgs.Close(); err != nil {
		return fmt.Errorf(`closing data socket failed - %v`, err)
	}

	return nil
}
