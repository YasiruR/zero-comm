package domain

type Peer struct {
	DID          string
	Endpoint     string
	PubKey       []byte
	ExchangeThId string // thread id used in did-exchange (to correlate any message to the peer)
}
