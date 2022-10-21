package did

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
)

type Handler struct{}

func (h *Handler) CreateDIDDoc(endpoint, typ string, encPubKey []byte) domain.DIDDocument {
	s := domain.Service{
		Id:              uuid.New().String(),
		Type:            typ,
		RecipientKeys:   []string{string(encPubKey)},
		RoutingKeys:     nil,
		ServiceEndpoint: endpoint,
		Accept:          nil,
	}

	return domain.DIDDocument{Service: []domain.Service{s}}
}

func (h *Handler) CreatePeerDID(doc domain.DIDDocument) (did string, err error) {
	// make a did-doc but omit DID value from doc = stored variant
	byts, err := json.Marshal(doc)
	if err != nil {
		return ``, fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// compute sha256 hash of stored variant = numeric basis
	hash := sha256.New()
	if _, err = hash.Write(byts); err != nil {
		return ``, fmt.Errorf(`generating sha256 hash of did doc failed - %v`, err)
	}

	// base58 encode numeric basis
	enc := base58.Encode(hash.Sum(nil))
	// did:peer:1z<encoded-numeric-basis>
	return `did:peer:1z` + enc, nil
}

func (h *Handler) ValidatePeerDID(did string) error {
	if len(did) < 11 {
		return fmt.Errorf(`invalid did in invitation: %s`, did)
	}

	// should ideally use a regex
	if did[:11] != `did:peer:1z` {
		return fmt.Errorf(`did type is not peer: %s`, did[:11])
	}

	return nil
}

func (h *Handler) CreateConnReq(pthid, did string, encDidDoc domain.AuthCryptMsg) (domain.ConnReq, error) {
	id := uuid.New().String()
	req := domain.ConnReq{
		Id:   id,
		Type: "https://didcomm.org/didexchange/1.0/request",
		Thread: struct {
			ThId  string `json:"thid"`
			PThId string `json:"pthid"`
		}{ThId: id, PThId: pthid},
		Label: `test-label`,
		Goal:  "connection establishment",
		DID:   did,
	}

	// marshals the encrypted did doc
	encDocBytes, err := json.Marshal(encDidDoc)
	if err != nil {
		return domain.ConnReq{}, fmt.Errorf(`marshalling encrypted did doc failed - %v`, err)
	}

	req.DIDDocAttach.Id = uuid.New().String()
	req.DIDDocAttach.MimeType = `application/json`
	req.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encDocBytes)

	return req, nil
}

func (h *Handler) ParseConnReq(data []byte) (thId string, encDocBytes []byte, err error) {
	var req domain.ConnReq
	if err = json.Unmarshal(data, &req); err != nil {
		return ``, nil, fmt.Errorf(`unmarshalling connection request failed - %v`, err)
	}

	encDocBytes, err = base64.StdEncoding.DecodeString(req.DIDDocAttach.Data.Base64)
	if err != nil {
		return ``, nil, fmt.Errorf(`decoding did doc failed - %v`, err)
	}

	//if err = json.Unmarshal(encDocBytes, &encDidDoc); err != nil {
	//	return ``, nil, fmt.Errorf(`unmarshalling encrypted did doc failed - %v`, err)
	//}

	return req.Thread.ThId, encDocBytes, nil
}

func (h *Handler) CreateConnRes(thId, did string, encDidDoc domain.AuthCryptMsg) (domain.ConnRes, error) {
	res := domain.ConnRes{
		Id:   uuid.New().String(),
		Type: "https://didcomm.org/didexchange/1.0/response",
		Thread: struct {
			ThId string `json:"thid"`
		}{ThId: thId},
		DID: did,
	}

	// marshals the encrypted did doc
	encDocBytes, err := json.Marshal(encDidDoc)
	if err != nil {
		return domain.ConnRes{}, fmt.Errorf(`marshalling encrypted did doc failed - %v`, err)
	}

	res.DIDDocAttach.Id = uuid.New().String()
	res.DIDDocAttach.MimeType = `application/json`
	res.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encDocBytes)

	return res, nil
}
