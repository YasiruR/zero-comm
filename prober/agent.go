package prober

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/tryfix/log"
)

type streams struct {
	connReq, connRes, data chan models.Message
}

type Prober struct {
	label           string
	invEndpoint     string
	exchEndpoint    string
	grpJoinEndpoint string
	ks              services.KeyManager
	packer          services.Packer
	did             services.DIDAgent
	conn            services.Connector
	oob             services.OutOfBand
	peers           map[string]models.Peer
	didDocs         map[string]messages.DIDDocument
	dids            map[string]string
	outChan         chan string
	connDone        chan models.Connection
	log             log.Logger
	client          services.Client
}

func NewProber(c *domain.Container) (p *Prober, err error) {
	p = &Prober{
		invEndpoint:     c.Cfg.InvEndpoint,
		exchEndpoint:    c.Cfg.InvEndpoint,
		grpJoinEndpoint: c.Cfg.InvEndpoint,
		ks:              c.KeyManager,
		packer:          c.Packer,
		log:             c.Log,
		did:             c.DidAgent,
		conn:            c.Connector,
		oob:             c.OOB,
		outChan:         c.OutChan,
		label:           c.Cfg.Args.Name,
		peers:           map[string]models.Peer{}, // name as the key may not be ideal
		didDocs:         map[string]messages.DIDDocument{},
		dids:            map[string]string{},
		connDone:        c.ConnDoneChan,
		client:          c.Client,
	}

	p.initHandlers(c.Server)
	return p, nil
}

func (p *Prober) initHandlers(serv services.Server) {
	// initializing message incoming streams for prober
	s := &streams{
		connReq: make(chan models.Message),
		connRes: make(chan models.Message),
		data:    make(chan models.Message),
	}

	serv.AddHandler(domain.MsgTypConnReq, s.connReq, true)
	serv.AddHandler(domain.MsgTypConnRes, s.connRes, true)
	serv.AddHandler(domain.MsgTypData, s.data, true)
	go p.listen(s)
}

