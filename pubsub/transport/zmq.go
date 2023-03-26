package transport

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/pubsub/stores"
	zmqPkg "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"strings"
	"sync"
)

type endpoints struct {
	state string
	data  string
}

type Zmq struct {
	ConChan    chan ConnectMsg
	PubChan    chan PublishMsg
	SubChan    chan SubscribeMsg
	AuthChan   chan AuthMsg
	ReplyChans map[string]chan error
	ReplyLock  *sync.RWMutex
	gs         *stores.Group
	log        log.Logger
	endpts     endpoints
	*auth
}

func NewZmqTransport(zmqCtx *zmqPkg.Context, gs *stores.Group, c *container.Container, stateFunc, dataFunc func(topic, msg string) error) (*Zmq, error) {
	authn, err := authenticator(c.Cfg.Name, c.Cfg.Verbose)
	if err != nil {
		return nil, fmt.Errorf(`initializing zmq authenticator failed - %v`, err)
	}

	z := &Zmq{
		ConChan:    make(chan ConnectMsg),
		PubChan:    make(chan PublishMsg),
		SubChan:    make(chan SubscribeMsg),
		AuthChan:   make(chan AuthMsg),
		ReplyChans: map[string]chan error{},
		ReplyLock:  &sync.RWMutex{},
		gs:         gs,
		auth:       authn,
		log:        c.Log,
		endpts:     endpoints{state: `ipc://state-` + c.Cfg.Name + `.ipc`, data: `ipc://data-` + c.Cfg.Name + `.ipc`},
	}

	go z.initConnector(zmqCtx)
	go z.publisher(zmqCtx, c.Cfg.PubEndpoint)
	go z.listenState(zmqCtx, stateFunc)
	go z.listenData(zmqCtx, dataFunc)

	return z, nil
}

func (z *Zmq) initConnector(zmqCtx *zmqPkg.Context) {
	intrnlPub, err := zmqCtx.NewSocket(zmqPkg.PUB)
	if err != nil {
		z.log.Fatal(fmt.Sprintf(`creating sub consumer socket failed - %v`, err))
	}

	if err = intrnlPub.Bind(z.endpts.state); err != nil {
		z.log.Fatal(fmt.Sprintf(`binding to internal state endpoint failed - %v`, err))
	}

	if err = intrnlPub.Bind(z.endpts.data); err != nil {
		z.log.Fatal(fmt.Sprintf(`binding to internal data endpoint failed - %v`, err))
	}

	for {
		select {
		case cm := <-z.ConChan:
			z.addRepChan(cm.Reply.State.Id, cm.Reply.Data.Id, cm.Reply.State.Chan, cm.Reply.Data.Chan)
			data, err := json.Marshal(cm)
			if err != nil {
				log.Error(fmt.Sprintf(`marshalling internal connect message failed - %v`, err))
				continue
			}

			if _, err = intrnlPub.SendMessage(fmt.Sprintf(`%s %v`, topicConnect, string(data))); err != nil {
				log.Error(fmt.Sprintf(`publishing message (%s) failed - %v`, string(data), err))
			}
		case sm := <-z.SubChan:
			z.addRepChan(sm.Reply.State.Id, sm.Reply.Data.Id, sm.Reply.State.Chan, sm.Reply.Data.Chan)
			data, err := json.Marshal(sm)
			if err != nil {
				log.Error(fmt.Sprintf(`marshalling internal subscribe message failed - %v`, err))
				continue
			}

			if _, err = intrnlPub.SendMessage(fmt.Sprintf(`%s %v`, topicSubscribe, string(data))); err != nil {
				log.Error(fmt.Sprintf(`publishing message (%s) failed - %v`, string(data), err))
			}
		case am := <-z.AuthChan:
			z.addRepChan(am.Reply.State.Id, am.Reply.Data.Id, am.Reply.State.Chan, am.Reply.Data.Chan)
			data, err := json.Marshal(am)
			if err != nil {
				log.Error(fmt.Sprintf(`marshalling internal authenticate message failed - %v`, err))
				continue
			}

			if _, err = intrnlPub.SendMessage(fmt.Sprintf(`%s %v`, topicAuthenticate, string(data))); err != nil {
				log.Error(fmt.Sprintf(`publishing message (%s) failed - %v`, string(data), err))
			}
		}
	}
}

