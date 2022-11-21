package reqrep

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/gorilla/mux"
	"github.com/tryfix/log"
	"io/ioutil"
	"net/http"
	"strconv"
)

type HTTP struct {
	port   int
	packer services.Packer
	ks     services.KeyManager
	log    log.Logger
	router *mux.Router
	client *http.Client
	inChan chan []byte
}

func NewHTTP(c *domain.Container) *HTTP {
	return &HTTP{
		port:   c.Cfg.Args.Port,
		packer: c.Packer,
		ks:     c.KS,
		log:    c.Log,
		client: &http.Client{},
		router: mux.NewRouter(),
		//inChan: c.InChan,
	}
}

func (h *HTTP) Start() {
	h.router.HandleFunc(domain.InvitationEndpoint, h.handleConnReqs).Methods(http.MethodPost)
	h.router.HandleFunc(domain.ExchangeEndpoint, h.handleInbound).Methods(http.MethodPost)
	if err := http.ListenAndServe(":"+strconv.Itoa(h.port), h.router); err != nil {
		h.log.Fatal(`http server initialization failed - %v`, err)
	}
}

func (h *HTTP) Send(data []byte, endpoint string) error {
	res, err := h.client.Post(endpoint, `application/json`, bytes.NewBuffer(data))
	if err != nil {
		h.log.Error(err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted || res.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf(`invalid status code: %d`, res.StatusCode)
}

func (h *HTTP) handleConnReqs(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.log.Error(err)
		return
	}

	h.inChan <- data
}

func (h *HTTP) handleInbound(_ http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.log.Error(err)
		return
	}

	h.inChan <- data
}

func (h *HTTP) Stop() error {
	return nil
}
