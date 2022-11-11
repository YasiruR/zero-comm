package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/btcsuite/btcutil/base58"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
)

type Publisher struct {
	label       string
	skt         *zmq.Socket
	prb         domain.DIDCommService
	ks          domain.KeyService
	packer      domain.Packer
	log         log.Logger
	subChan     chan domain.Message
	topicSubMap map[string]map[string][]byte // topic to subscriber to pub key map - use sync map, can extend to multiple keys per peer
}

// todo add publisher as a service endpoint in did doc

func NewPublisher(zmqCtx *zmq.Context, c *domain.Container) (*Publisher, error) {
	skt, err := zmqCtx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	if err = skt.Bind(c.Cfg.PubEndpoint); err != nil {
		return nil, fmt.Errorf(`binding zmq pub socket to %s failed - %v`, c.Cfg.PubEndpoint, err)
	}

	p := &Publisher{
		label:       c.Cfg.Args.Name,
		skt:         skt,
		prb:         c.Prober,
		ks:          c.KS,
		packer:      c.Packer,
		log:         c.Log,
		subChan:     c.SubChan,
		topicSubMap: map[string]map[string][]byte{},
	}

	go p.initAddSubs()
	return p, err
}

func (p *Publisher) Register(topic string) error {
	// can add topic details later to the invitation
	inv, err := p.prb.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	//fmt.Println("INV in pub : ", inv)

	status := messages.PublisherStatus{Active: true, Inv: inv}
	byts, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf(`marshalling publisher status failed - %v`, err)
	}

	if _, err = p.skt.SendMessage(fmt.Sprintf(`%s_pubs %s`, topic, string(byts))); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
	}

	return nil
}

// initAddSubs follows subscription of a topic which is done through a separate
// DIDComm message. Alternatively, it can be included in connection request.
func (p *Publisher) initAddSubs() {
	for {
		// add termination
		msg := <-p.subChan
		unpackedMsg, err := p.prb.ReadMessage(msg.Data)
		if err != nil {
			p.log.Error(fmt.Sprintf(`reading subscribe msg failed - %v`, err))
			continue
		}

		//fmt.Println("\nMSG DATA: ", string(msg.Data))
		fmt.Println("UNPACKED SUB MSG: ", unpackedMsg)

		var sub messages.SubscribeMsg
		if err = json.Unmarshal([]byte(unpackedMsg), &sub); err != nil {
			p.log.Error(fmt.Sprintf(`unmarshalling subscribe message failed - %v`, err))
			continue
		}

		subKey := base58.Decode(sub.PubKey)
		for _, t := range sub.Topics {
			if p.topicSubMap[t] == nil {
				p.topicSubMap[t] = map[string][]byte{}
			}
			p.topicSubMap[t][sub.Peer] = subKey
		}
	}
}

func (p *Publisher) Publish(topic, msg string) error {
	fmt.Println("PUBLISH 1: ", p.topicSubMap[topic])
	for sub, subKey := range p.topicSubMap[topic] {
		ownPubKey, err := p.ks.PublicKey(sub)
		if err != nil {
			return fmt.Errorf(`getting public key for connection with %s failed - %v`, sub, err)
		}

		ownPrvKey, err := p.ks.PrivateKey(sub)
		if err != nil {
			return fmt.Errorf(`getting private key for connection with %s failed - %v`, sub, err)
		}

		encryptdMsg, err := p.packer.Pack([]byte(msg), subKey, ownPubKey, ownPrvKey)
		if err != nil {
			p.log.Error(err)
			return err
		}

		data, err := json.Marshal(encryptdMsg)
		if err != nil {
			p.log.Error(err)
			return err
		}

		if _, err = p.skt.SendMessage(fmt.Sprintf(`%s_%s_%s %s`, topic, p.label, sub, string(data))); err != nil {
			return fmt.Errorf(`publishing message (%s) failed for %s - %v`, msg, sub, err)
		}
	}

	return nil
}

func (p *Publisher) Close() error {
	// publish inactive to all _pubs
	// close sockets
	// close context

	return nil
}
