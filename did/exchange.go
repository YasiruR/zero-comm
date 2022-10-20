package did

import (
	"encoding/base64"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/google/uuid"
)

func CreateConnReq(name, pthid, did string, encDoc []byte) domain.ConnReq {
	id := uuid.New().String()
	req := domain.ConnReq{
		Id:   id,
		Type: "https://didcomm.org/didexchange/1.0/request",
		Thread: struct {
			ThId  string `json:"thid"`
			PThId string `json:"pthid"`
		}{ThId: id, PThId: pthid},
		Label: name,
		Goal:  "connection establishment",
		DID:   did,
	}

	req.DIDDocAttach.Id = uuid.New().String()
	req.DIDDocAttach.MimeType = `application/json`
	req.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encDoc)
	
	return req
}
