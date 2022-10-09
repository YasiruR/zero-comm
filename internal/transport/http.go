package transport

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/tryfix/log"
	"io/ioutil"
	"net/http"
	"strconv"
)

type HTTP struct {
	port   int
	router *mux.Router
	client *http.Client
	logger log.Logger // remove later
}

func NewHTTP(port int, logger log.Logger) *HTTP {
	return &HTTP{port: port, router: mux.NewRouter(), client: &http.Client{}, logger: logger}
}

func (h *HTTP) Start() {
	h.router.HandleFunc(`/`, h.handleInbound).Methods(http.MethodPost)
	h.logger.Info(fmt.Sprintf("http server started listening on %d", h.port))
	if err := http.ListenAndServe(":"+strconv.Itoa(h.port), h.router); err != nil {
		h.logger.Fatal(err)
	}
}

func (h *HTTP) Send(data []byte, endpoint string) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(data))
	if err != nil {
		h.logger.Error(err)
		return err
	}

	res, err := h.client.Do(req)
	if err != nil {
		h.logger.Error(err)
		return err
	}

	if res.StatusCode == http.StatusAccepted || res.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf(`invalid status code: %d`, res.StatusCode)
}

func (h *HTTP) handleInbound(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.logger.Error(err)
		return
	}
}

func (h *HTTP) Stop() error {
	return nil
}
