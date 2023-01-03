package models

type Message struct {
	Type  string
	Data  []byte
	Reply chan []byte
}

type Connection struct {
	Peer     string
	Endpoint string
	PubKey   []byte
}

type Peer struct {
	Active       bool // todo (currently all stored peers are active since no disconnect is implemented)
	DID          string
	ExchangeThId string // thread id used in did-exchange (to correlate any message to the peer)
	Services     []Service
}

type Feature struct {
	Id    string   `json:"id"`
	Roles []string `json:"roles"`
}

type Service struct {
	Id       string
	Type     string
	Endpoint string
	PubKey   []byte
}

type Member struct {
	Active      bool   `json:"active"`
	Label       string `json:"label"`
	Inv         string `json:"inv"`
	PubEndpoint string `json:"pubEndpoint"`
}
