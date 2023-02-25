package messages

import (
	"github.com/YasiruR/didcomm-prober/domain/models"
)

type Subscribe struct {
	Id        string        `json:"@id"`
	Type      string        `json:"@type"`
	Subscribe bool          `json:"subscribe"`
	PubKey    string        `json:"pubKey"` // base58 encoding of public key
	Topic     string        `json:"topic"`
	Member    models.Member `json:"member"`
	Transport Transport     `json:"transport"`
}

type ResSubscribe struct {
	Publisher bool      `json:"publisher"`
	Transport Transport `json:"transport"`
	Checksum  string    `json:"checksum"`
}

type Transport struct {
	ServrPubKey  string `json:"servr_pub_key"`
	ClientPubKey string `json:"client_pub_key"`
}

type Status struct {
	Id       string            `json:"@id"`
	Type     string            `json:"@type"`
	Topic    string            `json:"topic"` // might be a redundant info in general mq systems
	AuthMsgs map[string]string `json:"auth_msgs"`
}

type ReqGroupJoin struct {
	Id           string `json:"@id"`
	Type         string `json:"@type"`
	Label        string `json:"label"`
	Topic        string `json:"topic"`
	RequesterInv string `json:"requesterInv"`
}

type ResGroupJoin struct {
	Id      string             `json:"@id"`
	Type    string             `json:"@type"`
	Params  models.GroupParams `json:"params"`
	Members []models.Member    `json:"members"` // includes acceptor
}
