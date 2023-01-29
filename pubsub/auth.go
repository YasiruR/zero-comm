package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	zmq "github.com/pebbe/zmq4"
)

const (
	domainGlobal = `global`
)

type auth struct {
	id    string
	ctx   *zmq.Context
	servr struct {
		pub  string
		prvt string
	}
	client struct {
		pub  string
		prvt string
	}
}

func initAuthenticator(zmqCtx *zmq.Context, label string, verbose bool) (*auth, error) {
	// check zmq version and return if curve is available
	zmq.AuthSetVerbose(verbose)
	if err := zmq.AuthStart(); err != nil {
		return nil, fmt.Errorf(`starting zmq authenticator failed - %v`, err)
	}

	a := &auth{ctx: zmqCtx, id: label}
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
	zmq.AuthCurveAdd(domainGlobal, zmq.CURVE_ALLOW_ANY) // todo change allow any to client keys

	return nil
}

func (a *auth) Allow(ip string) {
	zmq.AuthAllow(domainGlobal, ip) // todo move this to add member (might cause problems for further didexchanges)
}

func (a *auth) Deny() {
	// blacklist address
}

func (a *auth) pubSkt() (*zmq.Socket, error) {
	skt, err := a.ctx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf(`creating zmq pub socket failed - %v`, err)
	}

	if err = skt.SetIdentity(a.id); err != nil {
		return nil, fmt.Errorf(`setting socket identity failed - %v`, err)
	}

	if err = skt.ServerAuthCurve(domainGlobal, a.servr.prvt); err != nil {
		return nil, fmt.Errorf(`setting curve authentication to zmq server socket failed - %v`, err)
	}

	return skt, nil
}

func (a *auth) setAuthClient(skt *zmq.Socket, servPubKey string) error {
	if err := skt.ClientAuthCurve(servPubKey, a.client.pub, a.client.prvt); err != nil {
		return fmt.Errorf(`setting curve authentication to zmq client socket failed - %v`, err)
	}
	return nil
}

func (a *auth) Close() error {
	zmq.AuthStop()
	return nil
}
