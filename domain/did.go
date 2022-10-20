package domain

type Invitation struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	From string `json:"from"`
	Body struct {
		GoalCode string   `json:"goal_code"`
		Goal     string   `json:"goal"`
		Accept   []string `json:"accept"`
	} `json:"body"`
	Attachments interface{}
	Services    []Service `json:"services"` // deprecated in v2 but used for simplicity
}

type PeerDIDDoc struct {
	Context       string `json:"@context"`
	PublicKey     []key  `json:"publicKey"`
	Authorization auth   `json:"authorization"`
}

type key struct {
	Crv string `json:"crv"`
	Kty string `json:"kty"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
}

type auth struct {
	Profiles []struct {
		Key   string   `json:"key"`
		Roles []string `json:"roles"`
	} `json:"profiles"`
	Rules []struct {
		Grant []string `json:"grant"`
		When  struct {
			Roles string `json:"roles"`
		} `json:"when"`
	}
}

type DIDDocument struct {
	Context []string  `json:"@context"`
	Id      string    `json:"id"`
	Service []Service `json:"service"`
}

type Service struct {
	Id              string   `json:"id"`
	Type            string   `json:"type"`
	RecipientKeys   []string `json:"recipientKeys"`
	RoutingKeys     []string `json:"routingKeys"`
	ServiceEndpoint string   `json:"serviceEndpoint"`
	Accept          []string `json:"accept"`
}
