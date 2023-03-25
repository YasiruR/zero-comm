package transport

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/tryfix/log"
	"sync"
)

// Peers stores the list of connected subscribers in zmq level
type Peers struct {
	endpoints *sync.Map
	probr     services.Agent
	log       log.Logger
}

func InitPeerStore(c *container.Container) *Peers {
	ackChan := make(chan models.Message)
	c.Server.AddHandler(models.TypStatusAck, ackChan, true)
	p := &Peers{endpoints: &sync.Map{}, probr: c.Prober, log: c.Log}
	p.listen(ackChan)

	return p
}

func (p *Peers) Connected(endpoint string) bool {
	_, ok := p.endpoints.Load(endpoint)
	return ok
}

func (p *Peers) ConnectedList() (endpoints []string) {
	p.endpoints.Range(func(key, val any) bool {
		endpoint, ok := key.(string)
		if ok {
			endpoints = append(endpoints, endpoint)
		}
		return true
	})

	return endpoints
}

func (p *Peers) listen(ackChan chan models.Message) {
	go func() {
		for {
			msg := <-ackChan
			_, endpoint, err := p.probr.ReadMessage(models.Message{Type: models.TypStatusAck, Data: msg.Data})
			if err != nil {
				log.Error(fmt.Sprintf(`reading status ack failed - %v`, err))
				continue
			}
			p.endpoints.Store(endpoint, true)
			fmt.Println("PEER MSG RECEIVED AND STORED", endpoint)
		}
	}()
}
