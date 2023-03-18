package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	servicesPkg "github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/YasiruR/didcomm-prober/pubsub/stores"
	"github.com/YasiruR/didcomm-prober/pubsub/validator"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	zmqPkg "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"net/url"
	"strings"
	"time"
)

type state struct {
	myLabel     string
	pubEndpoint string
	invs        map[string]string // invitation per each topic
}

type services struct {
	km     servicesPkg.KeyManager
	probr  servicesPkg.Agent
	client servicesPkg.Client
	log    log.Logger
}

type internals struct {
	zmq      *zmq
	auth     *auth
	packr    *packer
	syncr    *syncer
	compactr *compactor
	gs       *stores.Group
	subs     *stores.Subscriber
}

type Agent struct {
	*state
	*internals
	*services
	proc     *processor
	outChan  chan string
	zmqBufms int
}

func NewAgent(zmqCtx *zmqPkg.Context, c *container.Container) (*Agent, error) {
	in, err := initInternals(zmqCtx, c)
	if err != nil {
		return nil, fmt.Errorf(`initializing internal services of group agent failed - %v`, err)
	}

	a := &Agent{
		state: &state{
			myLabel:     c.Cfg.Name,
			pubEndpoint: c.Cfg.PubEndpoint,
			invs:        make(map[string]string),
		},
		internals: in,
		services: &services{
			km:     c.KeyManager,
			probr:  c.Prober,
			client: c.Client,
			log:    c.Log,
		},
		proc:     newProcessor(c.Cfg.Name, c, in),
		outChan:  c.OutChan,
		zmqBufms: c.Cfg.ZmqBufMs,
	}

	a.start(c.Server)
	return a, nil
}

// initInternals initializes the internal components required by the group agent
func initInternals(zmqCtx *zmqPkg.Context, c *container.Container) (*internals, error) {
	gs := stores.NewGroupStore()
	compctr, err := newCompactor()
	if err != nil {
		return nil, fmt.Errorf(`initializing compressor failed - %v`, err)
	}

	transport, err := newZmqTransport(zmqCtx, gs, c)
	if err != nil {
		return nil, fmt.Errorf(`zmq transport init for group agent failed - %v`, err)
	}

	authn, err := authenticator(c.Cfg.Name, c.Cfg.Verbose)
	if err != nil {
		return nil, fmt.Errorf(`initializing zmq authenticator failed - %v`, err)
	}

	if err = authn.setPubAuthn(transport.pub); err != nil {
		return nil, fmt.Errorf(`setting authentication on pub socket failed - %v`, err)
	}

	if err = transport.start(c.Cfg.PubEndpoint); err != nil {
		return nil, fmt.Errorf(`starting zmq transport failed - %v`, err)
	}

	return &internals{
		subs:     stores.NewSubStore(),
		gs:       gs,
		auth:     authn,
		zmq:      transport,
		compactr: compctr,
		syncr:    newSyncer(gs),
		packr:    newPacker(c),
	}, nil
}

// start initializes handlers for subscribe and join requests (async),
// and listeners for all incoming messages eg: join requests,
// subscriptions, state changes (active/inactive) and data messages.
func (a *Agent) start(srvr servicesPkg.Server) {
	// add handler for subscribe messages
	subChan := make(chan models.Message)
	srvr.AddHandler(models.TypSubscribe, subChan, false)

	// sync handler for join-requests as requester expects
	// the group-info in return
	joinChan := make(chan models.Message)
	srvr.AddHandler(models.TypGroupJoin, joinChan, false)

	// initialize internal handlers for zmq requests on REQ sockets
	a.process(joinChan, a.proc.joinReqs)
	a.process(subChan, a.proc.subscriptions)

	// initialize listening on subscriptions on SUB sockets
	a.zmq.listen(typStateSkt, a.proc.state)
	a.zmq.listen(typMsgSkt, a.proc.data)
}

// Create constructs a group including creator's invitation
// for the group and its models.Member
func (a *Agent) Create(topic string, publisher bool, gp models.GroupParams) error {
	inv, err := a.probr.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	m := models.Member{
		Active:      true,
		Publisher:   publisher,
		Label:       a.myLabel,
		Inv:         inv,
		PubEndpoint: a.pubEndpoint,
	}

	a.invs[topic] = inv
	if err = a.gs.SetParams(topic, gp); err != nil {
		return fmt.Errorf(`updating group params failed - %v`, err)
	}

	if err = a.gs.AddMembrs(topic, m); err != nil {
		return fmt.Errorf(`adding group creator failed - %v`, err)
	}

	if gp.OrderEnabled {
		a.syncr.init(topic)
	}

	a.log.Info(fmt.Sprintf(`created '%s' group with the configured params`, topic), gp)
	return nil
}

