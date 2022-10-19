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
