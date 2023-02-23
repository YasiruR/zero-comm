package pubsub

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	maxId = 10000000
)

type ts struct {
	// in case lamport timestamps are equal for multiple messages
	// reader can sort them corresponding to the id and hence the
	// order will be consistent across the group
	Id       int    `json:"id"`
	LampTs   int64  `json:"lamp_ts"`
	Checksum string `json:"checksum"`
}

type groupMsg struct {
	Ts   ts     `json:"ts"`
	Data []byte `json:"data"`
}

// syncer is an additional layer enforced on the didcomm-message content
// to maintain causal order among messages. Only data messages published
// to the group will be impacted by syncer and can be enabled by command
// line arguments of the agent.
type syncer struct {
	id int
	// todo topic map
	lampTs int64
	*sync.RWMutex
}

func newSyncer(enabled bool) *syncer {
	if !enabled {
		return nil
	}

	s := rand.NewSource(time.Now().UnixNano())
	return &syncer{
		id:      rand.New(s).Intn(maxId),
		lampTs:  time.Now().Unix(),
		RWMutex: &sync.RWMutex{},
	}
}

// parse returns the didcomm-message content while updating the lamport timestamp
func (s *syncer) parse(msg string) (data string, err error) {
	var gm groupMsg
	if err = json.Unmarshal([]byte(msg), &gm); err != nil {
		return ``, fmt.Errorf(`unmarshalling message failed - %v`, err)
	}

	s.Lock()
	defer s.Unlock()
	if s.lampTs < gm.Ts.LampTs {
		s.lampTs = gm.Ts.LampTs
	}

	return string(gm.Data), nil
}

// message builds the didcomm-message with updated lamport timestamp
func (s *syncer) message(data []byte) (msg []byte, err error) {
	lampTs := time.Now().Unix()
	s.RLock()
	if lampTs < s.lampTs {
		lampTs = s.lampTs + 1
	}
	s.RUnlock()

	// todo if checksum is not equal to last received, then wait (ot let notify) until correct, before returning here
	// should only be done if strong consistency is required

	return json.Marshal(groupMsg{Ts: ts{Id: s.id, LampTs: lampTs}, Data: data})
}
