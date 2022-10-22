package prober

import (
	"encoding/json"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/tryfix/log"
)

type recipient struct {
	id        string
	endpoint  string
	publicKey []byte
}

type Prober struct {
	rec         *recipient
	transporter domain.Transporter
	enc         domain.Packer
	km          *crypto.KeyManager
	logger      log.Logger
}

func NewProber(t domain.Transporter, enc domain.Packer, km *crypto.KeyManager, logger log.Logger) (p *Prober, err error) {
	return &Prober{
		km:          km,
		transporter: t,
		enc:         enc,
		logger:      logger,
	}, nil
}

func (p *Prober) PublicKey() []byte {
	return p.km.PublicKey()
}

func (p *Prober) SetRecipient(name, endpoint string, key []byte) {
	p.rec = &recipient{id: name, endpoint: endpoint, publicKey: key}
}

// generate conn req - include peer did, did doc
// encrypt using rec keys

func (p *Prober) Connect() {

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
