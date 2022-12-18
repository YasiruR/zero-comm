package messages

// Invitation reference: https://identity.foundation/didcomm-messaging/spec/#invitation
type Invitation struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	From  string `json:"from"`
	Label string `json:"label"` // from aries rfc-0160
	Body  struct {
		GoalCode string   `json:"goal_code"`
		Goal     string   `json:"goal"`
		Accept   []string `json:"accept"`
	} `json:"body"`
	Attachments interface{}
	Services    []Service `json:"services"` // deprecated in v2 but used for simplicity
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

// ConnReq reference: https://github.com/hyperledger/aries-rfcs/tree/main/features/0023-did-exchange#request-message-example
type ConnReq struct {
	Id     string `json:"@id"`
	Type   string `json:"@type"`
	Thread struct {
		ThId string `json:"thid"`
		// should contain the id of the corresponding invitation (https://github.com/hyperledger/aries-rfcs/tree/main/features/0023-did-exchange#request-message-attributes)
		PThId string `json:"pthid"`
	} `json:"~thread"`
	Label        string `json:"label"`
	GoalCode     string `json:"goal_code"`
	Goal         string `json:"goal"`
	DID          string `json:"did"`
	DIDDocAttach struct {
		Id       string `json:"@id"`
		MimeType string `json:"mime-type"`
		Data     struct {
			Base64 string `json:"base64"`
		}
	} `json:"did_doc~attach"`
}

// ConnRes reference: https://github.com/hyperledger/aries-rfcs/tree/main/features/0023-did-exchange#response-message-example
type ConnRes struct {
	Id     string `json:"@id"`
	Type   string `json:"@type"`
	Thread struct {
		// must be a reference to the request message (https://github.com/hyperledger/aries-rfcs/tree/main/features/0023-did-exchange#response-message-attributes)
		ThId string `json:"thid"`
	} `json:"~thread"`
	DID          string `json:"did"`
	DIDDocAttach struct {
		Id       string `json:"@id"`
		MimeType string `json:"mime-type"`
		Data     struct {
			Base64 string `json:"base64"`
		}
	} `json:"did_doc~attach"`
}
