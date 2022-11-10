package messages

type PublisherStatus struct {
	Active bool   `json:"active"`
	Inv    string `json:"inv"`
}

type SubscribeMsg struct {
	Label  string   `json:"label"`
	Topics []string `json:"topics"`
}
