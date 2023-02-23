package models

type Peer struct {
	Active       bool // currently all stored peers are active since no disconnect is implemented
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
	Publisher   bool   `json:"publisher"`
	Label       string `json:"label"` // todo check if DID can be used
	Inv         string `json:"inv"`
	PubEndpoint string `json:"pubEndpoint"`
}

// todo check if group struct is required
