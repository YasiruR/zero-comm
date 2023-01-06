package pubsub

//type Subscriber struct {
//	label        string
//	sktPubs      *zmq.Socket
//	sktMsgs      *zmq.Socket
//	probr        services.Agent
//	km           services.KeyManager
//	topcBrokrMap *sync.Map // broker list for each topic
//	topcPubMap   *sync.Map
//	connDone     chan models.Connection
//	outChan      chan string
//	log          log.Logger
//}
//
//func NewSubscriber(zmqCtx *zmq.Context, c *domain.Container) (*Subscriber, error) {
//	sktPubs, err := zmqCtx.NewSocket(zmq.SUB)
//	if err != nil {
//		return nil, fmt.Errorf(`creating sub socket for %s topics failed - %v`, domain.PubTopicSuffix, err)
//	}
//
//	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
//	if err != nil {
//		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
//	}
//
//	s := &Subscriber{
//		label:        c.Cfg.Args.Name,
//		sktPubs:      sktPubs,
//		sktMsgs:      sktMsgs,
//		probr:        c.Prober,
//		km:           c.KeyManager,
//		log:          c.Log,
//		connDone:     c.ConnDoneChan,
//		outChan:      c.OutChan,
//		topcBrokrMap: &sync.Map{},
//		topcPubMap:   &sync.Map{},
//	}
//
//	go s.initPubListnr()
//	go s.initConnListnr()
//	go s.initMsgListnr()
//
//	return s, nil
//}
//
//// initPubListnr listens to the publisher status socket for any updates
//// (active/inactive) and handles accordingly in the subscriber
//func (s *Subscriber) initPubListnr() {
//	for {
//		// add termination
//		msg, err := s.sktPubs.Recv(0)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`receiving zmq message for publisher status failed - %v`, err))
//			continue
//		}
//
//		pub, err := s.parsePubStatus(msg)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`parsing publisher status message failed - %v`, err))
//			continue
//		}
//
//		if !pub.Active {
//			if err = s.unsubscribePub(pub.Topic, pub.Label); err != nil {
//				s.log.Error(fmt.Sprintf(`unsubscribing publisher failed - %v`, err))
//				continue
//			}
//
//			if err = s.deletePub(pub.Topic, pub.Label); err != nil {
//				s.log.Error(fmt.Sprintf(`removing publisher failed - %v`, err))
//				continue
//			}
//
//			s.outChan <- `Removed publisher '` + pub.Label + `' from topic '` + pub.Topic + `'`
//			continue
//		}
//
//		inv, err := s.parseInvURL(pub.Inv)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`parsing invitation url of publisher failed - %v`, err))
//			continue
//		}
//
//		inviter, err := s.probr.Accept(inv)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`accepting did invitation failed - %v`, err))
//			continue
//		}
//
//		if err = s.addPub(pub.Topic, inviter); err != nil {
//			s.log.Error(fmt.Sprintf(`adding peer to topic %s failed - %v`, pub.Topic, err))
//		}
//	}
//}
//
//// initConnListnr listens to any connections established by the agent and performs
//// setting up pub-sub relationship if the connected peer is a publisher
//func (s *Subscriber) initConnListnr() {
//	for {
//		// add termination
//		conn := <-s.connDone
//		subPublicKey, err := s.km.PublicKey(conn.Peer)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`getting public key for the connection with %s failed - %v`, conn.Peer, err))
//			continue
//		}
//
//		// fetching topics of the publisher connected
//		topics, err := s.topicsByPub(conn.Peer)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`fetching topics for publisher %s failed - %v`, conn.Peer, err))
//			continue
//		}
//
//		if len(topics) == 0 {
//			continue
//		}
//
//		sm := messages.SubscribeMsg{
//			Id:        uuid.New().String(),
//			Type:      messages.SubscribeV1,
//			Subscribe: true,
//			Peer:      s.label,
//			PubKey:    base58.Encode(subPublicKey),
//			Topics:    topics,
//		}
//
//		byts, err := json.Marshal(sm)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`marshalling subscribe message failed - %v`, err))
//			continue
//		}
//
//		if err = s.probr.SendMessage(domain.MsgTypSubscribe, conn.Peer, string(byts)); err != nil {
//			s.log.Error(fmt.Sprintf(`sending subscribe message failed - %v`, err))
//			continue
//		}
//
//		// subscribing to all topics of the publisher (topic syntax: topic_pub_sub)
//		for _, t := range topics {
//			subTopic := t + `_` + conn.Peer + `_` + s.label
//			if err = s.sktMsgs.SetSubscribe(subTopic); err != nil {
//				s.log.Error(fmt.Sprintf(`setting zmq subscription failed for topic %s - %v`, subTopic, err))
//				continue
//			}
//			s.log.Trace(fmt.Sprintf(`subscribed to sub-topic %s`, subTopic))
//		}
//	}
//}
//
//func (s *Subscriber) initMsgListnr() {
//	for {
//		msg, err := s.sktMsgs.Recv(0)
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
//			continue
//		}
//
//		frames := strings.Split(msg, ` `)
//		if len(frames) != 2 {
//			s.log.Error(fmt.Sprintf(`received an invalid subscribed message (%v) - %v`, frames, err))
//			continue
//		}
//
//		_, err = s.probr.ReadMessage(models.Message{Type: domain.MsgTypData, Data: []byte(frames[1])})
//		if err != nil {
//			s.log.Error(fmt.Sprintf(`reading subscribed message failed - %v`, err))
//			continue
//		}
//	}
//}
//
//func (s *Subscriber) AddBrokers(topic string, brokers []string) {
//	s.topcBrokrMap.Store(topic, brokers)
//}
//
//func (s *Subscriber) Subscribe(topic string) error {
//	if err := s.subscribePubs(topic); err != nil {
//		return fmt.Errorf(`subscribing to %s%s topic failed - %v`, topic, domain.PubTopicSuffix, err)
//	}
//
//	return nil
//}
//
//func (s *Subscriber) Unsubscribe(topic string) error {
//	pubs, err := s.pubsByTopic(topic)
//	if err != nil {
//		return fmt.Errorf(`fetching publishers for topic %s failed - %v`, topic, err)
//	}
//
//	if len(pubs) == 0 {
//		return fmt.Errorf(`no subscription found for topic %s`, topic)
//	}
//
//	if err = s.sktPubs.SetUnsubscribe(topic + domain.PubTopicSuffix); err != nil {
//		return fmt.Errorf(`unsubscribing %s%s via zmq socket failed - %v`, topic, domain.PubTopicSuffix, err)
//	}
//
//	for _, peer := range pubs {
//		subTopic := topic + `_` + peer + `_` + s.label
//		if err = s.sktMsgs.SetUnsubscribe(subTopic); err != nil {
//			return fmt.Errorf(`unsubscribing to zmq socket failed - %v`, err)
//		}
//
//		sm := messages.SubscribeMsg{
//			Id:        uuid.New().String(),
//			Type:      messages.SubscribeV1,
//			Subscribe: false,
//			Peer:      s.label,
//			Topics:    []string{topic},
//		}
//
//		byts, err := json.Marshal(sm)
//		if err != nil {
//			return fmt.Errorf(`marshalling unsubscribe message failed - %v`, err)
//		}
//
//		if err = s.probr.SendMessage(domain.MsgTypSubscribe, peer, string(byts)); err != nil {
//			return fmt.Errorf(`sending unsubscribe message failed - %v`, err)
//		}
//
//		s.log.Trace(fmt.Sprintf(`unsubscribed to sub-topic %s`, subTopic))
//	}
//
//	s.outChan <- `Unsubscribed to ` + topic
//	return nil
//}
//
//func (s *Subscriber) subscribePubs(topic string) error {
//	// todo should be continuous for dynamic subscriptions and publishers
//	// todo may need not to do for already connected pubs
//	brokrs, err := s.brokrsByTopic(topic)
//	if err != nil {
//		return fmt.Errorf(`fetching brokers for topic %s failed - %v`, topic, err)
//	}
//
//	for _, pubEndpoint := range brokrs {
//		if err = s.sktPubs.Connect(pubEndpoint); err != nil {
//			return fmt.Errorf(`connecting to publisher for status (%s) failed - %v`, pubEndpoint, err)
//		}
//
//		if err = s.sktMsgs.Connect(pubEndpoint); err != nil {
//			return fmt.Errorf(`connecting to publisher for messages (%s) failed - %v`, pubEndpoint, err)
//		}
//	}
//
//	// if zmq subscription is called multiple times, it will create multiple instances of subscribers
//	if err = s.sktPubs.SetSubscribe(topic + domain.PubTopicSuffix); err != nil {
//		return fmt.Errorf(`setting zmq subscription failed - %v`, err)
//	}
//
//	return nil
//}
//
//func (s *Subscriber) parsePubStatus(msg string) (*messages.PublisherStatus, error) {
//	frames := strings.Split(msg, " ")
//	if len(frames) != 2 {
//		return nil, fmt.Errorf(`received a message (%v) with an invalid format - frame count should be 2`, msg)
//	}
//
//	var pub messages.PublisherStatus
//	if err := json.Unmarshal([]byte(frames[1]), &pub); err != nil {
//		return nil, fmt.Errorf(`unmarshalling publisher status failed (msg: %s) - %v`, frames[1], err)
//	}
//
//	return &pub, nil
//}
//
//func (s *Subscriber) parseInvURL(rawUrl string) (inv string, err error) {
//	u, err := url.Parse(strings.TrimSpace(rawUrl))
//	if err != nil {
//		return ``, fmt.Errorf(`parsing url failed - %v`, err)
//	}
//
//	params, ok := u.Query()[`oob`]
//	if !ok {
//		return ``, fmt.Errorf(`url does not contain an 'oob' parameter`)
//	}
//
//	return params[0], nil
//}
//
//func (s *Subscriber) brokrsByTopic(topic string) (brokrs []string, err error) {
//	val, ok := s.topcBrokrMap.Load(topic)
//	if !ok {
//		return nil, fmt.Errorf(`no brokers found for topic %v`, topic)
//	}
//
//	brokrs, ok = val.([]string)
//	if !ok {
//		return nil, fmt.Errorf(`incompatible value found for brokers (%v) - should be []string`, val)
//	}
//
//	return brokrs, nil
//}
//
//func (s *Subscriber) pubsByTopic(topic string) (pubs []string, err error) {
//	val, ok := s.topcPubMap.Load(topic)
//	if !ok {
//		return nil, nil
//	}
//
//	pubs, ok = val.([]string)
//	if !ok {
//		return nil, fmt.Errorf(`invalid publishers found (%v) for topic %s - should be []string`, val, topic)
//	}
//
//	return pubs, nil
//}
//
//func (s *Subscriber) topicsByPub(pub string) (topics []string, err error) {
//	s.topcPubMap.Range(func(key, val any) bool {
//		topic, ok := key.(string)
//		if !ok {
//			err = fmt.Errorf(`invalid topic key (%v) - should be []string`, key)
//			return false
//		}
//
//		pubs, ok := val.([]string)
//		if !ok {
//			err = fmt.Errorf(`invalid publisher list found (%v) for topic %s - should be []string`, val, key)
//			return false
//		}
//
//		for _, connPub := range pubs {
//			if connPub == pub {
//				topics = append(topics, topic)
//				return true
//			}
//		}
//		return true
//	})
//
//	if err != nil {
//		return nil, fmt.Errorf(`iterating topic-publisher sync map failed - %v`, err)
//	}
//
//	return topics, nil
//}
//
//func (s *Subscriber) addPub(topic, pub string) error {
//	pubs, err := s.pubsByTopic(topic)
//	if err != nil {
//		return fmt.Errorf(`fetching publishers failed - %v`, err)
//	}
//
//	if pubs == nil {
//		pubs = []string{}
//	}
//
//	s.topcPubMap.Store(topic, append(pubs, pub))
//	return nil
//}
//
//func (s *Subscriber) deletePub(topic, label string) error {
//	pubs, err := s.pubsByTopic(topic)
//	if err != nil {
//		return fmt.Errorf(`fetching publishers failed - %v`, err)
//	}
//
//	var tmpPubs []string
//	for _, pub := range pubs {
//		if pub != label {
//			tmpPubs = append(tmpPubs, pub)
//		}
//	}
//	s.topcPubMap.Store(topic, tmpPubs)
//
//	return nil
//}
//
//func (s *Subscriber) unsubscribePub(topic, label string) error {
//	subTopic := topic + `_` + label + `_` + s.label
//	if err := s.sktMsgs.SetUnsubscribe(subTopic); err != nil {
//		return fmt.Errorf(`unsubscribing topic %s via zmq socket failed - %v`, subTopic, err)
//	}
//	return nil
//}
//
//func (s *Subscriber) Close() error {
//	return nil
//}
