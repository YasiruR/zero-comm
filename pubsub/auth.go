package pubsub

import (
	"fmt"
	zmqPkg "github.com/pebbe/zmq4"
	"sync"
)

const (
	domainGlobal = `global`
)

type metadata map[string]string

type zmqPeerKeys struct {
	servr  string
	client string
}

// auth serves as a transport layer authenticator to prevent unauthorized
// connections to PUB and SUB sockets
type auth struct {
	id    string
	servr struct {
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

func authenticator(label string, verbose bool) (*auth, error) {
	// check zmq version and return if curve is available
	zmqPkg.AuthSetVerbose(verbose)
	if err := zmqPkg.AuthStart(); err != nil {
		if err.Error() != "Auth is already running" {
			return nil, fmt.Errorf(`starting zmq authenticator failed - %v`, err)
		}
	}

	a := &auth{id: label, keys: &sync.Map{}, RWMutex: &sync.RWMutex{}}
	if err := a.generateCerts(metadata{}); err != nil {
		return nil, fmt.Errorf(`initializing certficates failed - %v`, err)
	}

	return a, nil
}

func (a *auth) generateCerts(_ metadata) error {
	servPub, servPrvt, err := zmqPkg.NewCurveKeypair()
	if err != nil {
		return fmt.Errorf(`generating curve key pair for server failed - %v`, err)
	}

	clientPub, clientPrvt, err := zmqPkg.NewCurveKeypair()
	if err != nil {
		return fmt.Errorf(`generating curve key pair for client failed - %v`, err)
	}

	a.servr.pub, a.servr.prvt = servPub, servPrvt
	a.client.pub, a.client.prvt = clientPub, clientPrvt

	return nil
}

func (a *auth) setPubAuthn(skt *zmqPkg.Socket) error {
	if err := skt.SetIdentity(a.id); err != nil {
		return fmt.Errorf(`setting socket identity failed - %v`, err)
	}

	if err := skt.ServerAuthCurve(domainGlobal, a.servr.prvt); err != nil {
		return fmt.Errorf(`setting curve authentication to zmq server socket failed - %v`, err)
	}

	return nil
}

func (a *auth) setPeerAuthn(peer, servPubKey, clientPubKey string, sktState, sktMsgs *zmqPkg.Socket) error {
	zmqPkg.AuthCurveAdd(domainGlobal, clientPubKey)
	if err := sktState.ClientAuthCurve(servPubKey, a.client.pub, a.client.prvt); err != nil {
		return fmt.Errorf(`setting curve client authentication to zmq state socket failed - %v`, err)
	}

	if sktMsgs != nil {
		if err := sktMsgs.ClientAuthCurve(servPubKey, a.client.pub, a.client.prvt); err != nil {
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

	zmqPkg.AuthCurveRemove(domainGlobal, ks.client)
	a.keys.Delete(peer)

	return nil
}

func (a *auth) close() error {
	zmqPkg.AuthStop()
	return nil
}
