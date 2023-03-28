package mock

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/gorilla/mux"
	"github.com/tryfix/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type mocker struct {
	ctr *container.Container
	log log.Logger
}

func Start(c *container.Container) {
	m := mocker{
		ctr: c,
		log: c.Log,
	}

	r := mux.NewRouter()
	r.HandleFunc(PingEndpoint, m.handlePing).Methods(http.MethodGet)
	r.HandleFunc(InvEndpoint, m.handleInv).Methods(http.MethodGet)
	r.HandleFunc(ConnectEndpoint, m.handleConnect).Methods(http.MethodPost)
	r.HandleFunc(CreateEndpoint, m.handleCreate).Methods(http.MethodPost)
	r.HandleFunc(JoinEndpoint, m.handleJoin).Methods(http.MethodPost)
	r.HandleFunc(KillEndpoint, m.handleKill).Methods(http.MethodPost)
	r.HandleFunc(GrpMsgAckEndpoint, m.handleGrpMsgListnr).Methods(http.MethodPost)

	go func(mockPort int, r *mux.Router) {
		if err := http.ListenAndServe(":"+strconv.Itoa(mockPort), r); err != nil {
			m.log.Fatal(`mocker`, fmt.Sprintf(`http server initialization failed - %v`, err))
		}
	}(c.Cfg.MockPort, r)

	c.Log.Info(fmt.Sprintf(`mock server initialized and started listening on %d`, c.Cfg.MockPort))
}

func (m *mocker) handlePing(_ http.ResponseWriter, _ *http.Request) {
	return
}

func (m *mocker) handleInv(w http.ResponseWriter, _ *http.Request) {
	m.log.Trace(`mocker received a request for invitation`)
	inv, err := m.ctr.Prober.Invite()
	if err != nil {
		m.log.Error(err)
		return
	}

	if _, err = w.Write([]byte(inv)); err != nil {
		m.log.Error(err)
	}
	m.log.Trace(`mocker sent invitation`, inv)
}

func (m *mocker) handleConnect(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.log.Error(err)
		return
	}

	u, err := url.Parse(strings.TrimSpace(string(data)))
	if err != nil {
		m.log.Error(`invalid url format, please try again`, err)
		return
	}

	inv, ok := u.Query()[`oob`]
	if !ok {
		m.log.Error(`invitation url must contain 'oob' parameter, please try again`, inv)
		return
	}

	if err = m.ctr.Prober.SyncAccept(inv[0]); err != nil {
		m.log.Error(`invitation may be invalid, please try again`, err)
	}
}

func (m *mocker) handleCreate(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.log.Error(err)
		return
	}

	var req reqCreate
	if err = json.Unmarshal(data, &req); err != nil {
		m.log.Error(err)
		return
	}
	m.log.Trace(`mocker received a request for create endpoint`, req.Topic)

	if err = m.ctr.PubSub.Create(req.Topic, req.Publisher, req.Params); err != nil {
		m.log.Error(err)
	}
}

func (m *mocker) handleJoin(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.log.Error(err)
		return
	}

	var req reqJoin
	if err = json.Unmarshal(data, &req); err != nil {
		m.log.Error(err)
		return
	}

	if err = m.ctr.PubSub.Join(req.Topic, req.Acceptor, req.Publisher); err != nil {
		m.log.Error(err)
	}
}

func (m *mocker) handleGrpMsgListnr(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.log.Error(fmt.Sprintf(`reading body failed - %v`, err))
		return
	}

	var req ReqRegAck
	if err = json.Unmarshal(data, &req); err != nil {
		m.log.Error(fmt.Sprintf(`unmarshalling error - %v`, err))
		return
	}

	var count int
	ackChan := make(chan string)
	m.ctr.PubSub.RegisterAck(req.Peer, ackChan)
	defer m.ctr.PubSub.UnregisterAck(req.Peer)
	for {
		msg := <-ackChan
		if msg != req.Msg {
			continue
		}

		count++
		if count == req.Count {
			m.log.Debug(fmt.Sprintf(`reached message count registered by tester (label=%s, count=%s)`, req.Peer, req.Count))
			return
		}
	}
}

func (m *mocker) handleKill(_ http.ResponseWriter, _ *http.Request) {
	if err := m.ctr.Stop(); err != nil {
		m.log.Error(`terminating container failed`, err)
	}
}
