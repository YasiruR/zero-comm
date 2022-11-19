package messages

type PublisherStatus struct {
	Active bool   `json:"active"`
	Inv    string `json:"inv"`
	Topic  string `json:"topic"` // might be a redundant info in general mq systems
}

type SubscribeMsg struct {
	Subscribe bool     `json:"subscribe"`
	Peer      string   `json:"peer"`
	PubKey    string   `json:"pubKey"` // base58 encoding of public key
	Topics    []string `json:"topics"`
}
