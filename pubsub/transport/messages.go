package transport

import (
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
)

type Initiator struct {
	Role  domain.Role `json:"role"`
	Label string      `json:"label"`
}

type ConnectMsg struct {
	Connect   bool          `json:"connect"`
	Initiator Initiator     `json:"initiator"`
	Topic     string        `json:"topic"`
	Peer      models.Member `json:"peer"`
}

type PublishMsg struct {
	Topic string `json:"topic"`
	Data  []byte `json:"data"`
}

type SubscribeMsg struct {
	Subscribe bool
	UnsubAll  bool
	MyLabel   string
	Topic     string
	State     bool
	Data      bool
	Peer      models.Member
}

type AuthMsg struct {
	Label        string
	ServrPubKey  string
	ClientPubKey string
	Data         bool
}
