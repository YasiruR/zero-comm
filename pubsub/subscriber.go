package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"net/url"
	"strings"
)

type Subscriber struct {
	label         string
	sktPubs       *zmq.Socket
	sktMsgs       *zmq.Socket
	prb           services.DIDComm
	ks            services.KeyManager
	log           log.Logger
	connDone      chan models.Connection
	outChan       chan string
	topicBrokrMap map[string][]string // broker list for each topic
	topicPeerMap  map[string][]string // use sync map for both
}

func NewSubscriber(zmqCtx *zmq.Context, c *domain.Container) (*Subscriber, error) {
	sktPubs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for %s topics failed - %v`, domain.PubTopicSuffix, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	s := &Subscriber{
		label:         c.Cfg.Args.Name,
		sktPubs:       sktPubs,
		sktMsgs:       sktMsgs,
		prb:           c.Prober,
		ks:            c.KS,
		log:           c.Log,
		connDone:      c.ConnDoneChan,
		outChan:       c.OutChan,
		topicBrokrMap: make(map[string][]string),
		topicPeerMap:  make(map[string][]string),
	}

	go s.initReqConns()
	go s.initAddPubs()
	go s.listen()

	return s, nil
}

func (s *Subscriber) AddBrokers(topic string, brokers []string) {
	s.topicBrokrMap[topic] = brokers
}

func (s *Subscriber) Subscribe(topic string) error {
	if err := s.subscribePubs(topic); err != nil {
		return fmt.Errorf(`subscribing to %s%s topic failed - %v`, topic, domain.PubTopicSuffix, err)
	}

	return nil
}

func (s *Subscriber) Unsubscribe(topic string) error {
	peers, ok := s.topicPeerMap[topic]
	if !ok {
		return fmt.Errorf(`no subscription found`)
	}

	if err := s.sktPubs.SetUnsubscribe(topic + domain.PubTopicSuffix); err != nil {
		return fmt.Errorf(`unsubscribing %s%s via zmq socket failed - %v`, topic, domain.PubTopicSuffix, err)
	}

	for _, peer := range peers {
		subTopic := topic + `_` + peer + `_` + s.label
		if err := s.sktMsgs.SetUnsubscribe(subTopic); err != nil {
			return fmt.Errorf(`unsubscribing to zmq socket failed - %v`, err)
		}

		sm := messages.SubscribeMsg{Subscribe: false, Peer: s.label, Topics: []string{topic}}
		byts, err := json.Marshal(sm)
		if err != nil {
			return fmt.Errorf(`marshalling unsubscribe message failed - %v`, err)
		}

		if err = s.prb.SendMessage(domain.MsgTypSubscribe, peer, string(byts)); err != nil {
			return fmt.Errorf(`sending unsubscribe message failed - %v`, err)
		}

		s.log.Trace(fmt.Sprintf(`unsubscribed to sub-topic %s`, subTopic))
	}

	s.outChan <- `Unsubscribed to ` + topic
	return nil
}

func (s *Subscriber) subscribePubs(topic string) error {
	// todo should be continuous for dynamic subscriptions and publishers
	for _, pubEndpoint := range s.topicBrokrMap[topic] {
		if err := s.sktPubs.Connect(pubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher for status (%s) failed - %v`, pubEndpoint, err)
		}

		if err := s.sktMsgs.Connect(pubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher for messages (%s) failed - %v`, pubEndpoint, err)
		}

		if err := s.sktPubs.SetSubscribe(topic + domain.PubTopicSuffix); err != nil {
			return fmt.Errorf(`setting zmq subscription failed - %v`, err)
		}
	}

	return nil
}

func (s *Subscriber) initReqConns() {
	for {
		// add termination
		msg, err := s.sktPubs.Recv(0)
		if err != nil {
			s.log.Error(fmt.Sprintf(`receiving zmq message for publisher status failed - %v`, err))
			continue
		}

		frames := strings.Split(msg, " ")
		if len(frames) != 2 {
			s.log.Error(fmt.Sprintf(`received a message (%v) with an invalid format - frame count should be 2`, msg))
			continue
		}

		var pub messages.PublisherStatus
		if err = json.Unmarshal([]byte(frames[1]), &pub); err != nil {
			s.log.Error(fmt.Sprintf(`unmarshalling publisher status failed (msg: %s) - %v`, frames[1], err))
			continue
		}

		if !pub.Active {
			if err = s.removePub(pub.Topic, pub.Label); err != nil {
				s.log.Error(fmt.Sprintf(`removing publisher failed - %v`, err))
				continue
			}

			s.outChan <- `Removed publisher '` + pub.Label + `' from topic '` + pub.Topic + `'`
			continue
		}

		inv, err := s.parseInvURL(pub.Inv)
		if err != nil {
			s.log.Error(fmt.Sprintf(`parsing invitation url of publisher failed - %v`, err))
			continue
		}

		inviter, err := s.prb.Accept(inv)
		if err != nil {
			s.log.Error(fmt.Sprintf(`accepting did invitation failed - %v`, err))
			continue
		}

		if s.topicPeerMap[pub.Topic] == nil {
			s.topicPeerMap[pub.Topic] = []string{}
		}
		s.topicPeerMap[pub.Topic] = append(s.topicPeerMap[pub.Topic], inviter)
	}
}

func (s *Subscriber) removePub(topic, label string) error {
	subTopic := topic + `_` + label + `_` + s.label
	if err := s.sktMsgs.SetUnsubscribe(subTopic); err != nil {
		return fmt.Errorf(`unsubscribing topic %s via zmq socket failed - %v`, subTopic, err)
	}

	var tmpPubs []string
	for _, pub := range s.topicPeerMap[topic] {
		if pub != label {
			tmpPubs = append(tmpPubs, pub)
		}
	}
	s.topicPeerMap[topic] = tmpPubs

	return nil
}

func (s *Subscriber) initAddPubs() {
	for {
		// add termination
		conn := <-s.connDone
		subPublicKey, err := s.ks.PublicKey(conn.Peer)
		if err != nil {
			s.log.Error(fmt.Sprintf(`getting public key for the connection with %s failed - %v`, conn.Peer, err))
			continue
		}

		// fetching topics of the publisher connected
		var topics []string
		for topic, pubs := range s.topicPeerMap {
			for _, pub := range pubs {
				if pub == conn.Peer {
					topics = append(topics, topic)
				}
			}
		}

		if len(topics) == 0 {
			continue
		}

		sm := messages.SubscribeMsg{Subscribe: true, Peer: s.label, PubKey: base58.Encode(subPublicKey), Topics: topics}
		byts, err := json.Marshal(sm)
		if err != nil {
			s.log.Error(fmt.Sprintf(`marshalling subscribe message failed - %v`, err))
			continue
		}

		if err = s.prb.SendMessage(domain.MsgTypSubscribe, conn.Peer, string(byts)); err != nil {
			s.log.Error(fmt.Sprintf(`sending subscribe message failed - %v`, err))
			continue
		}

		// subscribing to all topics of the publisher (topic syntax: topic_pub_sub)
		for _, t := range topics {
			subTopic := t + `_` + conn.Peer + `_` + s.label
			if err = s.sktMsgs.SetSubscribe(subTopic); err != nil {
				s.log.Error(fmt.Sprintf(`setting zmq subscription failed for topic %s - %v`, subTopic, err))
				continue
			}
			s.log.Trace(fmt.Sprintf(`subscribed to sub-topic %s`, subTopic))
		}
	}
}

func (s *Subscriber) parseInvURL(rawUrl string) (inv string, err error) {
	u, err := url.Parse(strings.TrimSpace(rawUrl))
	if err != nil {
		return ``, fmt.Errorf(`parsing url failed - %v`, err)
	}

	params, ok := u.Query()[`oob`]
	if !ok {
		return ``, fmt.Errorf(`url does not contain an 'oob' parameter`)
	}

	return params[0], nil
}

func (s *Subscriber) listen() {
	for {
		msg, err := s.sktMsgs.Recv(0)
		if err != nil {
			s.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
			continue
		}

		frames := strings.Split(msg, ` `)
		if len(frames) != 2 {
			s.log.Error(fmt.Sprintf(`received an invalid subscribed message (%v) - %v`, frames, err))
			continue
		}

		_, err = s.prb.ReadMessage(domain.MsgTypData, []byte(frames[1]))
		if err != nil {
			s.log.Error(fmt.Sprintf(`reading subscribed message failed - %v`, err))
			continue
		}
	}
}

func (s *Subscriber) Close() error {
	return nil
}
