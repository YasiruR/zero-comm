package zmq

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"sync"
)

type req struct {
	typ      models.MsgType
	data     []byte
	endpoint string
	resChan  chan res
}

type res struct {
	msg string
	err error
}

type Client struct {
	ctx     *zmq.Context
	chanMap *sync.Map
	log     log.Logger
}

func NewClient(zmqCtx *zmq.Context, log log.Logger) *Client {
	return &Client{
		ctx:     zmqCtx,
		chanMap: &sync.Map{},
		log:     log,
	}
}

// Send connects to the endpoint per each message since it is more appropriate8
// with DIDComm as by nature it manifests an asynchronous simplex communication.
func (c *Client) Send(typ models.MsgType, data []byte, endpoint string) (response string, err error) {
	inChan, ok := c.sendr(endpoint)
	if !ok {
		inChan = make(chan req)
		go c.initSendr(endpoint, inChan)
	}

	resChan := make(chan res)
	inChan <- req{typ: typ, data: data, endpoint: endpoint, resChan: resChan}
	resMsg := <-resChan
	if resMsg.err != nil {
		return ``, fmt.Errorf(`send error - %v`, resMsg.err)
	}

	return resMsg.msg, nil
}

func (c *Client) sendr(endpoint string) (inChan chan req, ok bool) {
	val, ok := c.chanMap.Load(endpoint)
	if !ok {
		return nil, false
	}

	inChan, ok = val.(chan req)
	if !ok {
		return nil, false
	}

	return inChan, true
}

func (c *Client) initSendr(endpoint string, inChan chan req) {
	c.chanMap.Store(endpoint, inChan)
	skt, err := c.ctx.NewSocket(zmq.REQ)
	if err != nil {
		c.log.Fatal(fmt.Sprintf(`creating new socket for endpoint %s failed - %v`, endpoint, err))
	}

	if err = skt.Connect(endpoint); err != nil {
		c.log.Fatal(fmt.Sprintf(`connecting to zmq socket (%s) failed - %v`, endpoint, err))
	}

	for {
		reqMsg := <-inChan
		if string(reqMsg.data) == domain.MsgTerminate {
			return
		}

		metaByts, err := json.Marshal(metadata{Type: int(reqMsg.typ)})
		if err != nil {
			reqMsg.resChan <- res{msg: ``, err: fmt.Errorf(`marshalling metadata failed - %v`, err)}
			continue
		}

		if _, err = skt.SendMessage([][]byte{metaByts, reqMsg.data}); err != nil {
			reqMsg.resChan <- res{msg: ``, err: fmt.Errorf(`sending zmq message by sender failed - %v`, err)}
			continue
		}

	receive:
		resMsgs, err := skt.RecvMessage(0)
		if err != nil {
			if err.Error() == errTempUnavail {
				goto receive
			}
			reqMsg.resChan <- res{msg: ``, err: fmt.Errorf(`receiving zmq message by sender failed - %v`, err)}
			continue
		}

		if len(resMsgs) == 0 {
			reqMsg.resChan <- res{msg: ``, err: fmt.Errorf(`received an empty message`)}
			continue
		}

		if resMsgs[0] == failedRes {
			reqMsg.resChan <- res{msg: ``, err: fmt.Errorf(`received an error message`)}
			continue
		}

		reqMsg.resChan <- res{msg: resMsgs[0], err: nil}
	}
}

func (c *Client) Close() error {
	c.chanMap.Range(func(key, val any) bool {
		inChan, ok := val.(chan req)
		if !ok {
			return true
		}

		inChan <- req{data: []byte(domain.MsgTerminate)}
		return true
	})

	return nil
}
