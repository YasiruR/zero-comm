package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/btcsuite/btcutil/base58"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"net/url"
	"strings"
)

type publisher struct {
	peer     string
	endpoint string
	pubKey   []byte
}

type Subscriber struct {
	label   string
	sktPubs *zmq.Socket
	sktMsgs *zmq.Socket
	prb     domain.DIDCommService
	ks      domain.KeyService
	log     log.Logger

	connDone chan domain.Connection // todo send msgs from agent only if not null
	//brokrPubKeyMap map[string][]byte
	//topicPubMap map[string][]publisher
	topicBrokrMap map[string][]string // publisher list for each topic
	topicPeerMap  map[string][]string
}

func NewSubscriber(zmqCtx *zmq.Context, c *domain.Container) (*Subscriber, error) {
	sktPubs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for _pubs topics failed - %v`, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	s := &Subscriber{
		label:    c.Cfg.Args.Name,
		sktPubs:  sktPubs,
		sktMsgs:  sktMsgs,
		prb:      c.Prober,
		ks:       c.KS,
		log:      c.Log,
		connDone: c.ConnDoneChan,
		//brokrPubKeyMap: make(map[string][]byte),
		//topicPubMap:    make(map[string][]publisher),
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

	//for _, b := range brokers {
	//	if s.topicPubMap[topic] == nil {
	//		s.topicPubMap[topic] = []publisher{}
	//	}
	//	s.topicPubMap[topic] = append(s.topicPubMap[topic], publisher{endpoint: b})
	//}
}

func (s *Subscriber) Subscribe(topic string) error {
	if err := s.subscribePubs(topic); err != nil {
		return fmt.Errorf(`subscribing to %s_pubs topic failed - %v`, topic, err)
	}

	return nil
}

func (s *Subscriber) subscribePubs(topic string) error {
	// todo should be continuous for dynamic subscriptions and publishers
	for _, pubEndpoint := range s.topicBrokrMap[topic] {
		if err := s.sktPubs.Connect(pubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher (%s) failed - %v`, pubEndpoint, err)
		}

		if err := s.sktPubs.SetSubscribe(topic + `_pubs`); err != nil {
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
			// remove publisher
			continue
		}

		inv, err := s.parseInvURL(pub.Inv)
		if err != nil {
			s.log.Error(fmt.Sprintf(`parsing invitation url of publisher failed - %v`, err))
		}

		inviter, err := s.prb.Accept(inv)
		if err != nil {
			s.log.Error(fmt.Sprintf(`accepting did invitation failed - %v`, err))
		}

		if s.topicPeerMap[pub.Topic] == nil {
			s.topicPeerMap[pub.Topic] = []string{}
		}
		s.topicPeerMap[pub.Topic] = append(s.topicPeerMap[pub.Topic], inviter)
	}
}

func (s *Subscriber) initAddPubs() {
	for {
		// add termination
		conn := <-s.connDone // todo check if this works with just req rep
		subPublicKey, err := s.ks.PublicKey(conn.Peer)
		if err != nil {
			s.log.Error(fmt.Sprintf(`getting public key for the connection with %s failed - %v`, conn.Peer, err))
			continue
		}

		// fetching topics of the publisher connected
		var topics []string
		for topic, pubs := range s.topicPeerMap {
			for _, pub := range pubs {
				fmt.Println("CHECKING PUBS: ", pub, conn.Peer)
				if pub == conn.Peer {
					topics = append(topics, topic)
				}
			}
		}

		sm := messages.SubscribeMsg{Peer: s.label, PubKey: base58.Encode(subPublicKey), Topics: topics}
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
		}

		//s.brokrPubKeyMap[conn.Peer] = conn.PubKey
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

		text, err := s.prb.ReadMessage([]byte(msg))
		if err != nil {
			s.log.Error(fmt.Sprintf(`reading subscribed message failed - %v`, err))
			continue
		}

		fmt.Printf("-> Message received: %s\n", text)
	}
}

func (s *Subscriber) Close() error {
	return nil
}
