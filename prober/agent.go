package prober

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tryfix/log"
)

type Prober struct {
	invEndpoint  string
	exchEndpoint string
	inChan       chan domain.Message
	outChan      chan string
	ks           domain.KeyService // single key-pair for now
	tr           domain.Transporter
	packer       domain.Packer
	ds           domain.DIDService
	oob          domain.OOBService
	log          log.Logger
	label        string
	peers        map[string]domain.Peer
	didDocs      map[string]messages.DIDDocument
	dids         map[string]string

	connDone chan domain.Connection
}

func NewProber(c *domain.Container) (p *Prober, err error) {
	p = &Prober{
		invEndpoint:  c.Cfg.InvEndpoint,
		exchEndpoint: c.Cfg.InvEndpoint,
		ks:           c.KS,
		tr:           c.Tr,
		packer:       c.Packer,
		log:          c.Log,
		ds:           c.DS,
		oob:          c.OOB,
		inChan:       c.InChan,
		outChan:      c.OutChan,
		label:        c.Cfg.Args.Name,
		peers:        map[string]domain.Peer{}, // name as the key may not be ideal
		didDocs:      map[string]messages.DIDDocument{},
		dids:         map[string]string{},
		connDone:     c.ConnDoneChan,
	}

	go p.Listen()
	return p, nil
}

func (p *Prober) Listen() {
	// can introduce concurrency
	for {
		chanMsg := <-p.inChan
		switch chanMsg.Type {
		case domain.MsgTypConnReq:
			if err := p.processConnReq(chanMsg.Data); err != nil {
				p.log.Error(err)
			}
		case domain.MsgTypConnRes:
			if err := p.processConnRes(chanMsg.Data); err != nil {
				p.log.Error(err)
			}
		case domain.MsgTypData:
			if _, err := p.ReadMessage(chanMsg.Data); err != nil {
				p.log.Error(err)
			}
		}
	}
}

func (p *Prober) Invite() (url string, err error) {
	if err = p.ks.GenerateInvKeys(); err != nil {
		return ``, fmt.Errorf(`generating invitation keys failed - %v`, err)
	}

	// creates a did doc for connection request with a separate endpoint and public key
	invDidDoc := p.ds.CreateDIDDoc(p.invEndpoint, `did-exchange`, p.ks.InvPublicKey())

	// but uses did created from default did doc as it serves as the identifier in invitation
	url, err = p.oob.CreateInv(p.label, ``, invDidDoc) // todo null did
	if err != nil {
		return ``, fmt.Errorf(`creating invitation failed - %v`, err)
	}

	return url, nil
}

