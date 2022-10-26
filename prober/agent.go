package prober

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

type Prober struct {
	did         string
	didDoc      domain.DIDDocument
	invEndpoint string
	state       string
	inChan      chan []byte
	outChan     chan string
	ks          domain.KeyService // single key-pair for now
	tr          domain.Transporter
	packer      domain.Packer
	ds          domain.DIDService
	oob         domain.OOBService
	log         log.Logger
	label       string
	peers       map[string]domain.Peer
}

func NewProber(c *domain.Container) (p *Prober, err error) {
	p = &Prober{
		invEndpoint: c.Cfg.InvEndpoint,
		ks:          c.KS,
		tr:          c.Tr,
		packer:      c.Packer,
		log:         c.Log,
		ds:          c.DS,
		oob:         c.OOB,
		inChan:      c.InChan,
		outChan:     c.OutChan,
		state:       stateInitial,
		label:       c.Cfg.Name,
		peers:       map[string]domain.Peer{}, // name as the key may not be ideal
	}

	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, p.ks.PublicKey())
	// removes redundant elements from the allocated byte slice
	encodedKey = bytes.Trim(encodedKey, "\x00")

	// creating own did and did doc
	didDoc := p.ds.CreateDIDDoc(c.Cfg.ExchangeEndpoint, `message-service`, encodedKey)
	dd, err := p.ds.CreatePeerDID(didDoc)
	if err != nil {
		return nil, fmt.Errorf(`creating peer did failed - %v`, err)
	}

	p.did = dd
	p.didDoc = didDoc
	return p, nil
}

func (p *Prober) Listen() {
	for {
		data := <-p.inChan
		switch p.state {
		case stateInitial:
			p.log.Error(`no invitation has been processed yet`)
		case stateInvSent:
			if err := p.ProcessConnReq(data); err != nil {
				p.log.Error(err)
			}
		case stateReqSent:
			if err := p.ProcessConnRes(data); err != nil {
				p.log.Error(err)
			}
		case stateConnected:
			if err := p.ReadMessage(data); err != nil {
				p.log.Error(err)
			}
		}
	}
}

func (p *Prober) PublicKey() []byte {
	return p.ks.PublicKey()
}

func (p *Prober) Invite() (url string, err error) {
	if err = p.ks.GenerateInvKeys(); err != nil {
		return ``, fmt.Errorf(`generating invitation keys failed - %v`, err)
	}

	// encoding invitation public key
	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, p.ks.InvPublicKey())
	// removes redundant elements from the allocated byte slice
	encodedKey = bytes.Trim(encodedKey, "\x00")

	// creates a did doc for connection request with a separate endpoint and public key
	invDidDoc := p.ds.CreateDIDDoc(p.invEndpoint, `did-exchange`, encodedKey)

	// but uses did created from default did doc as it serves as the identifier in invitation
	url, err = p.oob.CreateInv(p.label, p.did, invDidDoc)
	if err != nil {
		return ``, fmt.Errorf(`creating invitation failed - %v`, err)
	}

	p.state = stateInvSent
	return url, nil
}

// Accept creates a connection request and sends it to the invitation endpoint
func (p *Prober) Accept(encodedInv string) error {
	inv, invEndpoint, peerInvPubKey, err := p.oob.ParseInv(encodedInv)
	if err != nil {
		return fmt.Errorf(`parsing invitation failed - %v`, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDoc)
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDoc, err := p.packer.Pack(docBytes, peerInvPubKey, p.ks.PublicKey(), p.ks.PrivateKey())
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	// creates connection request
	connReq, err := p.ds.CreateConnReq(p.label, inv.Id, p.did, encDoc)
	if err != nil {
		return fmt.Errorf(`creating connection request failed - %v`, err)
	}

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if err = p.tr.Send(connReqBytes, invEndpoint); err != nil {
		return fmt.Errorf(`sending connection request failed - %v`, err)
	}

	p.state = stateReqSent
	p.peers[inv.Label] = domain.Peer{DID: inv.From, ExchangeThId: inv.Id}
	return nil
}

// ProcessConnReq parses the connection request, creates a connection response and sends it to did endpoint
func (p *Prober) ProcessConnReq(data []byte) error {
	peerLabel, pthId, peerDid, peerEncDocBytes, err := p.ds.ParseConnReq(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with invitation keys
	peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, p.ks.InvPublicKey(), p.ks.InvPrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	// marshals own did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDoc)
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDidDoc, err := p.packer.Pack(docBytes, peerPubKey, p.ks.PublicKey(), p.ks.PrivateKey())
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	connRes, err := p.ds.CreateConnRes(pthId, p.did, encDidDoc)
	if err != nil {
		return fmt.Errorf(`creating connection response failed - %v`, err)
	}

	connResBytes, err := json.Marshal(connRes)
	if err != nil {
		return fmt.Errorf(`marshalling connection response failed - %v`, err)
	}

	if err = p.tr.Send(connResBytes, peerEndpoint); err != nil {
		return fmt.Errorf(`sending connection response failed - %v`, err)
	}

	p.state = stateConnected
	p.peers[peerLabel] = domain.Peer{DID: peerDid, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}
	fmt.Printf("-> Connection established with %s\n", peerLabel)

	return nil
}

func (p *Prober) ProcessConnRes(data []byte) error {
	pthId, peerEncDocBytes, err := p.ds.ParseConnRes(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with default keys
	peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, p.ks.PublicKey(), p.ks.PrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	// todo send complete message

	p.state = stateConnected
	for name, peer := range p.peers {
		if peer.ExchangeThId == pthId {
			peerDid := peer.DID
			p.peers[name] = domain.Peer{DID: peerDid, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}
			fmt.Printf("-> Connection established with %s\n", name)
			return nil
		}
	}

	return fmt.Errorf(`requested peer is unknown to the agent`)
}

func (p *Prober) getPeerInfo(encDocBytes, recPubKey, recPrvKey []byte) (endpoint string, pubKey []byte, err error) {
	peerDocBytes, err := p.packer.Unpack(encDocBytes, recPubKey, recPrvKey)
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

func (p *Prober) SendMessage(to, text string) error {
	peer, ok := p.peers[to]
	if !ok {
		return fmt.Errorf(`no didcomm connection found for the recipient %s`, to)
	}

	msg, err := p.packer.Pack([]byte(text), peer.PubKey, p.ks.PublicKey(), p.ks.PrivateKey())
	if err != nil {
		p.log.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.log.Error(err)
		return err
	}

	err = p.tr.Send(data, peer.Endpoint)
	if err != nil {
		p.log.Error(err)
		return err
	}

	fmt.Printf("-> Message sent\n")
	return nil
}

func (p *Prober) ReadMessage(data []byte) error {
	textBytes, err := p.packer.Unpack(data, p.ks.PublicKey(), p.ks.PrivateKey())
	if err != nil {
		return fmt.Errorf(`unpacking message failed - %v`, err)
	}
	p.outChan <- string(textBytes)
	return nil
}
