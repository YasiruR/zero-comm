package transport

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

const (
	errTempUnavail = `resource temporarily unavailable`
)

type Zmq struct {
	server     *zmq.Socket
	log        log.Logger
	inChanExch chan []byte
	inChanData chan []byte
	ctx        *zmq.Context
	peers      map[string]*zmq.Socket // use sync map if accessed concurrently
}

func NewZmq(c *domain.Container) (*Zmq, error) {
	ctx, err := zmq.NewContext()
	if err != nil {
		return nil, fmt.Errorf(`zmq context initialization failed - %v`, err)
	}

	repSkt, err := ctx.NewSocket(zmq.REP)
	if err != nil {
		return nil, fmt.Errorf(`constructing zmq server socket failed - %v`, err)
	}

	if err = repSkt.Bind(c.Cfg.InvEndpoint); err != nil {
		return nil, fmt.Errorf(`binding zmq socket to %s failed - %v`, c.Cfg.InvEndpoint, err)
	}

	return &Zmq{ctx: ctx, peers: map[string]*zmq.Socket{}, server: repSkt, log: c.Log, inChanExch: c.InChanExch, inChanData: c.InChanData}, nil
}

func (z *Zmq) Socket(endpoint string) (skt *zmq.Socket, err error) {
	// todo check if different sockets are required for different clients
	skt, ok := z.peers[endpoint]
	if ok {
		return skt, nil
	}

	skt, err = z.ctx.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fmt.Errorf(`creating new socket for endpoint %s failed - %v`, endpoint, err)
	}

	if err = skt.Connect(endpoint); err != nil {
		return nil, fmt.Errorf(`connecting to zmq socket (%s) failed - %v`, endpoint, err)
	}

	z.peers[endpoint] = skt
	return skt, nil
}

func (z *Zmq) Start() {
	for {
		msg, err := z.server.RecvMessage(0)
		if err != nil {
			if err.Error() != errTempUnavail {
				z.log.Error(fmt.Sprintf(`receiving zmq message by receiver failed - %v`, err))
			}
			continue
		}

		if len(msg) != 2 {
			z.log.Error(`received an empty/invalid message`, msg)
			continue
		}

		switch msg[0] {
		case domain.MsgTypExchange:
			z.inChanExch <- []byte(msg[1])
		case domain.MsgTypData:
			z.inChanData <- []byte(msg[1])
		default:
			z.log.Error(`invalid message type`, msg)
		}

		if _, err = z.server.Send(`done`, 0); err != nil {
			z.log.Error(`sending zmq message by receiver failed - %v`, err)
		}
	}
}

// Send connects to the endpoint per each message since it is more appropriate
// with DIDComm as by nature it manifests an asynchronous simplex communication.
func (z *Zmq) Send(typ string, data []byte, endpoint string) error {
	skt, err := z.Socket(endpoint)
	if err != nil {
		return fmt.Errorf(`fetching zmq socket failed - %v`, err)
	}

	if _, err = skt.SendMessage(typ, string(data)); err != nil {
		return fmt.Errorf(`sending zmq message by sender failed - %v`, err)
	}

receive:
	if _, err = skt.RecvMessage(0); err != nil {
		if err.Error() == errTempUnavail {
			goto receive
		}
		return fmt.Errorf(`receiving zmq message by sender failed - %v`, err)
	}

	return nil
}

func (z *Zmq) Subscribe(topic string) {

}

func (z *Zmq) Stop() error {
	z.server.Close()
	for _, sckt := range z.peers {
		sckt.Close()
	}

	zmq.NewPoller()

	return nil
}
