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

func (h *Handler) CreateConnReq(name, pthid, did string, encryptedDoc []byte) domain.ConnReq {
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
	req.DIDDocAttach.Data.Base64 = base64.StdEncoding.EncodeToString(encryptedDoc)

	return req
}
