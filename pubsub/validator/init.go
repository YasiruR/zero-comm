package validator

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"sort"
)

func Calculate(grp []models.Member) (string, error) {
	byts, err := json.Marshal(order(grp))
	if err != nil {
		return ``, fmt.Errorf(`marshalling group members failed - %v`, err)
	}

	h := sha256.New()
	h.Write(byts)
	val := make([]byte, len(byts))
	base64.StdEncoding.Encode(val, h.Sum(nil))

	return string(bytes.Trim(val, "\x00")), nil
}

// Order can be used to maintain a consistent order in the member list for
// computation of group checksum values.
// NOTE: test the functionality and performance (multiple iterations of a map)
func order(grp []models.Member) (sorted []models.Member) {
	var labels []string
	for _, m := range grp {
		labels = append(labels, m.Label)
	}

	sort.Strings(labels)
	for _, l := range labels {
		for _, m := range grp {
			if m.Label == l {
				sorted = append(sorted, m)
				break
			}
		}
	}

	return sorted
}

// Verify takes a map of group-state hash values indexed by label and returns
// the list of members deviated from the majority. In cases where multiple
// intruder sets exist, the set with least number of deviated members is returned.
func Verify(states map[string]string) (invalidMems []string, ok bool) {
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
		return nil, true
	}

	return invalidMems, false
}

// ValidJoin checks if the initial member set returned by the acceptor is consistent
// across other members thus eliminating intruders in the initial state of the joiner.
// grpHashes is a map with hash values indexed by the member label.
func ValidJoin(accptr string, joinedSet []models.Member, grpHashes map[string]string) error {
	joinedChecksm, err := Calculate(joinedSet)
	if err != nil {
		return fmt.Errorf(`calculating checksum of initial member set failed - %v`, err)
	}
	grpHashes[accptr] = joinedChecksm

	invalidMems, ok := Verify(grpHashes)
	if !ok {
		return fmt.Errorf(`at least one inconsistent member set found (%v)`, invalidMems)
	}

	return nil
}