func (z *Zmq) listenState(zmqCtx *zmqPkg.Context, handlerFunc func(topic, msg string) error) {
	sktState, err := zmqCtx.NewSocket(zmqPkg.SUB)
	if err != nil {
		log.Fatal(fmt.Sprintf(`creating sub socket for status topic failed - %v`, err))
	}

	if err = sktState.Connect(z.endpts.state); err != nil {
		log.Fatal(fmt.Sprintf(`connecting to internal data endpoint failed - %v`, err))
	}

	for _, t := range internalTopics {
		if err = sktState.SetSubscribe(t); err != nil {
			log.Fatal(fmt.Sprintf(`subscribing internal topic (%s) by state socket failed - %v`, t, err))
		}
	}

	for {
		msg, err := sktState.Recv(0)
		if err != nil {
			z.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
			continue
		}

		frames := strings.SplitN(msg, ` `, 2)
		if len(frames) != 2 {
			z.log.Error(fmt.Sprintf(`received an invalid status message (length=%v)`, len(frames)))
			continue
		}
		topic, data := frames[0], frames[1]

		if topic == topicConnect {
			var cm ConnectMsg
			if err = json.Unmarshal([]byte(data), &cm); err != nil {
				z.replyErr(cm.Reply.State.Id, fmt.Errorf(`unmarshalling internal connect state message failed - %v`, err))
				continue
			}

			if !cm.Connect {
				if err = z.disconnectStatus(cm.Peer.PubEndpoint, sktState); err != nil {
					z.replyErr(cm.Reply.State.Id, fmt.Errorf(`disconnecting state socket failed - %v`, err))
					continue
				}

				z.replyErr(cm.Reply.State.Id, nil)
				continue
			}

			if err = z.connectState(cm, sktState); err != nil {
				z.replyErr(cm.Reply.State.Id, fmt.Errorf(`connecting to state socket failed - %v`, err))
				continue
			}

			z.replyErr(cm.Reply.State.Id, nil)
			continue
		}

		if topic == topicSubscribe {
			var sm SubscribeMsg
			if err = json.Unmarshal([]byte(data), &sm); err != nil {
				z.replyErr(sm.Reply.State.Id, fmt.Errorf(`unmarshalling internal state subscribe message failed - %v`, err))
				continue
			}

			if !sm.State {
				z.replyErr(sm.Reply.State.Id, nil)
				continue
			}

			if !sm.Subscribe {
				if err = z.unsubscribeState(sm.MyLabel, sm.Topic, sktState); err != nil {
					z.replyErr(sm.Reply.State.Id, fmt.Errorf(`unsubscribing state failed - %v`, err))
					continue
				}
			}

			z.replyErr(sm.Reply.State.Id, nil)
			continue
		}

		if topic == topicAuthenticate {
			var am AuthMsg
			if err = json.Unmarshal([]byte(data), &am); err != nil {
				z.replyErr(am.Reply.State.Id, fmt.Errorf(`unmarshalling internal state authenticate message failed - %v`, err))
				continue
			}

			if err = z.setPeerStateAuthn(am.Label, am.ServrPubKey, am.ClientPubKey, sktState); err != nil {
				z.replyErr(am.Reply.State.Id, fmt.Errorf(`setting state authentication failed - %v`, err))
				continue
			}

			z.replyErr(am.Reply.State.Id, nil)
			continue
		}

		go func(topic, data string) {
			if err = handlerFunc(topic, data); err != nil {
				z.log.Error(fmt.Sprintf(`processing received message failed - %v`, err))
			}
		}(topic, data)
	}
}

