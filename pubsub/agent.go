package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"net/url"
	"strings"
	"time"
)

const (
	zmqLatencyBufSec = 1
)

// todo remove skt from internal fields
type sockets struct {
	sktPub   *zmq.Socket
	sktState *zmq.Socket
	sktMsgs  *zmq.Socket
}

type compactor struct {
	zEncodr *zstd.Encoder
	zDecodr *zstd.Decoder
}

type Agent struct {
	myLabel     string
	pubEndpoint string
	invs        map[string]string // invitation per each topic
	probr       services.Agent
	client      services.Client
	km          services.KeyManager
	packer      services.Packer
	subs        *subStore
	gs          *groupStore
	log         log.Logger
	outChan     chan string
	*compactor
	*sockets
}

func NewAgent(zmqCtx *zmq.Context, c *domain.Container) (*Agent, error) {
	//authntcatr, err := initAuthenticator(zmqCtx, c.Cfg.Verbose)
	//if err != nil {
	//	return nil, fmt.Errorf(`initializing zmq authenticator failed - %v`, err)
	//}
	//
	//sktPub, err := authntcatr.pubSkt()
	//if err != nil {
	//	return nil, fmt.Errorf(`initializing authenticated zmq pub socket failed - %v`, err)
	//}

	// create PUB and SUB sockets for msgs and statuses
	sktPub, err := zmqCtx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	if err = sktPub.Bind(c.Cfg.PubEndpoint); err != nil {
		return nil, fmt.Errorf(`binding zmq pub socket to %s failed - %v`, c.Cfg.PubEndpoint, err)
	}

	sktStates, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for status topic failed - %v`, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	zstdEncoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf(`creating zstd encoder failed - %v`, err)
	}

	zstdDecoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf(`creating zstd decoder failed - %v`, err)
	}

	a := &Agent{
		myLabel:     c.Cfg.Args.Name,
		pubEndpoint: c.Cfg.PubEndpoint,
		invs:        make(map[string]string),
		probr:       c.Prober,
		client:      c.Client,
		km:          c.KeyManager,
		packer:      c.Packer,
		log:         c.Log,
		outChan:     c.OutChan,
		subs:        newSubStore(),
		gs:          newGroupStore(),
		compactor: &compactor{
			zEncodr: zstdEncoder,
			zDecodr: zstdDecoder,
		},
		sockets: &sockets{
			sktPub:   sktPub,
			sktState: sktStates,
			sktMsgs:  sktMsgs,
		},
	}

	a.init(c.Server)
	return a, nil
}

// init initializes handlers for subscribe and join requests (async),
// and listeners for all incoming messages eg: join requests,
// subscriptions, state changes (active/inactive) and data messages.
func (a *Agent) init(s services.Server) {
	// add handler for subscribe messages
	subChan := make(chan models.Message)
	s.AddHandler(domain.MsgTypSubscribe, subChan, true)

	// sync handler for join-requests as requester expects
	// the group-info in return
	joinChan := make(chan models.Message)
	s.AddHandler(domain.MsgTypGroupJoin, joinChan, false)

	go a.joinReqListnr(joinChan)
	go a.subscrptionLisntr(subChan)
	go a.statusListnr()
	go a.msgListnr()
}

// Create constructs a group including creator's invitation
// for the group and its models.Member
func (a *Agent) Create(topic string, publisher bool) error {
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
	a.gs.addMembr(topic, m)

	return nil
}

func (a *Agent) Join(topic, acceptor string, publisher bool) error {
	// check if already joined to the topic
	if a.gs.joined(topic) {
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

	// todo check if should be used after connect
	if err = a.subscribeStatus(topic); err != nil {
		return fmt.Errorf(`subscribing to status topic of %s failed - %v`, topic, err)
	}

	// adding this node as a member
	a.gs.addMembr(topic, models.Member{
		Active:      true,
		Publisher:   publisher,
		Label:       a.myLabel,
		Inv:         inv,
		PubEndpoint: a.pubEndpoint,
	})

	for _, m := range group.Members {
		// todo can do in parallel?
		if !m.Active {
			continue
		}

		if err = a.connectDIDComm(m); err != nil {
			return fmt.Errorf(`connecting to %s failed - %v`, m.Label, err)
		}

		if err = a.subscribeData(topic, publisher, m); err != nil {
			return fmt.Errorf(`subscribing to topic %s with %s failed - %v`, topic, m.Label, err)
		}

		if err = a.connectZmq(topic, m); err != nil {
			return fmt.Errorf(`settting up zmq socket connections failed - %v`, err)
		}

		// add as a member to be shared with another in future
		a.gs.addMembr(topic, m)

		if !publisher {
			continue
		}

		// todo improve (or can remove sending by sub msg to other peer)
		//pr, err := a.probr.Peer(m.Label)
		//if err != nil {
		//	return fmt.Errorf(`getting peer info for %s failed - %v`, m.Label, err)
		//}
		//
		//var sk []byte
		//for _, s := range pr.Services {
		//	if s.Type == domain.ServcGroupJoin {
		//		sk = s.PubKey
		//	}
		//}

		s, _, err := a.serviceInfo(m.Label)
		if err != nil {
			return fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
		}
		a.subs.add(topic, m.Label, s.PubKey)
	}

	// publish status
	time.Sleep(zmqLatencyBufSec * time.Second) // buffer for zmq subscription latency
	if err = a.notifyAll(topic, true, publisher); err != nil {
		return fmt.Errorf(`publishing status active failed - %v`, err)
	}

	return nil
}

// reqState checks if requester has already connected with acceptor
// via didcomm and if true, sends a didcomm group-join request using
// fetched peer's information. Returns the group-join response if both
// request is successful and requester is eligible.
func (a *Agent) reqState(topic, accptr, inv string) (*messages.ResGroupJoin, error) {
	//p, err := a.probr.Peer(accptr)
	//if err != nil {
	//	return nil, fmt.Errorf(`fetching acceptor failed - %v`, err)
	//}
	//
	//// fetching group-join endpoint and public key of acceptor
	//var srvcJoin models.Service
	//var accptrKey []byte
	//for _, s := range p.Services {
	//	if s.Type == domain.ServcGroupJoin {
	//		srvcJoin = s
	//		accptrKey = s.PubKey
	//		break
	//	}
	//}

	srvcJoin, _, err := a.serviceInfo(accptr)
	if err != nil {
		return nil, fmt.Errorf(`fetching service info failed for peer %s - %v`, accptr, err)
	}

	if srvcJoin.Type == `` {
		return nil, fmt.Errorf(`acceptor does not provide group-join service`)
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

	data, err := a.pack(accptr, srvcJoin.PubKey, byts)
	if err != nil {
		return nil, fmt.Errorf(`packing join-req for %s failed - %v`, accptr, err)
	}

	res, err := a.client.Send(domain.MsgTypGroupJoin, data, srvcJoin.Endpoint)
	if err != nil {
		return nil, fmt.Errorf(`group-join request failed - %v`, err)
	}
	a.log.Debug(`group-join response received`, res)

	var resGroup messages.ResGroupJoin
	if err = json.Unmarshal([]byte(res), &resGroup); err != nil {
		return nil, fmt.Errorf(`unmarshalling group-join response failed - %v`, err)
	}

	return &resGroup, nil
}

func (a *Agent) Send(topic, msg string) error {
	subs, err := a.subs.queryByTopic(topic)
	if err != nil {
		return fmt.Errorf(`fetching subscribers for topic %s failed - %v`, topic, err)
	}

	var published bool
	for sub, key := range subs {
		data, err := a.pack(sub, key, []byte(msg))
		if err != nil {
			return fmt.Errorf(`packing data message for %s failed - %v`, sub, err)
		}

		it := a.internalTopic(topic, a.myLabel, sub)
		if _, err = a.sktPub.SendMessage(fmt.Sprintf(`%s %s`, it, string(data))); err != nil {
			return fmt.Errorf(`publishing message (%s) failed for %s - %v`, msg, sub, err)
		}

		published = true
		a.log.Trace(fmt.Sprintf(`published %s to %s`, msg, it))
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

func (a *Agent) connectZmq(topic string, m models.Member) error {
	// B connects to member via SUB for statuses and msgs
	if err := a.sktState.Connect(m.PubEndpoint); err != nil {
		return fmt.Errorf(`connecting to publisher (%s) state socket failed - %v`, m.PubEndpoint, err)
	}

	if m.Publisher {
		if err := a.sktMsgs.Connect(m.PubEndpoint); err != nil {
			return fmt.Errorf(`connecting to publisher message socket failed - %v`, err)
		}

		// B subscribes via zmq
		it := a.internalTopic(topic, m.Label, a.myLabel)
		if err := a.sktMsgs.SetSubscribe(it); err != nil {
			return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, it, err)
		}
	}

	return nil
}

func (a *Agent) subscribeStatus(topic string) error {
	if err := a.sktState.SetSubscribe(a.stateTopic(topic)); err != nil {
		return fmt.Errorf(`setting zmq subscription failed for topic %s - %v`, a.stateTopic(topic), err)
	}

	return nil
}

// subscribe sets subscriptions via zmq for status topic of the member.
// If the member is a publisher, it proceeds with sending a subscription
// didcomm message and subscribing to message topic via zmq.
func (a *Agent) subscribeData(topic string, publisher bool, m models.Member) error {
	// get my public key corresponding to this member
	subPublcKey, err := a.km.PublicKey(m.Label)
	if err != nil {
		return fmt.Errorf(`fetching public key for the connection failed - %v`, err)
	}

	// B sends agent subscribe msg to pub
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
	}

	byts, err := json.Marshal(sm)
	if err != nil {
		return fmt.Errorf(`marshalling subscribe message failed - %v`, err)
	}

	//if err = a.probr.SendMessage(domain.MsgTypSubscribe, m.Label, string(byts)); err != nil {
	//	return fmt.Errorf(`sending subscribe message failed for topic %s - %v`, topic, err)
	//}

	s, _, err := a.serviceInfo(m.Label)
	if err != nil {
		return fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
	}

	data, err := a.pack(m.Label, s.PubKey, byts)
	if err != nil {
		return fmt.Errorf(`packing subscribe request for %s failed - %v`, m.Label, err)
	}

	res, err := a.client.Send(domain.MsgTypSubscribe, data, s.Endpoint)
	if err != nil {
		return fmt.Errorf(`sending subscribe message failed - %v`, err)
	}

	fmt.Println()
	fmt.Println("SUB MSG SENT: ", string(byts))
	fmt.Println()

	//if !m.Publisher {
	//	return nil
	//}

	return nil
}

func (a *Agent) joinReqListnr(joinChan chan models.Message) {
	for {
		msg := <-joinChan
		body, err := a.probr.ReadMessage(msg)
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading group-join authcrypt request failed - %v`, err))
			continue
		}

		var req messages.ReqGroupJoin
		if err = json.Unmarshal([]byte(body), &req); err != nil {
			a.log.Error(fmt.Sprintf(`unmarshalling group-join request failed - %v`, err))
			continue
		}

		if !a.validJoiner(req.Label) {
			a.log.Error(fmt.Sprintf(`group-join request denied to member (%s)`, req.Label))
			continue
		}

		byts, err := json.Marshal(messages.ResGroupJoin{
			Id:      uuid.New().String(),
			Type:    messages.JoinResponseV1,
			Members: a.gs.membrs(req.Topic),
		})
		if err != nil {
			a.log.Error(fmt.Sprintf(`marshalling group-join response failed - %v`, err))
		}
		a.log.Debug(fmt.Sprintf(`shared group state upon join request by %s`, req.Label), string(byts))

		// a null response is sent if an error occurred
		msg.Reply <- byts
	}
}

