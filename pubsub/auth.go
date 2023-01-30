package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmq "github.com/pebbe/zmq4"
	"sync"
)

const (
	domainGlobal = `global`
)

type zmqPeerKeys struct {
	servr  string
	client string
}

type auth struct {
	id       string
	sktMsgs  *zmq.Socket
	sktState *zmq.Socket
	servr    struct {
		pub  string
		prvt string
	}
	client struct {
		pub  string
		prvt string
	}
	keys *sync.Map // transport public keys of other members
	*sync.RWMutex
}

func initAuthenticator(label string, verbose bool) (*auth, error) {
	// check zmq version and return if curve is available
	zmq.AuthSetVerbose(verbose)
	if err := zmq.AuthStart(); err != nil {
		return nil, fmt.Errorf(`starting zmq authenticator failed - %v`, err)
	}

	a := &auth{id: label}
	if err := a.generateCerts(models.Metadata{}); err != nil {
		return nil, fmt.Errorf(`initializing certficates failed - %v`, err)
	}

	return a, nil
}

func (a *auth) generateCerts(md models.Metadata) error {
	servPub, servPrvt, err := zmq.NewCurveKeypair()
	if err != nil {
		return fmt.Errorf(`generating curve key pair for server failed - %v`, err)
	}

	clientPub, clientPrvt, err := zmq.NewCurveKeypair()
	if err != nil {
		return fmt.Errorf(`generating curve key pair for client failed - %v`, err)
	}

	a.servr.pub, a.servr.prvt = servPub, servPrvt
	a.client.pub, a.client.prvt = clientPub, clientPrvt

	return nil
}

func (a *auth) initSkts(zmqCtx *zmq.Context) (*wsockets, error) {
	pubSkt, err := zmqCtx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	if err = pubSkt.SetIdentity(a.id); err != nil {
		return nil, fmt.Errorf(`setting socket identity failed - %v`, err)
	}

	if err = pubSkt.ServerAuthCurve(domainGlobal, a.servr.prvt); err != nil {
		return nil, fmt.Errorf(`setting curve authentication to zmq server socket failed - %v`, err)
	}

	sktStates, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for status topic failed - %v`, err)
	}

	sktMsgs, err := zmqCtx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf(`creating sub socket for data topics failed - %v`, err)
	}

	a.sktState = sktStates
	a.sktMsgs = sktMsgs

	return &wsockets{pub: pubSkt, state: sktStates, msgs: sktMsgs}, nil
}

func (a *auth) setAuthn(peer, servPubKey, clientPubKey string, publisher bool) error {
	zmq.AuthCurveAdd(domainGlobal, clientPubKey)
	if err := a.sktState.ClientAuthCurve(servPubKey, a.client.pub, a.client.prvt); err != nil {
		return fmt.Errorf(`setting curve client authentication to zmq state socket failed - %v`, err)
	}

	if publisher {
		if err := a.sktMsgs.ClientAuthCurve(servPubKey, a.client.pub, a.client.prvt); err != nil {
			return fmt.Errorf(`setting curve client authentication to zmq data socket failed - %v`, err)
		}
	}

	a.Lock()
	defer a.Unlock()
	a.keys.Store(peer, zmqPeerKeys{servr: servPubKey, client: clientPubKey})

	return nil
}

func (a *auth) remvKeys(peer string) error {
	a.Lock()
	defer a.Unlock()
	val, ok := a.keys.Load(peer)
	if !ok {
		return fmt.Errorf(`loading transport keys failed`)
	}

	ks, ok := val.(zmqPeerKeys)
	if !ok {
		return fmt.Errorf(`incomaptible type found for transport keys (%v)`, val)
	}

	zmq.AuthCurveRemove(domainGlobal, ks.client)
	a.keys.Delete(peer)

	return nil
}

func (a *auth) Close() error {
	zmq.AuthStop()
	return nil
}
