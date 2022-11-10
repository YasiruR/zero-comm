package messages

type PublisherStatus struct {
	Active bool   `json:"active"`
	Inv    string `json:"inv"`
}

type SubscribeMsg struct {
	Peer   string   `json:"peer"`
	PubKey string   `json:"pubKey"` // base58 encoding of public key
	Topics []string `json:"topics"`
}