// todo check if subscription can be done with active status
func (a *Agent) subscrptionLisntr(subChan chan models.Message) {
	for {
		msg := <-subChan
		unpackedMsg, err := a.probr.ReadMessage(msg)
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading subscribe message failed - %v`, err))
			continue
		}

		var sm messages.Subscribe
		if err = json.Unmarshal([]byte(unpackedMsg), &sm); err != nil {
			a.log.Error(fmt.Sprintf(`unmarshalling subscribe message failed - %v`, err))
			continue
		}

		fmt.Println("SUB MSG RECVD: ", sm)

		// todo send response with pub key for transport
		// todo may want to make sub handler sync

		if !sm.Subscribe {
			a.subs.delete(sm.Topic, sm.Member.Label)
			continue
		}

		if !a.validJoiner(sm.Member.Label) {
			a.log.Error(fmt.Sprintf(`requester (%s) is not eligible`, sm.Member.Label))
			continue
		}

		var newTopic bool
		if _, err = a.subs.queryByTopic(sm.Topic); err != nil {
			newTopic = true
		}

		sk := base58.Decode(sm.PubKey)
		a.subs.add(sm.Topic, sm.Member.Label, sk)

		if err = a.sktState.Connect(sm.Member.PubEndpoint); err != nil {
			a.log.Error(fmt.Sprintf(`connecting to publisher state socket failed - %v`, err))
			continue
		}

		if newTopic {
			if err = a.subscribeStatus(sm.Topic); err != nil {
				a.log.Error(fmt.Sprintf(`subscribing to status topic of %s failed - %v`, sm.Topic, err))
				continue
			}
		}

		if sm.Member.Publisher {
			if err = a.sktMsgs.Connect(sm.Member.PubEndpoint); err != nil {
				a.log.Error(fmt.Sprintf(`connecting to publisher message socket failed - %v`, err))
				continue
			}

			it := a.internalTopic(sm.Topic, sm.Member.Label, a.myLabel)
			if err = a.sktMsgs.SetSubscribe(it); err != nil {
				a.log.Error(fmt.Sprintf(`setting zmq subscription failed for topic %s - %v`, it, err))
			}
		}
		a.log.Debug(`processed subscription request`, sm)
	}
}

func (a *Agent) statusListnr() {
	for {
		msg, err := a.sktState.Recv(0)
		if err != nil {
			a.log.Error(fmt.Sprintf(`receiving zmq message for member status failed - %v`, err))
			continue
		}

		frames := strings.SplitN(msg, ` `, 2)
		if len(frames) != 2 {
			a.log.Error(fmt.Sprintf(`received an invalid status message (length=%v) - %v`, len(frames), err))
			continue
		}

		sm, err := a.extractStatus(frames[1])
		if err != nil {
			a.log.Error(fmt.Sprintf(`extracting status message failed - %v`, err))
			continue
		}

		var validMsg string
		for exchId, encMsg := range sm.AuthMsgs {
			ok, _ := a.probr.ValidConn(exchId)
			if ok {
				validMsg = encMsg
				break
			}
		}

		if validMsg == `` {
			a.log.Error(fmt.Sprintf(`status update is not intended to this member`))
			continue
		}

		strAuthMsg, err := a.probr.ReadMessage(models.Message{Type: domain.MsgTypGroupStatus, Data: []byte(validMsg)})
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading status didcomm message failed - %v`, err))
			continue
		}

		var member models.Member
		if err = json.Unmarshal([]byte(strAuthMsg), &member); err != nil {
			a.log.Error(fmt.Sprintf(`unmarshalling member message failed - %v`, err))
			continue
		}

		if !member.Active {
			a.subs.delete(sm.Topic, member.Label)
			a.gs.deleteMembr(sm.Topic, member.Label)
			a.outChan <- member.Label + ` left group ` + sm.Topic
			continue
		}

		a.gs.addMembr(sm.Topic, member)
		a.log.Debug(fmt.Sprintf(`group state updated for member %s in topic %s`, member.Label, sm.Topic))
	}
}

