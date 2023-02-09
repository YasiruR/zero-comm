package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/klauspost/compress/zstd"
)

type compactor struct {
	zEncodr *zstd.Encoder
	zDecodr *zstd.Decoder
}

func newCompactor() (*compactor, error) {
	zstdEncoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf(`creating zstd encoder failed - %v`, err)
	}

	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf(`creating zstd decoder failed - %v`, err)
	}

	return &compactor{zEncodr: zstdEncoder, zDecodr: zstdDecoder}, nil
}

// packer is an internal wrapper for the packing process of didcomm messages
type packer struct {
	*services
	pckr servicesPkg.Packer
}

func newPacker(c *domain.Container) *packer {
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
		s, _, err := p.serviceInfo(receiver)
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

func (p *packer) serviceInfo(peer string) (*models.Service, *models.Peer, error) {
	pr, err := p.probr.Peer(peer)
	if err != nil {
		return nil, nil, fmt.Errorf(`no peer found - %v`, err)
	}

	var srvc *models.Service
	for _, s := range pr.Services {
		if s.Type == domain.ServcGroupJoin {
			srvc = &s
			break
		}
	}

	if srvc.Type == `` {
		return nil, nil, fmt.Errorf(`requested service (%s) is not by the peer`, domain.ServcGroupJoin)
	}

	return srvc, &pr, nil
}
