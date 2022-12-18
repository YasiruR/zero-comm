package messages

import "github.com/YasiruR/didcomm-prober/domain/models"

type QueryFeature struct {
	Type    string `json:"@type"`
	Id      string `json:"@id"`
	Query   string `json:"query"`
	Comment string `json:"comment"`
}

type DiscloseFeature struct {
	Type   string `json:"@type"`
	Thread struct {
		ThId string `json:"@thid"`
	} `json:"~thread"`
	Features []models.Feature `json:"features"`
}
