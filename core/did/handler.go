package did

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/btcsuite/btcutil/base58"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) CreateDIDDoc(svcs []models.Service) messages.DIDDocument {
	var msgSvcs []messages.Service
	for _, svc := range svcs {
		encodedKey := make([]byte, 64)
		base64.StdEncoding.Encode(encodedKey, svc.PubKey)
		// removes redundant elements from the allocated byte slice
		encodedKey = bytes.Trim(encodedKey, "\x00")

		s := messages.Service{
			Id:              svc.Id,
			Type:            svc.Type,
			RecipientKeys:   []string{string(encodedKey)},
			RoutingKeys:     nil,
			ServiceEndpoint: svc.Endpoint,
			Accept:          nil,
		}
		msgSvcs = append(msgSvcs, s)
	}

	return messages.DIDDocument{Service: msgSvcs}
}

func (h *Handler) CreatePeerDID(doc messages.DIDDocument) (did string, err error) {
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
