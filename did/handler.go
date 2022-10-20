package did

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
	"strings"
)

func CreateInvitation(invEndpoint, didEndpoint string, pubKey []byte) (url string, err error) {
	did, err := createPeerDID(createDIDDoc(didEndpoint, `message-service`, pubKey))
	if err != nil {
		return ``, fmt.Errorf(`creating peer did failed - %v`, err)
	}

	inv := domain.Invitation{
		Id:       uuid.New().String(), // todo use this as pthid in request
		Type:     "https://didcomm.org/out-of-band/1.0/invitation",
		From:     did,
		Services: createDIDDoc(didEndpoint, `did-communication`, pubKey).Service, // a separate service to reach back for exchange
	}

	bytes, err := json.Marshal(inv)
	if err != nil {
		return ``, fmt.Errorf(`marshalling invitation failed - %v`, err)
	}

	strInv := strings.ReplaceAll(string(bytes), ` `, ``)
	return invEndpoint + `?oob=` + base64.URLEncoding.EncodeToString([]byte(strInv)), nil
}

func createDIDDoc(endpoint, typ string, pubKey []byte) domain.DIDDocument {
	s := domain.Service{
		Id:              uuid.New().String(),
		Type:            typ,
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

func ParseInvitation(encInv string) (endpoint string, pubKey []byte, err error) {
	var bytInv []byte
	if _, err = base64.URLEncoding.Decode(bytInv, []byte(encInv)); err != nil {
		return ``, nil, fmt.Errorf(`base64 url decode failed - %v`, err)
	}

	var inv domain.Invitation
	if err = json.Unmarshal(bytInv, &inv); err != nil {
		return "", nil, fmt.Errorf(`received response is not a valid invitation - %v`, err)
	}

	if len(inv.Services) == 0 {
		return ``, nil, fmt.Errorf(`no service found in invitation [%v]`, inv)
	}

	for _, s := range inv.Services {
		if len(s.RecipientKeys) == 0 {
			continue
		}

		return s.ServiceEndpoint, []byte(s.RecipientKeys[0]), nil
	}

	return ``, nil, fmt.Errorf(`no recipient key found for a service - %v`, inv)
}

func validatePeerDID(did string) error {
	if len(did) < 11 {
		return fmt.Errorf(`invalid did in invitation: %s`, did)
	}

	// should ideally use a regex
	if did[:11] != `did:peer:1z` {
		return fmt.Errorf(`did type is not peer: %s`, did[:11])
	}

	return nil
}
