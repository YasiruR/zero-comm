package mock

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/gorilla/mux"
	"github.com/tryfix/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type mocker struct {
	c   *domain.Container
	log log.Logger
}

func Start(c *domain.Container) {
	m := mocker{
		c:   c,
		log: c.Log,
	}

	r := mux.NewRouter()
	r.HandleFunc(domain.OOBEndpoint, m.handleOOBInv).Methods(http.MethodPost)
	r.HandleFunc(domain.KillEndpoint, m.handleKill).Methods(http.MethodPost)

	go func(mockPort int, r *mux.Router) {
		if err := http.ListenAndServe(":"+strconv.Itoa(mockPort), r); err != nil {
			m.log.Fatal(`mocker`, fmt.Sprintf(`http server initialization failed - %v`, err))
		}
	}(c.Cfg.MockPort, r)

	c.Log.Info(fmt.Sprintf(`mock server initialized and started listening on %d`, c.Cfg.MockPort))
}

func (m *mocker) handleOOBInv(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.log.Error(`mocker`, err)
		return
	}
	m.log.Trace(`mocker`, `received message`, string(data))

	u, err := url.Parse(strings.TrimSpace(string(data)))
	if err != nil {
		m.log.Error(`mocker`, `invalid url format, please try again`, err)
		return
	}

	inv, ok := u.Query()[`oob`]
	if !ok {
		m.log.Error(`mocker`, `invitation url must contain 'oob' parameter, please try again`, err)
		return
	}

	if err = m.c.Prober.SyncAccept(inv[0]); err != nil {
		m.log.Error(`mocker`, `invitation may be invalid, please try again`, err)
	}
}

func (m *mocker) handleKill(_ http.ResponseWriter, _ *http.Request) {
	if err := m.c.Stop(); err != nil {
		m.log.Error(`mocker`, `terminating container failed`, err)
	}
}
