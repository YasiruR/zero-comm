package transport

import (
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
)

type Initiator struct {
	Role  domain.Role `json:"role"`
	Label string      `json:"label"`
}

type Reply struct {
	Id   string     `json:"id"`
	Chan chan error `json:"-"`
}

type ConnectMsg struct {
	Connect   bool          `json:"connect"`
	Initiator Initiator     `json:"initiator"`
	Topic     string        `json:"topic"`
	Peer      models.Member `json:"peer"`
	Reply     struct {
		State Reply `json:"state"`
		Data  Reply `json:"data"`
	} `json:"reply"`
}

type PublishMsg struct {
	Topic string `json:"topic"`
	Data  []byte `json:"data"`
	Reply Reply  `json:"reply"`
}

type SubscribeMsg struct {
	Subscribe bool          `json:"subscribe"`
	UnsubAll  bool          `json:"unsubAll"`
	MyLabel   string        `json:"myLabel"`
	Topic     string        `json:"topic"`
	State     bool          `json:"state"`
	Data      bool          `json:"data"`
	Peer      models.Member `json:"peer"`
	Reply     struct {
		State Reply `json:"state"`
		Data  Reply `json:"data"`
	} `json:"reply"`
}

type AuthMsg struct {
	Label        string `json:"label"`
	ServrPubKey  string `json:"servrPubKey"`
	ClientPubKey string `json:"clientPubKey"`
	Data         bool   `json:"data"`
	Reply        struct {
		State Reply `json:"state"`
		Data  Reply `json:"data"`
	} `json:"reply"`
}
