package invitation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"github.com/google/uuid"
)

type OOBService struct {
	invEndpoint string
}

func NewOOBService(cfg *container.Config) *OOBService {
	return &OOBService{invEndpoint: cfg.Hostname}
}

func (o *OOBService) CreateInv(label, did string, didDoc messages.DIDDocument) (url string, err error) {
	inv := messages.Invitation{
		Id:       uuid.New().String(),
		Type:     messages.OOBInvitationV1,
		From:     did,
		Label:    label,
		Services: didDoc.Service, // a separate service to reach back for exchange
		//Services: createDIDDoc(didEndpoint, `did-communication`, encodedKey).Service, // a separate service to reach back for exchange
	}

	byts, err := json.Marshal(inv)
	if err != nil {
		return ``, fmt.Errorf(`marshalling invitation failed - %v`, err)
	}

	return o.invEndpoint + `?oob=` + base64.URLEncoding.EncodeToString(byts), nil
}

func (o *OOBService) ParseInv(encInv string) (inv messages.Invitation, endpoint string, pubKey []byte, err error) {
	bytInv := make([]byte, len(encInv))
	if _, err = base64.URLEncoding.Decode(bytInv, []byte(encInv)); err != nil {
		return messages.Invitation{}, ``, nil, fmt.Errorf(`base64 url decode failed - %v`, err)
	}
	// removes redundant elements from the allocated byte slice
	bytInv = bytes.Trim(bytInv, "\x00")

	if err = json.Unmarshal(bytInv, &inv); err != nil {
		return messages.Invitation{}, "", nil, fmt.Errorf(`received response is not a valid invitation - %v`, err)
	}

	if len(inv.Services) == 0 {
		return messages.Invitation{}, ``, nil, fmt.Errorf(`no service found in invitation [%v]`, inv)
	}

	for _, s := range inv.Services {
		if len(s.RecipientKeys) == 0 {
			continue
		}

		pubKey, err = base64.StdEncoding.DecodeString(s.RecipientKeys[0])
		if err != nil {
			return messages.Invitation{}, ``, nil, fmt.Errorf(`decoding recipient key failed - %v`, err)
		}
		return inv, s.ServiceEndpoint, pubKey, nil
	}

	return messages.Invitation{}, ``, nil, fmt.Errorf(`no recipient key found for a service - %v`, inv)
}
