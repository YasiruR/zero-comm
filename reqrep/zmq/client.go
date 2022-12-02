package zmq

import (
	"encoding/json"
	"fmt"
	zmq "github.com/pebbe/zmq4"
)

type Client struct {
	ctx   *zmq.Context
	peers map[string]*zmq.Socket // use sync map if accessed concurrently
}

func NewClient(zmqCtx *zmq.Context) *Client {
	return &Client{
		ctx:   zmqCtx,
		peers: make(map[string]*zmq.Socket),
	}
}

// Send connects to the endpoint per each message since it is more appropriate8
// with DIDComm as by nature it manifests an asynchronous simplex communication.
func (c *Client) Send(typ string, data []byte, endpoint string) (res string, err error) {
	skt, err := c.socket(endpoint)
	if err != nil {
		return ``, fmt.Errorf(`fetching zmq socket failed - %v`, err)
	}

	metaByts, err := json.Marshal(metadata{Type: typ})
	if err != nil {
		return ``, fmt.Errorf(`marshalling metadata failed - %v`, err)
	}

	if _, err = skt.SendMessage([][]byte{metaByts, data}); err != nil {
		return ``, fmt.Errorf(`sending zmq message by sender failed - %v`, err)
	}

receive:
	msgs, err := skt.RecvMessage(0)
	if err != nil {
		if err.Error() == errTempUnavail {
			goto receive
		}
		return ``, fmt.Errorf(`receiving zmq message by sender failed - %v`, err)
	}

	if len(msgs) == 0 {
		return ``, fmt.Errorf(`received an empty message`)
	}
	
	if msgs[0] == failedRes {
		return ``, fmt.Errorf(`received an error message`)
	}

	return msgs[0], nil
}

func (c *Client) socket(endpoint string) (skt *zmq.Socket, err error) {
	skt, ok := c.peers[endpoint]
	if ok {
		return skt, nil
	}

	skt, err = c.ctx.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fmt.Errorf(`creating new socket for endpoint %s failed - %v`, endpoint, err)
	}

	if err = skt.Connect(endpoint); err != nil {
		return nil, fmt.Errorf(`connecting to zmq socket (%s) failed - %v`, endpoint, err)
	}

	c.peers[endpoint] = skt
	return skt, nil
}
