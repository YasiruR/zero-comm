package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/container"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
)

// packer is an internal wrapper for the packing processes of group agent
type packer struct {
	*services
	pckr servicesPkg.Packer
}

func newPacker(c *container.Container) *packer {
	return &packer{
		services: &services{
			km:    c.KeyManager,
			probr: c.Prober,
		},
		pckr: c.Packer,
	}
}

// pack constructs and encodes an authcrypt message to the given receiver
func (p *packer) pack(receiver string, recPubKey []byte, msg []byte) ([]byte, error) {
	if recPubKey == nil {
		s, err := p.probr.Service(domain.ServcGroupJoin, receiver)
		if err != nil {
			return nil, fmt.Errorf(`fetching service info failed for peer %s - %v`, receiver, err)
		}
		recPubKey = s.PubKey
	}

	ownPubKey, err := p.km.PublicKey(receiver)
	if err != nil {
		return nil, fmt.Errorf(`getting public key for connection with %s failed - %v`, receiver, err)
	}

	ownPrvKey, err := p.km.PrivateKey(receiver)
	if err != nil {
		return nil, fmt.Errorf(`getting private key for connection with %s failed - %v`, receiver, err)
	}

	encryptdMsg, err := p.pckr.Pack(msg, recPubKey, ownPubKey, ownPrvKey)
	if err != nil {
		return nil, fmt.Errorf(`packing error - %v`, err)
	}

	data, err := json.Marshal(encryptdMsg)
	if err != nil {
		return nil, fmt.Errorf(`marshalling packed message failed - %v`, err)
	}

	return data, nil
}