func (z *Zmq) listenData(zmqCtx *zmqPkg.Context, handlerFunc func(topic, msg string) error) {
	sktData, err := zmqCtx.NewSocket(zmqPkg.SUB)
	if err != nil {
		log.Fatal(fmt.Sprintf(`creating sub socket for data topics failed - %v`, err))
	}

	if err = sktData.Connect(z.endpts.data); err != nil {
		log.Fatal(fmt.Sprintf(`connecting to internal data endpoint failed - %v`, err))
	}

	for _, t := range internalTopics {
		if err = sktData.SetSubscribe(t); err != nil {
			log.Fatal(fmt.Sprintf(`subscribing internal topic (%s) by data socket failed - %v`, t, err))
		}
	}

	for {
		msg, err := sktData.Recv(0)
		if err != nil {
			z.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
			continue
		}

		frames := strings.SplitN(msg, ` `, 2)
		if len(frames) != 2 {
			z.log.Error(fmt.Sprintf(`received an invalid status message (length=%v)`, len(frames)))
			continue
		}
		topic, data := frames[0], frames[1]

		if topic == topicConnect {
			var cm ConnectMsg
			if err = json.Unmarshal([]byte(data), &cm); err != nil {
				z.replyErr(cm.Reply.Data.Id, fmt.Errorf(`unmarshalling internal connect data message failed - %v`, err))
				continue
			}

			if cm.Connect {
				if err = z.connectData(cm, sktData); err != nil {
					z.replyErr(cm.Reply.Data.Id, fmt.Errorf(`connecting to data socket failed - %v`, err))
					continue
				}
			}

			z.replyErr(cm.Reply.Data.Id, nil)
			continue
		}

		if topic == topicSubscribe {
			var sm SubscribeMsg
			if err = json.Unmarshal([]byte(data), &sm); err != nil {
				z.replyErr(sm.Reply.Data.Id, fmt.Errorf(`unmarshalling internal subscribe message failed - %v`, err))
				continue
			}

			if sm.Subscribe == false {
				if sm.UnsubAll == true && sm.Data == true {
					if err = z.unsubscribeAllData(sm.MyLabel, sm.Topic, sktData); err != nil {
						z.replyErr(sm.Reply.Data.Id, fmt.Errorf(`unsubscribing all data topics failed - %v`, err))
						continue
					}

					z.replyErr(sm.Reply.Data.Id, nil)
					continue
				}

				if sm.Data == true {
					// todo pass only sm
					if err = z.unsubscribeData(sm.MyLabel, sm.Topic, sm.Peer, sktData); err != nil {
						z.replyErr(sm.Reply.Data.Id, fmt.Errorf(`unsubscribing data topic failed - %v`, err))
						continue
					}
				}
			}

			z.replyErr(sm.Reply.Data.Id, nil)
			continue
		}

		if topic == topicAuthenticate {
			var am AuthMsg
			if err = json.Unmarshal([]byte(data), &am); err != nil {
				z.replyErr(am.Reply.Data.Id, fmt.Errorf(`unmarshalling internal data authenticate message failed - %v`, err))
				continue
			}

			if !am.Data {
				z.replyErr(am.Reply.Data.Id, nil)
				continue
			}

			if err = z.setPeerDataAuthn(am.ServrPubKey, sktData); err != nil {
				z.replyErr(am.Reply.Data.Id, fmt.Errorf(`setting data authentication failed - %v`, err))
				continue
			}

			z.replyErr(am.Reply.Data.Id, nil)
			continue
		}

		go func(topic, data string) {
			if err = handlerFunc(topic, data); err != nil {
				z.log.Error(fmt.Sprintf(`processing received message failed - %v`, err))
			}
		}(topic, data)
	}
}

