package did

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
)

func CreateInvitation(endpoint string, pubKey []byte) (domain.Invitation, error) {
	did, err := createPeerDID(createDIDDoc(endpoint, pubKey))
	if err != nil {
		return domain.Invitation{}, fmt.Errorf(`creating peer did failed - %v`, err)
	}

	return domain.Invitation{
		Id:   uuid.New().String(),
		Type: "https://didcomm.org/out-of-band/2.0/invitation",
		From: did,
		Body: struct {
			GoalCode string   `json:"goal_code"`
			Goal     string   `json:"goal"`
			Accept   []string `json:"accept"`
		}{},
		Attachments: nil,
	}, nil
}

func createDIDDoc(endpoint string, pubKey []byte) domain.DIDDocument {
	s := domain.Service{
		Id:              uuid.New().String(),
		Type:            "message-service",
		RecipientKeys:   []string{string(pubKey)},
		RoutingKeys:     nil,
		ServiceEndpoint: endpoint,
		Accept:          nil,
	}

	return domain.DIDDocument{Service: []domain.Service{s}}
}

func createPeerDID(doc domain.DIDDocument) (did string, err error) {
	// make a did-doc but omit DID value from doc = stored variant
	bytes, err := json.Marshal(doc)
	if err != nil {
		return ``, fmt.Errorf(`marshalling did doc failed - %v`, err)
	}

	// compute sha256 hash of stored variant = numeric basis
	h := sha256.New()
	if _, err = h.Write(bytes); err != nil {
		return ``, fmt.Errorf(`generating sha256 hash of did doc failed - %v`, err)
	}

	// base58 encode numeric basis
	enc := base58.Encode(h.Sum(nil))
	// did:peer:1z<encoded-numeric-basis>
	return `did:peer:1z` + enc, nil
}

func ParseInvitation(data []byte) (endpoint string, pubKey []byte, err error) {
	var inv domain.Invitation
	if err = json.Unmarshal(data, &inv); err != nil {
		return "", nil, fmt.Errorf(`received response is not a valid invitation - %v`, err)
	}

	return
}

func parsePeerDID(did string) (doc domain.DIDDocument, err error) {
	if len(did) < 11 {
		return domain.DIDDocument{}, fmt.Errorf(`invalid did in invitation: %s`, did)
	}

	if did[:11] != `did:peer:1z` {
		return domain.DIDDocument{}, fmt.Errorf(`did type is not peer: %s`, did[:11])
	}

	return
}