func (p *Prober) listen(s *streams) {
	for {
		select {
		case m := <-s.connReq:
			if err := p.processConnReq(m); err != nil {
				p.log.Error(err)
			}
		case m := <-s.connRes:
			if err := p.processConnRes(m); err != nil {
				p.log.Error(err)
			}
		case m := <-s.data:
			if _, err := p.ReadMessage(m); err != nil {
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
	invDidDoc := p.did.CreateDIDDoc([]models.Service{
		{Id: uuid.New().String(), Type: domain.ServcDIDExchange, Endpoint: p.invEndpoint, PubKey: p.ks.InvPublicKey()},
	})

	// but uses did created from default did doc as it serves as the identifier in invitation
	url, err = p.oob.CreateInv(p.label, ``, invDidDoc) // todo null did
	if err != nil {
		return ``, fmt.Errorf(`creating invitation failed - %v`, err)
	}

	return url, nil
}

// Accept creates a connection request and sends it to the invitation endpoint
func (p *Prober) Accept(encodedInv string) (sender string, err error) {
	inv, invEndpoint, peerInvPubKey, err := p.oob.ParseInv(encodedInv)
	if err != nil {
		return ``, fmt.Errorf(`parsing invitation failed - %v`, err)
	}

	// set up prerequisites for a connection (diddoc, did, keys)
	pubKey, prvKey, err := p.setConnPrereqs(inv.Label)
	if err != nil {
		return ``, fmt.Errorf(`setting up prerequisites for connection with %s failed - %v`, inv.Label, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(p.didDocs[inv.Label])
	if err != nil {
		return ``, fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDoc, err := p.packer.Pack(docBytes, peerInvPubKey, pubKey, prvKey)
	if err != nil {
		return ``, fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	// todo check how concurrent conn requests go along (since same invitation and hence pthid)
	// creates connection request
	connReq, err := p.conn.CreateConnReq(p.label, inv.Id, p.dids[inv.Label], encDoc)
	if err != nil {
		return ``, fmt.Errorf(`creating connection request failed - %v`, err)
	}

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return ``, fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if _, err = p.client.Send(domain.MsgTypConnReq, connReqBytes, invEndpoint); err != nil {
		return ``, fmt.Errorf(`sending connection request failed - %v`, err)
	}

	p.peers[inv.Label] = models.Peer{DID: inv.From, ExchangeThId: inv.Id}
	return inv.Label, nil
}

// processConnReq parses the connection request, creates a connection response and sends it to did endpoint
func (p *Prober) processConnReq(msg models.Message) error {
	peerLabel, pthId, peerDid, peerEncDocBytes, err := p.conn.ParseConnReq(msg.Data)
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

	connRes, err := p.conn.CreateConnRes(pthId, p.dids[peerLabel], encDidDoc)
	if err != nil {
		return fmt.Errorf(`creating connection response failed - %v`, err)
	}

	connResBytes, err := json.Marshal(connRes)
	if err != nil {
		return fmt.Errorf(`marshalling connection response failed - %v`, err)
	}

	if _, err = p.client.Send(domain.MsgTypConnRes, connResBytes, peerEndpoint); err != nil {
		return fmt.Errorf(`sending connection response failed - %v`, err)
	}

	p.peers[peerLabel] = models.Peer{DID: peerDid, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}
	p.outChan <- `Connection established with ` + peerLabel

	return nil
}

func (p *Prober) processConnRes(msg models.Message) error {
	pthId, peerEncDocBytes, err := p.conn.ParseConnRes(msg.Data)
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

			p.peers[name] = models.Peer{DID: peer.DID, Endpoint: peerEndpoint, PubKey: peerPubKey, ExchangeThId: pthId}

			// should not be sent to non-pubsub relationships but the validation is done in pubsub module
			if p.connDone != nil {
				p.connDone <- models.Connection{Peer: name, PubKey: peerPubKey}
			}

			p.outChan <- `Connection established with ` + name
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
		return fmt.Errorf(`packing message failed - %v`, err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf(`marshalling didcomm message failed - %v`, err)
	}

	if _, err = p.client.Send(typ, data, peer.Endpoint); err != nil {
		return fmt.Errorf(`sending siscomm message failed - %v`, err)
	}

	if typ == domain.MsgTypData {
		p.outChan <- `Message sent`
	} else {
		p.log.Trace(fmt.Sprintf(`'%s' message sent`, typ))
	}

	return nil
}

func (p *Prober) ReadMessage(msg models.Message) (text string, err error) {
	peerName, err := p.peerByMsg(msg.Data)
	if err != nil {
		//p.log.Debug(fmt.Sprintf(`getting peer info failed - %v`, err))
		return ``, fmt.Errorf(`getting peer info failed - %v`, err)
	}

	ownPubKey, err := p.ks.PublicKey(peerName)
	if err != nil {
		return ``, fmt.Errorf(`getting public key for connection with %s failed - %v`, peerName, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(peerName)
	if err != nil {
		return ``, fmt.Errorf(`getting private key for connection with %s failed - %v`, peerName, err)
	}

	textBytes, err := p.packer.Unpack(msg.Data, ownPubKey, ownPrvKey)
	if err != nil {
		return ``, fmt.Errorf(`unpacking message failed - %v`, err)
	}

	if msg.Type == domain.MsgTypData {
		p.outChan <- `Message received: '` + string(textBytes) + `'`
	} else {
		p.log.Trace(fmt.Sprintf(`message received for type '%s' - %s`, msg.Type, string(textBytes)))
	}

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
	didDoc := p.did.CreateDIDDoc([]models.Service{
		{Id: uuid.New().String(), Type: domain.ServcMessage, Endpoint: p.exchEndpoint, PubKey: pubKey},
		{Id: uuid.New().String(), Type: domain.ServcGroupJoin, Endpoint: p.grpJoinEndpoint, PubKey: pubKey},
	})
	did, err := p.did.CreatePeerDID(didDoc)
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
		return ``, fmt.Errorf(`unmarshalling authcrypt message failed - %v`, err)
	}

	// decode protected payload
	var payload messages.Payload
	decodedVal, err := base64.StdEncoding.DecodeString(msg.Protected)
	if err != nil {
		return ``, fmt.Errorf(`decoding protected value with base64 failed - %v`, err)
	}

	err = json.Unmarshal(decodedVal, &payload)
	if err != nil {
		return ``, fmt.Errorf(`unmarshalling protected payload failed - %v`, err)
	}

	decodedPubKey := base58.Decode(payload.Recipients[0].Header.Kid)
	return p.ks.Peer(decodedPubKey)
}
