package pubsub

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"sync"
)

type validator struct {
	hashes *sync.Map
}

func newValidator() *validator {
	return &validator{
		hashes: &sync.Map{},
	}
}

// updateHash should be called whenever the group state is updated
// with respect to status or role. A hasher is initiated as per each
// update request to be on the safe side with concurrent writes.
func (v *validator) updateHash(topic string, grp []models.Member) error {
	val, err := v.calculate(grp)
	if err != nil {
		return fmt.Errorf(`calculating hash value failed - %v`, err)
	}

	v.hashes.Store(topic, val)
	return nil
}

func (v *validator) calculate(grp []models.Member) (string, error) {
	byts, err := json.Marshal(grp)
	if err != nil {
		return ``, fmt.Errorf(`marshalling group members failed - %v`, err)
	}

	h := sha256.New()
	h.Write(byts)
	return string(h.Sum(nil)), nil
}

func (v *validator) hash(topic string) (string, error) {
	val, ok := v.hashes.Load(topic)
	if !ok {
		return ``, fmt.Errorf(`hash value for the topic %s does not exist`, topic)
	}

	str, ok := val.(string)
	if !ok {
		return ``, fmt.Errorf(`hash value type is incompatible (%v) - should be string`, val)
	}

	return str, nil
}

// verify takes a map of group-state hash values indexed by label and returns
// the list of members deviated from the majority. In cases where multiple
// intruder sets exist, the set with least number of deviated members is returned.
func (v *validator) verify(states map[string]string) (invalidMems []string, ok bool) {
	// inverse map of states where key is hash and value is the list of members
	mems := make(map[string][]string)
	for membr, val := range states {
		if mems[val] == nil {
			mems[val] = []string{}
		}
		mems[val] = append(mems[val], membr)
	}

	// choose the set with the least number of deviated members
	deviated := len(states)
	for _, membrs := range mems {
		if len(membrs) < deviated {
			deviated = len(membrs)
			invalidMems = membrs
		}
	}

	if deviated == len(states) {
		return nil, ok
	}

	return invalidMems, false
}