func (a *Agent) msgListnr() {
	for {
		msg, err := a.sktMsgs.Recv(0)
		if err != nil {
			a.log.Error(fmt.Sprintf(`receiving subscribed message failed - %v`, err))
			continue
		}

		frames := strings.Split(msg, ` `)
		if len(frames) != 2 {
			a.log.Error(fmt.Sprintf(`received an invalid subscribed message (%v) - %v`, frames, err))
			continue
		}

		_, err = a.probr.ReadMessage(models.Message{Type: domain.MsgTypData, Data: []byte(frames[1])})
		if err != nil {
			a.log.Error(fmt.Sprintf(`reading subscribed message failed - %v`, err))
		}
	}
}

// notifyAll constructs a single status message with different didcomm
// messages packed per each member of the group and includes in a map with
// exchange ID as the key.
func (a *Agent) notifyAll(topic string, active, publisher bool) error {
	comprsd, err := a.compressStatus(topic, active, publisher)
	if err != nil {
		return fmt.Errorf(`compress status failed - %v`, err)
	}
	a.log.Trace(fmt.Sprintf(`compressed status message (length=%d)`, len(comprsd)))

	if _, err = a.sktPub.SendMessage(fmt.Sprintf(`%s %s`, a.stateTopic(topic), string(comprsd))); err != nil {
		return fmt.Errorf(`publishing active status failed - %v`, err)
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

	mems := a.gs.membrs(topic)
	for _, m := range mems {
		if m.Label == a.myLabel {
			continue
		}

		//pr, err := a.probr.Peer(m.Label)
		//if err != nil {
		//	return nil, fmt.Errorf(`no peer found - %v`, err)
		//}
		//var sk []byte
		//for _, s := range pr.Services {
		//	if s.Type == domain.ServcGroupJoin {
		//		sk = s.PubKey
		//	}
		//}

		s, pr, err := a.serviceInfo(m.Label)
		if err != nil {
			return nil, fmt.Errorf(`fetching service info failed for peer %s - %v`, m.Label, err)
		}

		data, err := a.pack(m.Label, s.PubKey, byts)
		if err != nil {
			return nil, fmt.Errorf(`packing message for %s failed - %v`, m.Label, err)
		}

		sm.AuthMsgs[pr.ExchangeThId] = string(data)
	}

	encodedStatus, err := json.Marshal(sm)
	if err != nil {
		return nil, fmt.Errorf(`marshalling status message failed - %v`, err)
	}

	return a.zEncodr.EncodeAll(encodedStatus, make([]byte, 0, len(encodedStatus))), nil
}

func (a *Agent) serviceInfo(peer string) (*models.Service, *models.Peer, error) {
	pr, err := a.probr.Peer(peer)
	if err != nil {
		return nil, nil, fmt.Errorf(`no peer found - %v`, err)
	}

	var srvc *models.Service
	for _, s := range pr.Services {
		if s.Type == domain.ServcGroupJoin {
			srvc = &s
		}
	}

	return srvc, &pr, nil
}

func (a *Agent) extractStatus(msg string) (*messages.Status, error) {
	out, err := a.zDecodr.DecodeAll([]byte(msg), nil)
	if err != nil {
		return nil, fmt.Errorf(`decode error - %v`, err)
	}

	var sm messages.Status
	if err = json.Unmarshal(out, &sm); err != nil {
		return nil, fmt.Errorf(`unmarshal error - %v`, err)
	}

	return &sm, nil
}

// pack constructs and encodes an authcrypt message to the given receiver
func (a *Agent) pack(receiver string, recPubKey []byte, msg []byte) ([]byte, error) {
	ownPubKey, err := a.km.PublicKey(receiver)
	if err != nil {
		return nil, fmt.Errorf(`getting public key for connection with %s failed - %v`, receiver, err)
	}

	ownPrvKey, err := a.km.PrivateKey(receiver)
	if err != nil {
		return nil, fmt.Errorf(`getting private key for connection with %s failed - %v`, receiver, err)
	}

	encryptdMsg, err := a.packer.Pack(msg, recPubKey, ownPubKey, ownPrvKey)
	if err != nil {
		return nil, fmt.Errorf(`packing error - %v`, err)
	}

	data, err := json.Marshal(encryptdMsg)
	if err != nil {
		return nil, fmt.Errorf(`marshalling packed message failed - %v`, err)
	}

	return data, nil
}

// constructs the URN in the format of 'urn:<NID>:<NSS>' (https://www.rfc-editor.org/rfc/rfc2141#section-2)
func (a *Agent) internalTopic(topic, pub, sub string) string {
	return `urn:didcomm-queue:` + topic + `:data:` + pub + `:` + sub
}

func (a *Agent) stateTopic(topic string) string {
	return `urn:didcomm-queue:` + topic + `:state`
}

func (a *Agent) parseMembrStatus(msg string) (*messages.Status, error) {
	frames := strings.Split(msg, " ")
	if len(frames) != 2 {
		return nil, fmt.Errorf(`received a message (%v) with an invalid format - frame count should be 2`, msg)
	}

	var ms messages.Status
	if err := json.Unmarshal([]byte(frames[1]), &ms); err != nil {
		return nil, fmt.Errorf(`unmarshalling publisher status failed (msg: %s) - %v`, frames[1], err)
	}

	return &ms, nil
}

// dummy validation for PoC
func (a *Agent) validJoiner(label string) bool {
	return true
}

func (a *Agent) Leave(topic string) error {
	if err := a.sktState.SetUnsubscribe(a.stateTopic(topic)); err != nil {
		return fmt.Errorf(`unsubscribing %s via zmq socket failed - %v`, a.stateTopic(topic), err)
	}

	membrs := a.gs.membrs(topic)
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

	a.subs.deleteTopic(topic)
	a.gs.deleteTopic(topic)
	a.outChan <- `Left group ` + topic
	return nil
}

func (a *Agent) Info(topic string) []models.Member {
	return a.gs.membrs(topic)
}

func (a *Agent) Close() error {
	// close all sockets
	return nil
}
