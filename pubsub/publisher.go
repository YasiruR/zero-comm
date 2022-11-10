package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/prober"
	zmq "github.com/pebbe/zmq4"
)

type Publisher struct {
	ctx *zmq.Context
	skt *zmq.Socket
	prb *prober.Prober
	ks  domain.KeyService

	peerKeyMap   map[string][]string
	topicPeerMap map[string]map[string][]string
}

func NewPublisher() {
	// init pub struct
}

func (p *Publisher) Register(topic string) error {
	// can add topic details later to the invitation
	inv, err := p.prb.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	status := messages.PublisherStatus{Active: true, Inv: inv}
	byts, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf(`marshalling publisher status failed - %v`, err)
	}

	if _, err = p.skt.SendMessage(fmt.Sprintf(`%s %s`, topic, string(byts))); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
	}

	return nil
}

func (p *Publisher) addDIDConn() {

}

func (p *Publisher) AddSubscription() {

}

func (p *Publisher) Publish() {

}

func (p *Publisher) Close() {

}

// create a pub socket for topic_pubs
// generate one invitation for the topic
// publish status with inv to topic_pubs

// prober receives multiple conn requests
// for each request, establish didcomm conn - auto
