package connection

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/google/uuid"
)

type Connector struct{}

func NewConnector() *Connector {
	return &Connector{}
}

func (c *Connector) CreateConnReq(label, pthid, did string, encDidDoc messages.AuthCryptMsg) (messages.ConnReq, error) {
	id := uuid.New().String()
	req := messages.ConnReq{
		Id:   id,
		Type: "https://didcomm.org/didexchange/1.0/request",
		Thread: struct {
			ThId  string `json:"thid"`
			PThId string `json:"pthid"`
		}{ThId: id, PThId: pthid},
		Label: label,
		Goal:  "connection establishment",
		DID:   did,
	}

	// marshals the encrypted did doc
	encDocBytes, err := json.Marshal(encDidDoc)
	if err != nil {
		return messages.ConnReq{}, fmt.Errorf(`marshalling encrypted did doc failed - %v`, err)
	}

	req.DIDDocAttach.Id = uuid.New().String()
	req.DIDDocAttach.MimeType = `application/json`
	req.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encDocBytes)

	return req, nil
}

func (c *Connector) ParseConnReq(data []byte) (label, pthId, peerDid string, encDocBytes []byte, err error) {
	var req messages.ConnReq
	if err = json.Unmarshal(data, &req); err != nil {
		return ``, ``, ``, nil, fmt.Errorf(`unmarshalling connection request failed - %v`, err)
	}

	encDocBytes, err = base64.StdEncoding.DecodeString(req.DIDDocAttach.Data.Base64)
	if err != nil {
		return ``, ``, ``, nil, fmt.Errorf(`decoding did doc failed - %v`, err)
	}

	return req.Label, req.Thread.PThId, req.DID, encDocBytes, nil
}

func (c *Connector) CreateConnRes(pthId, did string, encDidDoc messages.AuthCryptMsg) (messages.ConnRes, error) {
	res := messages.ConnRes{
		Id:   uuid.New().String(),
		Type: "https://didcomm.org/didexchange/1.0/response",
		Thread: struct {
			ThId string `json:"thid"`
		}{ThId: pthId},
		DID: did,
	}

	// marshals the encrypted did doc
	encDocBytes, err := json.Marshal(encDidDoc)
	if err != nil {
		return messages.ConnRes{}, fmt.Errorf(`marshalling encrypted did doc failed - %v`, err)
	}

	res.DIDDocAttach.Id = uuid.New().String()
	res.DIDDocAttach.MimeType = `application/json`
	res.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encDocBytes)

	return res, nil
}

func (c *Connector) ParseConnRes(data []byte) (pthId string, encDocBytes []byte, err error) {
	var res messages.ConnRes
	if err = json.Unmarshal(data, &res); err != nil {
		return ``, nil, fmt.Errorf(`unmarshalling connection response failed - %v`, err)
	}

	encDocBytes, err = base64.StdEncoding.DecodeString(res.DIDDocAttach.Data.Base64)
	if err != nil {
		return ``, nil, fmt.Errorf(`decoding did doc failed - %v`, err)
	}

	return res.Thread.ThId, encDocBytes, nil
}
