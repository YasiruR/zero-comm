package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/klauspost/compress/zstd"
)

// compactor is an instance of zstandard compression algorithm
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

// packer is an internal wrapper for the packing processes of group agent
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
