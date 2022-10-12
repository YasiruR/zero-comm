package prober

import (
	"encoding/base64"
	"encoding/json"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/tryfix/log"
)

type recipient struct {
	name      string
	endpoint  string
	publicKey []byte
}

type Prober struct {
	rec         *recipient
	transporter domain.Transporter
	enc         *crypto.Packer
	km          *crypto.KeyManager
	logger      log.Logger
}

func NewProber(t domain.Transporter, enc *crypto.Packer, km *crypto.KeyManager, logger log.Logger) (p *Prober, err error) {
	return &Prober{
		km:          km,
		transporter: t,
		enc:         enc,
		logger:      logger,
	}, nil
}

func (p *Prober) SetRecipient(name, endpoint string, encodedKey string) error {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	p.rec = &recipient{name: name, endpoint: endpoint, publicKey: key}
	return nil
}

func (p *Prober) Send(text string) error {
	msg, err := p.enc.Pack(text, p.rec.publicKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		p.logger.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	err = p.transporter.Send(data, p.rec.endpoint)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	return nil
}