func (z *Zmq) connectState(cm ConnectMsg, sktState *zmqPkg.Socket) error {
	// B connects to member via SUB for statuses and msgs
	if err := sktState.Connect(cm.Peer.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher (%s) state socket failed - %v`, cm.Peer.PubEndpoint, err)
	}

	if err := z.subscribeStatus(cm.Topic, sktState); err != nil {
		return fmt.Errorf(`subscribing to status topic of %s failed - %v`, cm.Topic, err)
	}

	return nil
}

func (z *Zmq) connectData(cm ConnectMsg, sktData *zmqPkg.Socket) error {
	if cm.Peer.Publisher {
		if err := sktData.Connect(cm.Peer.PubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
		}

		var pub, sub string
		switch cm.Initiator.Role {
		case domain.RolePublisher:
			pub, sub = cm.Initiator.Label, cm.Peer.Label
		case domain.RoleSubscriber:
			pub, sub = cm.Peer.Label, cm.Initiator.Label
		default:
			return fmt.Errorf(`incompatible role (%v) for initiator of the connection`, cm.Initiator.Role)
		}

		if err := z.subscribeData(cm.Topic, pub, sub, sktData); err != nil {
			return fmt.Errorf(`subscribing data failed - %v`, err)
		}
	}

	return nil
}

func (z *Zmq) publisher(zmqCtx *zmqPkg.Context, externalEndpoint string) {
	sktPub, err := zmqCtx.NewSocket(zmqPkg.PUB)
	if err != nil {
		z.log.Fatal(fmt.Sprintf(`creating zmq pub socket failed - %v`, err))
	}

	if err = z.auth.setPubAuthn(sktPub); err != nil {
		z.log.Fatal(fmt.Sprintf(`setting authentication on pub socket failed - %v`, err))
	}

	if err = sktPub.Bind(externalEndpoint); err != nil {
		z.log.Fatal(fmt.Sprintf(`binding zmq pub socket to %s failed - %v`, externalEndpoint, err))
	}

	for {
		pm := <-z.PubChan
		if _, err = sktPub.SendMessage(fmt.Sprintf(`%s %s`, pm.Topic, string(pm.Data))); err != nil {
			pm.Reply.Chan <- fmt.Errorf(`publishing message (%s) failed - %v`, string(pm.Data), err)
			continue
		}
		pm.Reply.Chan <- nil
	}
}

func (z *Zmq) subscribeStatus(topic string, sktState *zmqPkg.Socket) error {
	if z.topicExists(topic) {
		z.log.Debug(fmt.Sprintf(`subscription already exists with topic %s for states`, topic))
		return nil
	}

	if err := sktState.SetSubscribe(z.StateTopic(topic)); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, z.StateTopic(topic), err)
	}

	return nil
}

// subscribeData may subscribe the same topic multiple times in the single
// queue mode but all identical subscriptions will be unsubscribed upon leaving
func (z *Zmq) subscribeData(topic, pub, sub string, sktData *zmqPkg.Socket) error {
	dt := z.DataTopic(topic, pub, sub)
	if err := sktData.SetSubscribe(dt); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, dt, err)
	}

	return nil
}

func (z *Zmq) unsubscribeAllData(label, topic string, sktData *zmqPkg.Socket) error {
	for _, m := range z.gs.Membrs(topic) {
		if m.Label == label {
			continue
		}

		if !m.Publisher {
			continue
		}

		dt := z.DataTopic(topic, m.Label, label)
		if err := sktData.SetUnsubscribe(dt); err != nil {
			return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, dt, err)
		}

		if err := sktData.Disconnect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`disconnecting data endpoint (%s) failed - %v`, m.PubEndpoint, err)
		}
	}

	return nil
}

func (z *Zmq) unsubscribeState(label, topic string, sktState *zmqPkg.Socket) error {
	if err := sktState.SetUnsubscribe(z.StateTopic(topic)); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, z.StateTopic(topic), err)
	}

	for _, m := range z.gs.Membrs(topic) {
		if m.Label == label {
			continue
		}

		if err := sktState.Disconnect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`disconnecting state endpoint (%s) failed - %v`, m.PubEndpoint, err)
		}
	}

	return nil
}

// unsubscribeData includes disconnecting status socket since the function
// is called only when removing an inactive member
func (z *Zmq) unsubscribeData(label, topic string, m models.Member, sktData *zmqPkg.Socket) error {
	dt := z.DataTopic(topic, m.Label, label)
	if err := sktData.SetUnsubscribe(dt); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, dt, err)
	}

	if err := sktData.Disconnect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`disconnecting data endpoint failed - %v`, err)
	}

	return nil
}

func (z *Zmq) disconnectStatus(endpoint string, sktState *zmqPkg.Socket) error {
	if err := sktState.Disconnect(endpoint); err != nil {
		return fmt.Errorf(`disconnecting state endpoint failed - %v`, err)
	}
	return nil
}

func (z *Zmq) topicExists(topic string) bool {
	if len(z.gs.Membrs(topic)) < 2 {
		return false
	}
	return true
}

// DataTopic constructs the URN in the format of 'urn:<NID>:<NSS>' (https://www.rfc-editor.org/rfc/rfc2141#section-2)
func (z *Zmq) DataTopic(topic, pub, sub string) string {
	if z.gs.Mode(topic) == domain.SingleQueueMode {
		return domain.TopicPrefix + topic + `:data`
	}
	return domain.TopicPrefix + topic + `:data:` + pub + `:` + sub
}

func (z *Zmq) StateTopic(topic string) string {
	return domain.TopicPrefix + topic + `:state`
}

func (z *Zmq) GroupNameByDataTopic(topic string) string {
	parts := strings.Split(topic, `:`)
	if len(parts) < 3 {
		z.log.Warn(`group name could not be fetched from topic`, topic)
		return ``
	}

	return parts[2]
}

func (z *Zmq) addRepChan(stateId, dataId string, stateChan, dataChan chan error) {
	z.ReplyLock.Lock()
	defer z.ReplyLock.Unlock()
	if stateId != `` {
		z.ReplyChans[stateId] = stateChan
	}

	if dataId != `` {
		z.ReplyChans[dataId] = dataChan
	}
}

func (z *Zmq) replyErr(id string, err error) {
	z.ReplyLock.RLock()
	c := z.ReplyChans[id]
	z.ReplyLock.RUnlock()
	c <- err
}

func (z *Zmq) close(pub, state, msgs *zmqPkg.Socket) error {
	if err := pub.Close(); err != nil {
		return fmt.Errorf(`closing publisher socket failed - %v`, err)
	}

	if err := state.Close(); err != nil {
		return fmt.Errorf(`closing state socket failed - %v`, err)
	}

	if err := msgs.Close(); err != nil {
		return fmt.Errorf(`closing data socket failed - %v`, err)
	}

	return nil
}
