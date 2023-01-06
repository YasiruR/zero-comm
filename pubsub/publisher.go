package pubsub

//type Publisher struct {
//	label   string
//	skt     *zmq.Socket
//	prb     services.Agent
//	ks      services.KeyManager
//	packer  services.Packer
//	log     log.Logger
//	outChan chan string
//	ts      *keyStore
//}
//
//// todo add publisher as a service endpoint in did doc / invitation
//
//func NewPublisher(zmqCtx *zmq.Context, c *domain.Container) (*Publisher, error) {
//	skt, err := zmqCtx.NewSocket(zmq.PUB)
//	if err != nil {
//		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
//	}
//
//	if err = skt.Bind(c.Cfg.PubEndpoint); err != nil {
//		return nil, fmt.Errorf(`binding zmq pub socket to %s failed - %v`, c.Cfg.PubEndpoint, err)
//	}
//
//	p := &Publisher{
//		label:   c.Cfg.Args.Name,
//		skt:     skt,
//		prb:     c.Prober,
//		ks:      c.KeyManager,
//		packer:  c.Packer,
//		log:     c.Log,
//		outChan: c.OutChan,
//		ts:      &keyStore{RWMutex: &sync.RWMutex{}, subs: map[string]subKey{}},
//	}
//
//	p.initHandlers(c.Server)
//	return p, err
//}
//
//func (p *Publisher) initHandlers(s services.Server) {
//	subChan := make(chan models.Message)
//	s.AddHandler(domain.MsgTypSubscribe, subChan, true)
//	go p.listen(subChan)
//}
//
//func (p *Publisher) Register(topic string) error {
//	// can add topic details later to the invitation
//	inv, err := p.prb.Invite()
//	if err != nil {
//		return fmt.Errorf(`generating invitation failed - %v`, err)
//	}
//
//	status := messages.PublisherStatus{Label: p.label, Active: true, Inv: inv, Topic: topic}
//	byts, err := json.Marshal(status)
//	if err != nil {
//		return fmt.Errorf(`marshalling publisher status failed - %v`, err)
//	}
//
//	if _, err = p.skt.SendMessage(fmt.Sprintf(`%s%s %s`, topic, domain.PubTopicSuffix, string(byts))); err != nil {
//		return fmt.Errorf(`publishing active status failed - %v`, err)
//	}
//
//	return nil
//}
//
//// listen follows subscription of a topic which is done through a separate
//// DIDComm message. Alternatively, it can be included in connection request.
//func (p *Publisher) listen(subChan chan models.Message) {
//	for {
//		// add termination
//		msg := <-subChan
//		unpackedMsg, err := p.prb.ReadMessage(msg)
//		if err != nil {
//			p.log.Error(fmt.Sprintf(`reading subscribe msg failed - %v`, err))
//			continue
//		}
//
//		var sm messages.SubscribeMsg
//		if err = json.Unmarshal([]byte(unpackedMsg), &sm); err != nil {
//			p.log.Error(fmt.Sprintf(`unmarshalling subscribe message failed - %v`, err))
//			continue
//		}
//
//		if !sm.Subscribe {
//			p.deleteSub(sm)
//			continue
//		}
//
//		sk := base58.Decode(sm.PubKey)
//		for _, t := range sm.Topics {
//			p.addSub(t, sm.Peer, sk)
//		}
//	}
//}
//
//func (p *Publisher) Publish(topic, msg string) error {
//	subs, err := p.subsByTopic(topic)
//	if err != nil {
//		return fmt.Errorf(`fetching subscribers for topic %s failed - %v`, topic, err)
//	}
//
//	var published bool
//	for sub, key := range subs {
//		ownPubKey, err := p.ks.PublicKey(sub)
//		if err != nil {
//			return fmt.Errorf(`getting public key for connection with %s failed - %v`, sub, err)
//		}
//
//		ownPrvKey, err := p.ks.PrivateKey(sub)
//		if err != nil {
//			return fmt.Errorf(`getting private key for connection with %s failed - %v`, sub, err)
//		}
//
//		encryptdMsg, err := p.packer.Pack([]byte(msg), key, ownPubKey, ownPrvKey)
//		if err != nil {
//			p.log.Error(err)
//			return err
//		}
//
//		data, err := json.Marshal(encryptdMsg)
//		if err != nil {
//			p.log.Error(err)
//			return err
//		}
//
//		subTopic := topic + `_` + p.label + `_` + sub
//		if _, err = p.skt.SendMessage(fmt.Sprintf(`%s %s`, subTopic, string(data))); err != nil {
//			return fmt.Errorf(`publishing message (%s) failed for %s - %v`, msg, sub, err)
//		}
//
//		published = true
//		p.log.Trace(fmt.Sprintf(`published %s to %s`, msg, subTopic))
//	}
//
//	if published {
//		p.outChan <- `Published '` + msg + `' to '` + topic + `'`
//	}
//	return nil
//}
//
//func (p *Publisher) Unregister(topic string) error {
//	status := messages.PublisherStatus{Label: p.label, Active: false, Topic: topic}
//	byts, err := json.Marshal(status)
//	if err != nil {
//		return fmt.Errorf(`marshalling publisher inactive status failed - %v`, err)
//	}
//
//	if _, err = p.skt.SendMessage(fmt.Sprintf(`%s%s %s`, topic, domain.PubTopicSuffix, string(byts))); err != nil {
//		return fmt.Errorf(`publishing inactive status failed - %v`, err)
//	}
//
//	p.deleteTopic(topic)
//	p.outChan <- `Unregistered ` + topic
//	return nil
//}
//
//func (p *Publisher) subsByTopic(topic string) (subKey, error) {
//	p.ts.RLock()
//	defer p.ts.RUnlock()
//	subs, ok := p.ts.subs[topic]
//	if !ok {
//		return nil, fmt.Errorf(`topic (%s) is not registered`, topic)
//	}
//
//	return subs, nil
//}
//
//// addSub replaces the key if already exists for the subscriber
//func (p *Publisher) addSub(topic, sub string, key []byte) {
//	p.ts.Lock()
//	defer p.ts.Unlock()
//	if p.ts.subs[topic] == nil {
//		p.ts.subs[topic] = subKey{}
//	}
//	p.ts.subs[topic][sub] = key
//}
//
//func (p *Publisher) deleteSub(sm messages.SubscribeMsg) {
//	p.ts.Lock()
//	defer p.ts.Unlock()
//	for _, t := range sm.Topics {
//		delete(p.ts.subs[t], sm.Peer)
//	}
//}
//
//func (p *Publisher) deleteTopic(topic string) {
//	p.ts.Lock()
//	defer p.ts.Unlock()
//	delete(p.ts.subs, topic)
//}
//
//func (p *Publisher) Close() error {
//	// todo publish inactive to all _pubs
//	// close sockets
//	// close context
//
//	return nil
//}
