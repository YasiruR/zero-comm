package prober

import (
	"encoding/base64"
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

	p.rec = &recipient{id: name, endpoint: endpoint, publicKey: key}
	return nil
}

//func (p *Prober) SetRecipientWithDoc(doc domain.DIDDocument) error {
//	if len(doc.Service) == 0 {
//		return errors.New(`DID document does not contain any service endpoints`)
//	}
//
//	service := doc.Service[0]
//	if len(service.RoutingKeys) == 0 {
//		return errors.New(`DID document does not contain any routing keys`)
//	}
//
//	key, err := base64.StdEncoding.DecodeString(service.RoutingKeys[0])
//	if err != nil {
//		p.logger.Error(err)
//		return err
//	}
//
//	p.rec = &recipient{id: service.Id, endpoint: service.ServiceEndpoint, publicKey: key}
//	return nil
//}

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
