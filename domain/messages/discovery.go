package messages

type FeatQuery struct {
	Type    string `json:"@type"`
	Id      string `json:"@id"`
	Query   string `json:"query"`
	Comment string `json:"comment"`
}

type FeatDisclose struct {
	Type   string `json:"@type"`
	Thread struct {
		ThId string `json:"@thid"`
	} `json:"~thread"`
	Protocols []Protocol `json:"protocols"`
}

type Protocol struct {
	PId   string   `json:"pid"`
	Roles []string `json:"roles"`
}
