package reqrep

//const (
//	errTempUnavail = `resource temporarily unavailable`
//)
//
//type Zmq struct {
//	ctx                        *zmq.Context
//	server                     *zmq.Socket
//	log                        log.Logger
//	inChan, subChan, queryChan chan models.Message
//	peers                      map[string]*zmq.Socket // use sync map if accessed concurrently
//}
//
//func NewZmq(zmqCtx *zmq.Context, c *domain.Container) (*Zmq, error) {
//	repSkt, err := zmqCtx.NewSocket(zmq.REP)
//	if err != nil {
//		return nil, fmt.Errorf(`constructing zmq server socket failed - %v`, err)
//	}
//
//	if err = repSkt.Bind(c.Cfg.InvEndpoint); err != nil {
//		return nil, fmt.Errorf(`binding zmq socket to %s failed - %v`, c.Cfg.InvEndpoint, err)
//	}
//
//	return &Zmq{
//		ctx:     zmqCtx,
//		peers:   map[string]*zmq.Socket{},
//		server:  repSkt,
//		log:     c.Log,
//		inChan:  c.InChan,
//		subChan: c.SubChan,
//	}, nil
//}
//
//func (z *Zmq) Socket(endpoint string) (skt *zmq.Socket, err error) {
//	skt, ok := z.peers[endpoint]
//	if ok {
//		return skt, nil
//	}
//
//	skt, err = z.ctx.NewSocket(zmq.REQ)
//	if err != nil {
//		return nil, fmt.Errorf(`creating new socket for endpoint %s failed - %v`, endpoint, err)
//	}
//
//	if err = skt.Connect(endpoint); err != nil {
//		return nil, fmt.Errorf(`connecting to zmq socket (%s) failed - %v`, endpoint, err)
//	}
//
//	z.peers[endpoint] = skt
//	return skt, nil
//}
//
//func (z *Zmq) Start() {
//	for {
//		msg, err := z.server.RecvMessage(0)
//		if err != nil {
//			if err.Error() != errTempUnavail {
//				z.log.Error(fmt.Sprintf(`receiving zmq message by receiver failed - %v`, err))
//			}
//			continue
//		}
//
//		if len(msg) != 2 {
//			z.log.Error(`received an empty/invalid message`, msg)
//			continue
//		}
//
//		m := models.Message{Type: msg[0], Data: []byte(msg[1])}
//		switch msg[0] {
//		case domain.MsgTypConnReq:
//			z.inChan <- m
//		case domain.MsgTypConnRes:
//			z.inChan <- m
//		case domain.MsgTypData:
//			z.inChan <- m
//		case domain.MsgTypSubscribe:
//			z.subChan <- m
//		case domain.MsgTypQuery:
//			z.queryChan <- m
//		default:
//			z.log.Error(`invalid message type`, msg)
//		}
//
//		if _, err = z.server.Send(`done`, 0); err != nil {
//			z.log.Error(`sending zmq message by receiver failed - %v`, err)
//		}
//	}
//}
//
//// Send connects to the endpoint per each message since it is more appropriate
//// with DIDComm as by nature it manifests an asynchronous simplex communication.
//func (z *Zmq) Send(typ string, data []byte, endpoint string) (msg []string, err error) {
//	skt, err := z.Socket(endpoint)
//	if err != nil {
//		return nil, fmt.Errorf(`fetching zmq socket failed - %v`, err)
//	}
//
//	if _, err = skt.SendMessage(typ, string(data)); err != nil {
//		return nil, fmt.Errorf(`sending zmq message by sender failed - %v`, err)
//	}
//
//receive:
//	if msg, err = skt.RecvMessage(0); err != nil {
//		if err.Error() == errTempUnavail {
//			goto receive
//		}
//		return nil, fmt.Errorf(`receiving zmq message by sender failed - %v`, err)
//	}
//
//	return msg, nil
//}
//
//func (z *Zmq) Stop() error {
//	z.server.Close()
//	for _, sckt := range z.peers {
//		sckt.Close()
//	}
//
//	zmq.NewPoller()
//
//	return nil
//}
