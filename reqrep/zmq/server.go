package zmq

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

type handler struct {
	async    bool
	notifier chan models.Message
}

type Server struct {
	skt      *zmq.Socket
	handlers map[string]*handler
	log      log.Logger
}

func NewServer(zmqCtx *zmq.Context, c *domain.Container) (*Server, error) {
	skt, err := zmqCtx.NewSocket(zmq.REP)
	if err != nil {
		return nil, fmt.Errorf(`constructing zmq server socket failed - %v`, err)
	}

	if err = skt.Bind(c.Cfg.InvEndpoint); err != nil {
		return nil, fmt.Errorf(`binding zmq socket to %s failed - %v`, c.Cfg.InvEndpoint, err)
	}

	return &Server{
		skt:      skt,
		handlers: make(map[string]*handler),
		log:      c.Log,
	}, nil
}

func (s *Server) AddHandler(msgType string, notifier chan models.Message, async bool) {
	s.handlers[msgType] = &handler{async: async, notifier: notifier}
}

func (s *Server) RemoveHandler(msgType string) {
	delete(s.handlers, msgType)
}

func (s *Server) Start() error {
	for {
		msg, err := s.skt.RecvMessage(0)
		if err != nil {
			if err.Error() != errTempUnavail {
				s.log.Error(fmt.Sprintf(`receiving zmq message by receiver failed - %v`, err))
			}
			s.sendAck(false)
			continue
		}

		if len(msg) != 2 {
			s.log.Error(`received an empty/invalid message`, msg)
			s.sendAck(false)
			continue
		}

		m := models.Message{Type: msg[0], Data: []byte(msg[1])}
		h, ok := s.handlers[m.Type]
		if !ok {
			s.log.Error(fmt.Sprintf(`no handler defined for the received message type (%s)`, m.Type))
			s.sendAck(false)
			continue
		}

		if h.async {
			h.notifier <- m
			s.sendAck(true)
			continue
		}

		m.Reply = make(chan []byte)
		h.notifier <- m
		rep := <-m.Reply
		s.sendRes(rep) // todo remove rep and check
	}
}

func (s *Server) sendAck(success bool) {
	msg := `ok`
	if !success {
		msg = `failed`
	}

	if _, err := s.skt.Send(msg, 0); err != nil {
		s.log.Error(fmt.Sprintf(`sending zmq ack message by receiver failed - %v`, err))
	}
}

func (s *Server) sendRes(data []byte) {
	if _, err := s.skt.Send(string(data), 0); err != nil {
		s.log.Error(fmt.Sprintf(`sending zmq response message by receiver failed - %v`, err))
	}
}

func (s *Server) Stop() error {
	return nil
}
