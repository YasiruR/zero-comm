package prober

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/did"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/tryfix/log"
)

type connection struct {
	peerDID      string
	peerEndpoint string
	peerPubKey   []byte
}

type Prober struct {
	did    string
	didDoc domain.DIDDocument

	rec       *connection // single connection for now
	transport domain.Transporter
	enc       domain.Packer
	km        *crypto.KeyManager // single key-pair for now
	logger    log.Logger

	invEndpoint string
	dh          *did.Handler
	oob         *did.OOBService
}

func NewProber(cfg *domain.Config, dh *did.Handler, t domain.Transporter, enc domain.Packer, km *crypto.KeyManager, logger log.Logger) (p *Prober, err error) {
	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, p.km.PublicKey())
	// removes redundant elements from the allocated byte slice
	encodedKey = bytes.Trim(encodedKey, "\x00")

	// creating own did and did doc
	didDoc := dh.CreateDIDDoc(cfg.Hostname+domain.ExchangeEndpoint, `message-service`, encodedKey)
	dd, err := dh.CreatePeerDID(didDoc)
	if err != nil {
		return nil, fmt.Errorf(`creating peer did failed - %v`, err)
	}

	return &Prober{
		did:       dd,
		didDoc:    didDoc,
		km:        km,
		transport: t,
		enc:       enc,
		logger:    logger,
	}, nil
}

func (p *Prober) PublicKey() []byte {
	return p.km.PublicKey()
}

func (p *Prober) PeerDID() string {
	return p.did
}

func (p *Prober) DIDDoc() domain.DIDDocument {
	return p.didDoc
}

func (p *Prober) SetRecipient(name, endpoint string, key []byte) {
	p.rec = &connection{peerDID: name, peerEndpoint: endpoint, peerPubKey: key}
}

func (p *Prober) GenerateInv() (url string, err error) {
	if err = p.km.GenerateInvKeys(); err != nil {
		return ``, fmt.Errorf(`generating invitation keys failed - %v`, err)
	}

	// encoding invitation public key
	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, p.km.InvPublicKey())
	// removes redundant elements from the allocated byte slice
	encodedKey = bytes.Trim(encodedKey, "\x00")

	// creates a did doc for connection request with a separate endpoint and public key
	invDidDoc := p.dh.CreateDIDDoc(p.invEndpoint, `did-exchange`, encodedKey)

	// but uses did created from default did doc as it serves as the identifier in invitation
	return p.oob.CreateInvitation(p.did, invDidDoc)
}

func (p *Prober) ProcessInv(encodedInv string) error {
	inv, invEndpoint, peerInvPubKey, err := p.oob.ParseInvitation(encodedInv)
	if err != nil {
		return fmt.Errorf(`parsing invitation failed - %v`, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDoc)
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encryptedDoc, err := p.enc.Pack(docBytes, peerInvPubKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	// marshals the encrypted did doc
	encryptedDocBytes, err := json.Marshal(encryptedDoc)
	if err != nil {
		return fmt.Errorf(`marshalling encrypted did doc failed - %v`, err)
	}

	// creates connection request
	connReq := p.dh.CreateConnReq(`test-label`, inv.Id, inv.From, encryptedDocBytes)

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if err = p.transport.Send(connReqBytes, invEndpoint); err != nil {
		return fmt.Errorf(`sending connection request failed - %v`, err)
	}
}

// generate conn req - include peer did, did doc
// encrypt using rec keys

func (p *Prober) Connect() {

}

func (p *Prober) SendMessage(text string) error {
	msg, err := p.enc.Pack([]byte(text), p.rec.peerPubKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		p.logger.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	err = p.transport.Send(data, p.rec.peerEndpoint)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	return nil
}
