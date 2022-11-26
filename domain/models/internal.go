package models

type Message struct {
	Type string
	Data []byte
}

type Connection struct {
	Peer     string
	Endpoint string
	PubKey   []byte
}

type Peer struct {
	DID          string
	Endpoint     string
	PubKey       []byte
	ExchangeThId string // thread id used in did-exchange (to correlate any message to the peer)
}

type Feature struct {
	Id    string
	Roles []string
}

type HandlerFunc func(msg Message) error