func (a *Agent) Join(topic, acceptor string, publisher bool) error {
	// check if already Joined to the topic
	if a.gs.Joined(topic) {
		return fmt.Errorf(`already connected to group %s`, topic)
	}

	inv, err := a.probr.Invite()
	if err != nil {
		return fmt.Errorf(`generating invitation failed - %v`, err)
	}

	a.invs[topic] = inv
	group, err := a.reqState(topic, acceptor, inv)
	if err != nil {
		return fmt.Errorf(`requesting group state from %s failed - %v`, acceptor, err)
	}

	// adding this node as a member
	joiner := models.Member{
		Active:      true,
		Publisher:   publisher,
		Label:       a.myLabel,
		Inv:         inv,
		PubEndpoint: a.pubEndpoint,
	}

	if err = a.gs.SetParams(topic, group.Params); err != nil {
		return fmt.Errorf(`setting consistency failed - %v`, err)
	}

	if group.Params.OrderEnabled {
		a.syncr.init(topic)
	}

	hashes := make(map[string]string)
	for _, m := range group.Members {
		if !m.Active {
			continue
		}

		checksum, err := a.connectMember(topic, publisher, m)
		if err != nil {
			a.log.Error(fmt.Sprintf(`adding %s as a member failed - %v`, m.Label, err))
			continue
		}
		hashes[m.Label] = checksum
	}

	if len(group.Members) > 1 {
		if err = validator.ValidJoin(acceptor, group.Members, hashes); err != nil {
			if group.Params.JoinConsistent {
				return fmt.Errorf(`join failed due to inconsistent view of the group - %v`, err)
			}
			a.log.Warn(fmt.Sprintf(`group verification failed but proceeded with registration - %v`, err))
		} else {
			a.log.Info(`verified virtual synchrony of the group via consistent views`)
		}
	}

	if err = a.gs.AddMembrs(topic, append(group.Members, joiner)...); err != nil {
		return fmt.Errorf(`adding group members failed - %v`, err)
	}

	// buffer for zmq subscription latency
	time.Sleep(time.Duration(a.zmqBufms) * time.Millisecond)

	// wait till didcomm connections are established with all group members
	for _, m := range group.Members {
	checkPeer:
		_, err = a.probr.Peer(m.Label)
		if err != nil {
			goto checkPeer
			// can use sleep to reduce resource utilization
		}
	}

	// publish status
	if err = a.notifyAll(topic, true, publisher); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
	}

	return nil
}

func (a *Agent) connectMember(topic string, publisher bool, m models.Member) (checksum string, err error) {
	if err = a.connectDIDComm(m); err != nil {
		return ``, fmt.Errorf(`connecting to %s failed - %v`, m.Label, err)
	}

	checksum, err = a.subscribeData(topic, publisher, m)
	if err != nil {
		return ``, fmt.Errorf(`subscribing to topic %s with %s failed - %v`, topic, m.Label, err)
	}

	if err = a.zmq.connect(domain.RoleSubscriber, a.myLabel, topic, m); err != nil {
		return ``, fmt.Errorf(`transport connection failed - %v`, err)
	}

	if !publisher {
		return checksum, nil
	}

	s, err := a.probr.Service(domain.ServcGroupJoin, m.Label)
	if err != nil {
		return ``, fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
	}
	a.subs.Add(topic, m.Label, s.PubKey)

	return checksum, nil
}

// reqState checks if requester has already connected with acceptor
// via didcomm and if true, sends a didcomm group-join request using
// fetched peer's information. Returns the group-join response if both
// request is successful and requester is eligible.
func (a *Agent) reqState(topic, accptr, inv string) (*messages.ResGroupJoin, error) {
	s, err := a.probr.Service(domain.ServcGroupJoin, accptr)
	if err != nil {
		return nil, fmt.Errorf(`fetching service info failed for peer %s - %v`, accptr, err)
	}

	byts, err := json.Marshal(messages.ReqGroupJoin{
		Id:           uuid.New().String(),
		Type:         messages.JoinRequestV1,
		Label:        a.myLabel,
		Topic:        topic,
		RequesterInv: inv,
	})
	if err != nil {
		return nil, fmt.Errorf(`marshalling group-join request failed - %v`, err)
	}

	data, err := a.packr.pack(accptr, s.PubKey, byts)
	if err != nil {
		return nil, fmt.Errorf(`packing join-req for %s failed - %v`, accptr, err)
	}

	res, err := a.client.Send(models.TypGroupJoin, data, s.Endpoint)
	if err != nil {
		return nil, fmt.Errorf(`group-join request failed - %v`, err)
	}
	a.log.Debug(`group-join response received`, res)

	_, unpackedMsg, err := a.probr.ReadMessage(models.Message{Type: models.TypGroupJoin, Data: []byte(res), Reply: nil})
	if err != nil {
		return nil, fmt.Errorf(`unpacking group-join response failed - %v`, err)
	}

	var resGroup messages.ResGroupJoin
	if err = json.Unmarshal([]byte(unpackedMsg), &resGroup); err != nil {
		return nil, fmt.Errorf(`unmarshalling group-join response failed - %v`, err)
	}

	return &resGroup, nil
}

