package models

import "github.com/YasiruR/didcomm-prober/domain"

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

type GroupParams struct {
	OrderEnabled   bool             `json:"ordered"`
	JoinConsistent bool             `json:"consistent_join"`
	Mode           domain.GroupMode `json:"mode"`
}

type Group struct {
	Topic    string
	Checksum string
	Members  map[string]Member // map is used to prevent multiple instances of a member
	*GroupParams
	LastUpdated int64
}
