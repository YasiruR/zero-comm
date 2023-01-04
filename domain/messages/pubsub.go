package messages

import "github.com/YasiruR/didcomm-prober/domain/models"

type PublisherStatus struct {
	Id     string `json:"@id"`
	Type   string `json:"@type"`
	Label  string `json:"label"` // todo check if DID can be used
	Active bool   `json:"active"`
	Inv    string `json:"inv"`
	Topic  string `json:"topic"` // might be a redundant info in general mq systems
}

type SubscribeMsg struct {
	Id          string   `json:"@id"`
	Type        string   `json:"@type"`
	Subscribe   bool     `json:"subscribe"`
	Peer        string   `json:"peer"`
	PubKey      string   `json:"pubKey"` // base58 encoding of public key
	Topics      []string `json:"topics"`
	PubEndpoint string   `json:"pubEndpoint"`

	Topic string `json:"topic"`
}

type SubscribeMsgNew struct {
	Id        string        `json:"@id"`
	Type      string        `json:"@type"`
	Subscribe bool          `json:"subscribe"`
	PubKey    string        `json:"pubKey"` // base58 encoding of public key
	Topic     string        `json:"topic"`
	Member    models.Member `json:"member"`
}

type Status struct {
	Id     string        `json:"@id"`
	Type   string        `json:"@type"`
	Topic  string        `json:"topic"` // might be a redundant info in general mq systems
	Member models.Member `json:"member"`
}

type ReqGroupJoin struct {
	Label        string `json:"label"`
	Topic        string `json:"topic"`
	RequesterInv string `json:"requesterInv"`
}

type ResGroupJoin struct {
	Members []models.Member `json:"members"` // includes acceptor
}