func (a *Agent) Send(topic, msg string) error {
	curntMembr := a.gs.Membr(topic, a.myLabel)
	if curntMembr == nil {
		return fmt.Errorf(`member information does not exist for the current member`)
	}

	if !curntMembr.Publisher {
		return fmt.Errorf(`current member is not registered as a publisher`)
	}

	subs, err := a.subs.QueryByTopic(topic)
	if err != nil {
		return fmt.Errorf(`fetching subscribers for topic %s failed - %v`, topic, err)
	}

	// including order-metadata only if it is a group message and syncing is enabled by params of topic
	syncdMsg, err := a.syncr.message(topic, []byte(msg))
	if err != nil {
		return fmt.Errorf(`constructing ordered group message failed - %v`, err)
	}

	var published bool
	for sub, key := range subs {
		data, err := a.packr.pack(sub, key, syncdMsg)
		if err != nil {
			return fmt.Errorf(`packing data message for %s failed - %v`, sub, err)
		}

		if err = a.zmq.publish(a.zmq.dataTopic(topic, a.myLabel, sub), data); err != nil {
			return fmt.Errorf(`zmq transport error - %v`, err)
		}

		published = true
		a.log.Trace(fmt.Sprintf(`published %s to %s of %s`, msg, topic, sub))
	}

	if published {
		a.outChan <- `Published '` + msg + `' to '` + topic + `'`
	}
	return nil
}

func (a *Agent) connectDIDComm(m models.Member) error {
	_, err := a.probr.Peer(m.Label)
	if err != nil {
		u, err := url.Parse(strings.TrimSpace(m.Inv))
		if err != nil {
			return fmt.Errorf(`parsing invitation url failed - %v`, err)
		}

		inv, ok := u.Query()[`oob`]
		if !ok {
			return fmt.Errorf(`invitation url does not contain oob query param`)
		}

		if err = a.probr.SyncAccept(inv[0]); err != nil {
			return fmt.Errorf(`accepting group-member invitation failed - %v`, err)
		}
	}

	return nil
}

// subscribeData sets subscriptions via zmq for status topic of the member.
// If the member is a publisher, it proceeds with sending a subscription
// didcomm message and subscribing to message topic via zmqPkg. A checksum
// of the group maintained by the added group member is returned.
func (a *Agent) subscribeData(topic string, publisher bool, m models.Member) (checksum string, err error) {
	// get my public key corresponding to this member
	subPublcKey, err := a.km.PublicKey(m.Label)
	if err != nil {
		return ``, fmt.Errorf(`fetching public key for the connection failed - %v`, err)
	}

	// B sends agent subscribe msg to member
	sm := messages.Subscribe{
		Id:        uuid.New().String(),
		Type:      messages.SubscribeV1,
		Subscribe: true,
		PubKey:    base58.Encode(subPublcKey),
		Topic:     topic,
		Member: models.Member{
			Active:      true,
			Publisher:   publisher,
			Label:       a.myLabel,
			Inv:         a.invs[topic],
			PubEndpoint: a.pubEndpoint,
		},
		Transport: messages.Transport{
			ServrPubKey:  a.auth.servr.pub,
			ClientPubKey: a.auth.client.pub,
		},
	}

	byts, err := json.Marshal(sm)
	if err != nil {
		return ``, fmt.Errorf(`marshalling subscribe message failed - %v`, err)
	}

	s, err := a.probr.Service(domain.ServcGroupJoin, m.Label)
	if err != nil {
		return ``, fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
	}

	data, err := a.packr.pack(m.Label, s.PubKey, byts)
	if err != nil {
		return ``, fmt.Errorf(`packing subscribe request for %s failed - %v`, m.Label, err)
	}

	res, err := a.client.Send(models.TypSubscribe, data, s.Endpoint)
	if err != nil {
		return ``, fmt.Errorf(`sending subscribe message failed - %v`, err)
	}

	_, unpackedMsg, err := a.probr.ReadMessage(models.Message{Type: models.TypSubscribe, Data: []byte(res), Reply: nil})
	if err != nil {
		return ``, fmt.Errorf(`reading subscribe didcomm response failed - %v`, err)
	}

	var resSm messages.ResSubscribe
	if err = json.Unmarshal([]byte(unpackedMsg), &resSm); err != nil {
		return ``, fmt.Errorf(`unmarshalling didcomm message into subscribe response struct failed - %v`, err)
	}

	var sktMsgs *zmqPkg.Socket = nil
	if resSm.Publisher {
		sktMsgs = a.zmq.msgs
	}

	if err = a.auth.setPeerAuthn(m.Label, resSm.Transport.ServrPubKey, resSm.Transport.ClientPubKey, a.zmq.state, sktMsgs); err != nil {
		return ``, fmt.Errorf(`setting zmq transport authentication failed - %v`, err)
	}

	return resSm.Checksum, nil
}

