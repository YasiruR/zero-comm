package prober

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/tryfix/log"
	"sync"
	"time"
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
	did             services.DIDUtils
	conn            services.Connector
	oob             services.OutOfBand
	peers           *peers
	didStore        *didStore
	outChan         chan string
	log             log.Logger
	client          services.Client
	syncCons        *sync.Map
}

func NewProber(c *container.Container) (p *Prober, err error) {
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
		peers:           initPeerStore(c.Log),
		didStore:        initDIDStore(),
		client:          c.Client,
		syncCons:        &sync.Map{},
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

	serv.AddHandler(models.TypConnReq, s.connReq, true)
	serv.AddHandler(models.TypConnRes, s.connRes, true)
	serv.AddHandler(models.TypData, s.data, true)
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
			if _, _, err := p.ReadMessage(m); err != nil {
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

func (p *Prober) SyncAccept(encodedInv string) error {
	inviter, err := p.Accept(encodedInv)
	if err != nil {
		return fmt.Errorf(`accepting invitation failed - %v`, err)
	}

	// todo set a timeout for waiting
	syncChan := make(chan bool)
	p.syncCons.Store(inviter, syncChan)
	<-syncChan

	return nil
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

	did, doc, err := p.didStore.get(inv.Label)
	if err != nil {
		return ``, fmt.Errorf(`fetching dids failed - %v`, err)
	}

	// marshals did doc to proceed with packing process
	docBytes, err := json.Marshal(doc)
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
	connReq, err := p.conn.CreateConnReq(p.label, inv.Id, did, encDoc)
	if err != nil {
		return ``, fmt.Errorf(`creating connection request failed - %v`, err)
	}

	// marshals connection request
	connReqBytes, err := json.Marshal(connReq)
	if err != nil {
		return ``, fmt.Errorf(`marshalling connection request failed - %v`, err)
	}

	if _, err = p.client.Send(models.TypConnReq, connReqBytes, invEndpoint); err != nil {
		return ``, fmt.Errorf(`sending connection request failed - %v`, err)
	}

	p.peers.add(inv.Label, models.Peer{DID: inv.From, ExchangeThId: connReq.Thread.ThId})
	return inv.Label, nil
}

// processConnReq parses the connection request, creates a connection response and sends it to did endpoint
func (p *Prober) processConnReq(msg models.Message) error {
	peerLabel, exchId, peerDid, peerEncDocBytes, err := p.conn.ParseConnReq(msg.Data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// decrypts peer did doc which is encrypted with invitation keys
	svcs, err := p.getPeerInfo(peerEncDocBytes, p.ks.InvPublicKey(), p.ks.InvPrivateKey())
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	prMsgEndpnt, prMsgPubKy, err := p.infoByServc(domain.ServcMessage, svcs)
	if err != nil {
		return fmt.Errorf(`getting message endpoint failed - %v`, err)
	}

	// set up prerequisites for a connection (diddoc, did, keys)
	pubKey, prvKey, err := p.setConnPrereqs(peerLabel)
	if err != nil {
		return fmt.Errorf(`setting up prerequisites for connection with %s failed - %v`, peerLabel, err)
	}

	did, doc, err := p.didStore.get(peerLabel)
	if err != nil {
		return fmt.Errorf(`fetching did-doc failed - %v`, err)
	}

	// marshals own did doc to proceed with packing process
	docBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// encrypts did doc with peer invitation public key and default own key pair
	encDidDoc, err := p.packer.Pack(docBytes, prMsgPubKy, pubKey, prvKey)
	if err != nil {
		return fmt.Errorf(`encrypting did doc failed - %v`, err)
	}

	connRes, err := p.conn.CreateConnRes(exchId, did, encDidDoc)
	if err != nil {
		return fmt.Errorf(`creating connection response failed - %v`, err)
	}

	connResBytes, err := json.Marshal(connRes)
	if err != nil {
		return fmt.Errorf(`marshalling connection response failed - %v`, err)
	}

	if _, err = p.client.Send(models.TypConnRes, connResBytes, prMsgEndpnt); err != nil {
		return fmt.Errorf(`sending connection response failed - %v`, err)
	}

	p.peers.add(peerLabel, models.Peer{DID: peerDid, Services: svcs, ExchangeThId: exchId})
	p.outChan <- `Connection established with ` + peerLabel

	return nil
}

func (p *Prober) processConnRes(msg models.Message) error {
	pthId, peerEncDocBytes, err := p.conn.ParseConnRes(msg.Data)
	if err != nil {
		return fmt.Errorf(`parsing connection request failed - %v`, err)
	}

	// todo send complete message

	var retryCount int
retry:
	name, pr, ok := p.peers.peerByExchId(pthId)
	if !ok {
		if retryCount != domain.RetryCount {
			retryCount++
			time.Sleep(domain.RetryIntervalMs * time.Millisecond)
			goto retry
		}
		return fmt.Errorf(`peer does not exist for exchange id %s`, pthId)
	}

	ownPubKey, err := p.ks.PublicKey(name)
	if err != nil {
		return fmt.Errorf(`getting public key for connection with %s failed - %v`, name, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(name)
	if err != nil {
		return fmt.Errorf(`getting private key for connection with %s failed - %v`, name, err)
	}

	// decrypts peer did doc which is encrypted with default keys
	svcs, err := p.getPeerInfo(peerEncDocBytes, ownPubKey, ownPrvKey)
	if err != nil {
		return fmt.Errorf(`getting peer data failed - %v`, err)
	}

	p.peers.add(name, models.Peer{DID: pr.DID, Services: svcs, ExchangeThId: pthId})
	val, ok := p.syncCons.Load(name)
	if ok {
		syncChan, ok := val.(chan bool)
		if !ok {
			return fmt.Errorf(`incompatible type for sync channel (%v)`, val)
		}
		syncChan <- true
	}

	p.outChan <- `Connection established with ` + name
	return nil
}

func (p *Prober) getPeerInfo(encDocBytes, recPubKey, recPrvKey []byte) (svcs []models.Service, err error) {
	peerDocBytes, err := p.packer.Unpack(encDocBytes, recPubKey, recPrvKey)
	if err != nil {
		return nil, fmt.Errorf(`decrypting did doc failed - %v`, err)
	}

	// unmarshalls decrypted did doc
	var peerDidDoc messages.DIDDocument
	if err = json.Unmarshal(peerDocBytes, &peerDidDoc); err != nil {
		return nil, fmt.Errorf(`unmarshalling decrypted did doc failed - %v`, err)
	}

	if len(peerDidDoc.Service) == 0 {
		return nil, fmt.Errorf(`did doc does not contain a service`)
	}

	for _, s := range peerDidDoc.Service {
		if len(s.RecipientKeys) == 0 {
			p.log.Error(fmt.Sprintf(`did doc does not contain recipient keys for the service (%s)`, s.Type))
			continue
		}

		for _, rk := range s.RecipientKeys {
			peerPubKey, err := base64.StdEncoding.DecodeString(rk) // assumes the first eligible key-pair works fine for POC
			if err != nil {
				p.log.Error(fmt.Sprintf(`decoding recipient key failed for service (%s) - %v`, s.Type, err))
				continue
			}
			svcs = append(svcs, models.Service{Id: s.Id, Type: s.Type, Endpoint: s.ServiceEndpoint, PubKey: peerPubKey})
			break
		}
	}

	return svcs, nil
}

func (p *Prober) SendMessage(mt models.MsgType, to, text string) error {
	peer, err := p.peers.peerByLabel(to)
	if err != nil {
		return fmt.Errorf(`no didcomm connection found for the recipient %s - %v`, to, err)
	}

	ownPubKey, err := p.ks.PublicKey(to)
	if err != nil {
		return fmt.Errorf(`getting public key for connection with %s failed - %v`, to, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(to)
	if err != nil {
		return fmt.Errorf(`getting private key for connection with %s failed - %v`, to, err)
	}

	prMsgEndpnt, prMsgPubKy, err := p.infoByServc(domain.ServcMessage, peer.Services)
	if err != nil {
		return fmt.Errorf(`getting message endpoint failed - %v`, err)
	}

	msg, err := p.packer.Pack([]byte(text), prMsgPubKy, ownPubKey, ownPrvKey)
	if err != nil {
		return fmt.Errorf(`packing message failed - %v`, err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf(`marshalling didcomm message failed - %v`, err)
	}

	if _, err = p.client.Send(mt, data, prMsgEndpnt); err != nil {
		return fmt.Errorf(`sending didcomm message failed - %v`, err)
	}

	if mt == models.TypData {
		p.outChan <- `Message sent`
	} else {
		p.log.Trace(fmt.Sprintf(`'%s' message sent to %s`, mt.String(), to))
	}

	return nil
}

func (p *Prober) ReadMessage(msg models.Message) (sender, text string, err error) {
	peerName, err := p.peerByMsg(msg.Data)
	if err != nil {
		//p.log.Debug(fmt.Sprintf(`getting peer info failed - %v`, err))
		return ``, ``, fmt.Errorf(`getting peer info failed - %v`, err)
	}

	ownPubKey, err := p.ks.PublicKey(peerName)
	if err != nil {
		return ``, ``, fmt.Errorf(`getting public key for connection with %s failed - %v`, peerName, err)
	}

	ownPrvKey, err := p.ks.PrivateKey(peerName)
	if err != nil {
		return ``, ``, fmt.Errorf(`getting private key for connection with %s failed - %v`, peerName, err)
	}

	textBytes, err := p.packer.Unpack(msg.Data, ownPubKey, ownPrvKey)
	if err != nil {
		return ``, ``, fmt.Errorf(`unpacking message failed - %v`, err)
	}

	if msg.Type == models.TypData {
		p.outChan <- `Message received: '` + string(textBytes) + `'`
	} else {
		p.log.Trace(fmt.Sprintf(`message received for type '%s' by %s - %s`, msg.Type, peerName, string(textBytes)))
	}

	return peerName, string(textBytes), nil
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

	p.didStore.add(peer, did, didDoc)
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

func (p *Prober) infoByServc(filter string, svcs []models.Service) (endpoint string, pubKey []byte, err error) {
	for _, s := range svcs {
		if s.Type == filter {
			return s.Endpoint, s.PubKey, nil
		}
	}

	return ``, nil, fmt.Errorf(`services does not contain %s`, filter)
}

// Peer returns the connected models.Peer queried by label
func (p *Prober) Peer(label string) (models.Peer, error) {
	pr, err := p.peers.peerByLabel(label)
	if err != nil {
		return models.Peer{}, fmt.Errorf(`no peer found for the label (%s) - %v`, label, err)
	}
	return pr, nil
}

func (p *Prober) Service(name, peer string) (*models.Service, error) {
	pr, err := p.Peer(peer)
	if err != nil {
		return nil, fmt.Errorf(`no peer found - %v`, err)
	}

	var srvc *models.Service
	for _, s := range pr.Services {
		if s.Type == name {
			srvc = &s
			break
		}
	}

	if srvc.Type == `` {
		return nil, fmt.Errorf(`requested service (%s) is not found for peer (%s)`, name, peer)
	}

	return srvc, nil
}

func (p *Prober) SyncService(name, peer string, timeoutMs int64) (*models.Service, error) {
	c := make(chan *models.Service)
	tickr := time.NewTicker(time.Duration(timeoutMs) * time.Millisecond)

	go func(c chan *models.Service) {
		for {
			tmpSvc, err := p.Service(name, peer)
			if err == nil {
				c <- tmpSvc
				break
			}
		}
	}(c)

	for {
		select {
		case svc := <-c:
			return svc, nil
		case <-tickr.C:
			return nil, fmt.Errorf(`timedout waiting for the service info (service=%s, peer=%s)`, name, peer)
		}
	}
}

func (p *Prober) ValidConn(exchId string) (pr models.Peer, ok bool) {
	_, pr, ok = p.peers.peerByExchId(exchId)
	return pr, ok
}
