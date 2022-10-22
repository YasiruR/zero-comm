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

const (
	stateInitial   = `initial`
	stateInvSent   = `invitation-sent`
	stateReqSent   = `request-sent`
	stateResSent   = `response-sent`
	stateConnected = `connected` // ideally after complete message is sent and received
)

type connection struct {
	peerDID      string
	peerEndpoint string
	peerPubKey   []byte
}

type Prober struct {
	did    string
	didDoc domain.DIDDocument

	conn      *connection // single connection for now
	transport domain.Transporter
	enc       domain.Packer
	km        *crypto.KeyManager // single key-pair for now
	logger    log.Logger

	invEndpoint string
	dh          *did.Handler
	oob         *did.OOBService
	state       string
	inChan      chan []byte
	outChan     chan string
}

// todo container for services

func NewProber(cfg *domain.Config, dh *did.Handler, oob *did.OOBService, t domain.Transporter, enc domain.Packer, km *crypto.KeyManager, inChan chan []byte, outChan chan string, logger log.Logger) (p *Prober, err error) {
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
		invEndpoint: cfg.Hostname + domain.InvitationEndpoint,
		did:         dd,
		didDoc:      didDoc,
		km:          km,
		transport:   t,
		enc:         enc,
		logger:      logger,
		dh:          dh,
		oob:         oob,
		inChan:      inChan,
		outChan:     outChan,
		state:       stateInitial,
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
	p.conn = &connection{peerDID: name, peerEndpoint: endpoint, peerPubKey: key}
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
	url, err = p.oob.CreateInvitation(p.did, invDidDoc)
	if err != nil {
		return ``, fmt.Errorf(`creating invitation failed - %v`, err)
	}

	p.state = stateInvSent
	return url, nil
}

func (p *Prober) Listen() {
	for {
		data := <-p.inChan
		switch p.state {
		case stateInitial:
			p.logger.Error(`no invitation has been processed yet`)
		case stateInvSent:
			if err := p.ProcessConnReq(data); err != nil {
				p.logger.Error(err)
			}
		case stateReqSent:
			if err := p.ProcessConnRes(data); err != nil {
				p.logger.Error(err)
			}
		case stateConnected:
			if err := p.ReadMessage(data); err != nil {
				p.logger.Error(err)
			}
		}
	}
}

// ProcessInv creates a connection request and sends it to the invitation endpoint
func (p *Prober) ProcessInv(encodedInv string) (from string, err error) {
	inv, invEndpoint, peerInvPubKey, err := p.oob.ParseInvitation(encodedInv)
	if err != nil {
		return ``, fmt.Errorf(`parsing invitation failed - %v`, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDoc)
	if err != nil {
		return ``, fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDoc, err := p.enc.Pack(docBytes, peerInvPubKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		return ``, fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	// creates connection request
	connReq, err := p.dh.CreateConnReq(inv.Id, p.did, encDoc)
	if err != nil {
		return ``, fmt.Errorf(`creating connection request failed - %v`, err)
	}

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return ``, fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if err = p.transport.Send(connReqBytes, invEndpoint); err != nil {
		return ``, fmt.Errorf(`sending connection request failed - %v`, err)
	}

	p.state = stateReqSent
	return inv.From, nil
}

// ProcessConnReq parses the connection request, creates a connection response and sends it to did endpoint
func (p *Prober) ProcessConnReq(data []byte) error {
	thId, peerEncDocBytes, err := p.dh.ParseConnReq(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with invitation keys
	peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, p.km.InvPublicKey(), p.km.InvPrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	// marshals own did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDoc)
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDidDoc, err := p.enc.Pack(docBytes, peerPubKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	connRes, err := p.dh.CreateConnRes(thId, p.did, encDidDoc)
	if err != nil {
		return fmt.Errorf(`creating connection response failed - %v`, err)
	}

	connResBytes, err := json.Marshal(connRes)
	if err != nil {
		return fmt.Errorf(`marshalling connection response failed - %v`, err)
	}

	if err = p.transport.Send(connResBytes, peerEndpoint); err != nil {
		return fmt.Errorf(`sending connection response failed - %v`, err)
	}

	p.conn = &connection{peerEndpoint: peerEndpoint, peerPubKey: peerPubKey}
	p.state = stateConnected
	return nil
}

func (p *Prober) ProcessConnRes(data []byte) error {
	_, peerEncDocBytes, err := p.dh.ParseConnRes(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with default keys
	peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	// todo send complete message

	p.conn = &connection{peerEndpoint: peerEndpoint, peerPubKey: peerPubKey}
	p.state = stateConnected
	return nil
}

func (p *Prober) getPeerInfo(encDocBytes, recPubKey, recPrvKey []byte) (endpoint string, pubKey []byte, err error) {
	peerDocBytes, err := p.enc.Unpack(encDocBytes, recPubKey, recPrvKey)
	if err != nil {
		return ``, nil, fmt.Errorf(`decrypting did doc failed - %v`, err)
	}

	// unmarshalls decrypted did doc
	var peerDidDoc domain.DIDDocument
	if err = json.Unmarshal(peerDocBytes, &peerDidDoc); err != nil {
		return ``, nil, fmt.Errorf(`unmarshalling decrypted did doc failed - %v`, err)
	}

	if len(peerDidDoc.Service) == 0 {
		return ``, nil, fmt.Errorf(`did doc does not contain a service`)
	}

	// assumes first service is the valid one
	if len(peerDidDoc.Service[0].RecipientKeys) == 0 {
		return ``, nil, fmt.Errorf(`did doc does not contain recipient keys for the service`)
	}

	peerEndpoint := peerDidDoc.Service[0].ServiceEndpoint
	peerPubKey, err := base64.StdEncoding.DecodeString(peerDidDoc.Service[0].RecipientKeys[0])
	if err != nil {
		return ``, nil, fmt.Errorf(`decoding recipient key failed - %v`, err)
	}

	return peerEndpoint, peerPubKey, nil
}

func (p *Prober) SendMessage(text string) error {
	msg, err := p.enc.Pack([]byte(text), p.conn.peerPubKey, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		p.logger.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	err = p.transport.Send(data, p.conn.peerEndpoint)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	return nil
}

func (p *Prober) ReadMessage(data []byte) error {
	textBytes, err := p.enc.Unpack(data, p.km.PublicKey(), p.km.PrivateKey())
	if err != nil {
		return fmt.Errorf(`unpacking message failed - %v`, err)
	}
	p.outChan <- string(textBytes)
	return nil
}