func (a *Agent) process(inChan chan models.Message, handlerFunc func(msg *models.Message) error) {
	go func() {
		for {
			msg := <-inChan
			if err := handlerFunc(&msg); err != nil {
				a.log.Error(fmt.Sprintf(`processing message by handler failed - %v`, err))
			}
		}
	}()
}

// notifyAll constructs a single status message with different didcomm
// messages packed per each member of the group and includes in a map with
// exchange ID as the key.
func (a *Agent) notifyAll(topic string, active, publisher bool) error {
	comprsd, err := a.compressStatus(topic, active, publisher)
	if err != nil {
		return fmt.Errorf(`compress status failed - %v`, err)
	}

	if err = a.zmq.publish(a.zmq.stateTopic(topic), comprsd); err != nil {
		return fmt.Errorf(`zmq transport error for status - %v`, err)
	}

	a.log.Debug(fmt.Sprintf(`published status (topic: %s, active: %t, publisher: %t)`, topic, active, publisher))
	return nil
}

func (a *Agent) compressStatus(topic string, active, publisher bool) ([]byte, error) {
	sm := messages.Status{Id: uuid.New().String(), Type: messages.MemberStatusV1, Topic: topic, AuthMsgs: map[string]string{}}
	byts, err := json.Marshal(models.Member{
		Label:       a.myLabel,
		Active:      active,
		Inv:         a.invs[topic],
		Publisher:   publisher,
		PubEndpoint: a.pubEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf(`marshalling member failed - %v`, err)
	}

	mems := a.gs.Membrs(topic)
	for _, m := range mems {
		if m.Label == a.myLabel {
			continue
		}

		pr, err := a.probr.Peer(m.Label)
		if err != nil {
			return nil, fmt.Errorf(`fetching peer failed - %v`, err)
		}

		s, err := a.probr.Service(domain.ServcGroupJoin, m.Label)
		if err != nil {
			return nil, fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
		}

		data, err := a.packr.pack(m.Label, s.PubKey, byts)
		if err != nil {
			return nil, fmt.Errorf(`packing message for %s failed - %v`, m.Label, err)
		}

		sm.AuthMsgs[pr.ExchangeThId] = string(data)
	}

	encodedStatus, err := json.Marshal(sm)
	if err != nil {
		return nil, fmt.Errorf(`marshalling status message failed - %v`, err)
	}

	cmprsd := a.compactr.zEncodr.EncodeAll(encodedStatus, make([]byte, 0, len(encodedStatus)))
	a.log.Trace(fmt.Sprintf(`compressed status message (from %d to %d #bytes)`, len(encodedStatus), len(cmprsd)))
	return cmprsd, nil
}

func (a *Agent) Leave(topic string) error {
	if err := a.zmq.unsubscribeAll(a.myLabel, topic); err != nil {
		return fmt.Errorf(`zmw unsubsription failed - %v`, err)
	}

	membrs := a.gs.Membrs(topic)
	if len(membrs) == 0 {
		return fmt.Errorf(`no members found`)
	}

	var publisher bool
	for _, m := range membrs {
		if m.Label == a.myLabel {
			publisher = m.Publisher
		}
	}

	if err := a.notifyAll(topic, false, publisher); err != nil {
		return fmt.Errorf(`publishing inactive status failed - %v`, err)
	}

	a.subs.DeleteTopic(topic)
	a.gs.DeleteTopic(topic)
	a.outChan <- `Left group ` + topic
	return nil
}

func (a *Agent) Info(topic string) (gp models.GroupParams, mems []models.Member) {
	// removing invitation for more clarity
	for _, m := range a.gs.Membrs(topic) {
		m.Inv = ``
		mems = append(mems, m)
	}

	params := a.gs.Params(topic)
	if params == nil {
		return models.GroupParams{}, mems
	}

	return *params, mems
}

func (a *Agent) Close() error {
	if err := a.auth.close(); err != nil {
		return fmt.Errorf(`closing authenticator failed - %v`, err)
	}

	return a.zmq.close()
}