// Accept creates a connection request and sends it to the invitation endpoint
func (p *Prober) Accept(encodedInv string) error {
	inv, invEndpoint, peerInvPubKey, err := p.oob.ParseInv(encodedInv)
	if err != nil {
		return fmt.Errorf(`parsing invitation failed - %v`, err)
	}

	// set up prerequisites for a connection (diddoc, did, keys)
	pubKey, prvKey, err := p.setConnPrereqs(inv.Label)
	if err != nil {
		return fmt.Errorf(`setting up prerequisites for connection with %s failed - %v`, inv.Label, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDocs[inv.Label])
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDoc, err := p.packer.Pack(docBytes, peerInvPubKey, pubKey, prvKey)
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	// creates connection request
	connReq, err := p.ds.CreateConnReq(p.label, inv.Id, p.dids[inv.Label], encDoc)
	if err != nil {
		return fmt.Errorf(`creating connection request failed - %v`, err)
	}

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if err = p.tr.Send(domain.MsgTypConnReq, connReqBytes, invEndpoint); err != nil {
		return fmt.Errorf(`sending connection request failed - %v`, err)
	}

	p.peers[inv.Label] = domain.Peer{DID: inv.From, ExchangeThId: inv.Id}
	return nil
}

// processConnReq parses the connection request, creates a connection response and sends it to did endpoint
func (p *Prober) processConnReq(data []byte) error {
	peerLabel, pthId, peerDid, peerEncDocBytes, err := p.ds.ParseConnReq(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with invitation keys
	peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, p.ks.InvPublicKey(), p.ks.InvPrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	// set up prerequisites for a connection (diddoc, did, keys)
	pubKey, prvKey, err := p.setConnPrereqs(peerLabel)
	if err != nil {
		return fmt.Errorf(`setting up prerequisites for connection with %s failed - %v`, peerLabel, err)
	}

	// marshals own did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDocs[peerLabel])
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDidDoc, err := p.packer.Pack(docBytes, peerPubKey, pubKey, prvKey)
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	connRes, err := p.ds.CreateConnRes(pthId, p.dids[peerLabel], encDidDoc)
	if err != nil {
		return fmt.Errorf(`creating connection response failed - %v`, err)
	}

	connResBytes, err := json.Marshal(connRes)
	if err != nil {
		return fmt.Errorf(`marshalling connection response failed - %v`, err)
	}

	if err = p.tr.Send(domain.MsgTypConnRes, connResBytes, peerEndpoint); err != nil {
		return fmt.Errorf(`sending connection response failed - %v`, err)
	}

	p.peers[peerLabel] = domain.Peer{DID: peerDid, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}
	fmt.Printf("-> Connection established with %s\n", peerLabel)

	return nil
}

func (p *Prober) processConnRes(data []byte) error {
	pthId, peerEncDocBytes, err := p.ds.ParseConnRes(data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// todo send complete message

	for name, peer := range p.peers {
		if peer.ExchangeThId == pthId {
			ownPubKey, err := p.ks.PublicKey(name)
			if err != nil {
				return fmt.Errorf(`getting public key for connection with %s failed - %v`, name, err)
			}

			ownPrvKey, err := p.ks.PrivateKey(name)
			if err != nil {
				return fmt.Errorf(`getting private key for connection with %s failed - %v`, name, err)
			}

			// decrypts peer did doc which is encrypted with default keys
			peerEndpoint, peerPubKey, err := p.getPeerInfo(peerEncDocBytes, ownPubKey, ownPrvKey)
			if err != nil {
				return fmt.Errorf(`getting peer data failed - %v`, err)
			}

			p.peers[name] = domain.Peer{DID: peer.DID, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}

			if p.connDone != nil {
				p.connDone <- domain.Connection{Peer: name, PubKey: peerPubKey}
			}

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
	var peerDidDoc messages.DIDDocument
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

func (p *Prober) SendMessage(typ, to, text string) error {
	peer, ok := p.peers[to]
	if !ok {
		return fmt.Errorf(`no didcomm connection found for the recipient %s`, to)
	}

	ownPubKey, err := p.ks.PublicKey(to)
	if err != nil {
		return fmt.Errorf(`getting public key for connection with %s failed - %v`, to, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(to)
	if err != nil {
		return fmt.Errorf(`getting private key for connection with %s failed - %v`, to, err)
	}

	msg, err := p.packer.Pack([]byte(text), peer.PubKey, ownPubKey, ownPrvKey)
	if err != nil {
		p.log.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.log.Error(err)
		return err
	}

	err = p.tr.Send(typ, data, peer.Endpoint)
	if err != nil {
		p.log.Error(err)
		return err
	}

	fmt.Printf("-> Message sent\n")
	return nil
}

func (p *Prober) ReadMessage(data []byte) (msg string, err error) {
	peerName, err := p.peerByMsg(data)
	if err != nil {
		p.log.Debug(fmt.Sprintf(`getting peer info failed - %v`, err))
		return ``, nil
	}

	ownPubKey, err := p.ks.PublicKey(peerName)
	if err != nil {
		return ``, fmt.Errorf(`getting public key for connection with %s failed - %v`, peerName, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(peerName)
	if err != nil {
		return ``, fmt.Errorf(`getting private key for connection with %s failed - %v`, peerName, err)
	}

	textBytes, err := p.packer.Unpack(data, ownPubKey, ownPrvKey)
	if err != nil {
		return ``, fmt.Errorf(`unpacking message failed - %v`, err)
	}
	p.outChan <- string(textBytes)

	return string(textBytes), nil
}

func (p *Prober) setConnPrereqs(peer string) (pubKey, prvKey []byte, err error) {
	if err = p.ks.GenerateKeys(peer); err != nil {
		return nil, nil, fmt.Errorf(`generating keys failed - %v`, err)
	}

	// omitted errors since they should not occur
	pubKey, _ = p.ks.PublicKey(peer)
	prvKey, _ = p.ks.PrivateKey(peer)

	// creating own did and did doc
	didDoc := p.ds.CreateDIDDoc(p.exchEndpoint, `message-service`, pubKey)
	did, err := p.ds.CreatePeerDID(didDoc)
	if err != nil {
		return nil, nil, fmt.Errorf(`creating peer did failed - %v`, err)
	}

	p.didDocs[peer] = didDoc
	p.dids[peer] = did
	return pubKey, prvKey, nil
}

// can improve this since all this unmarshalling will be done again in unpack todo
// recipient[0] is hardcoded for now
func (p *Prober) peerByMsg(data []byte) (name string, err error) {
	// unmarshal into authcrypt message
	var msg messages.AuthCryptMsg
	err = json.Unmarshal(data, &msg)
	if err != nil {
		p.log.Error(err)
		return ``, err
	}

	// decode protected payload
	var payload messages.Payload
	decodedVal, err := base64.StdEncoding.DecodeString(msg.Protected)
	if err != nil {
		p.log.Error(err)
		return ``, err
	}

	err = json.Unmarshal(decodedVal, &payload)
	if err != nil {
		p.log.Error(err)
		return ``, err
	}

	decodedPubKey := base58.Decode(payload.Recipients[0].Header.Kid)

	return p.ks.Peer(decodedPubKey)
}
