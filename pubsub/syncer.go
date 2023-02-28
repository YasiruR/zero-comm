package pubsub

import (
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/pubsub/stores"
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
	SenderId int   `json:"send_id"`
	LampTs   int64 `json:"lamp_ts"`
}

type groupMsg struct {
	Ts   ts     `json:"ts"`
	Data []byte `json:"data"`
}

// syncer is an additional layer enforced on the didcomm-message content
// to maintain causal order among messages. Only data messages published
// to the group will be impacted by syncer and can be enabled for each topic.
type syncer struct {
	id    int
	tsMap map[string]*ts
	gs    *stores.Group
	*sync.Mutex
}

func newSyncer(gs *stores.Group) *syncer {
	s := rand.NewSource(time.Now().UnixNano())
	return &syncer{
		id:    rand.New(s).Intn(maxId),
		tsMap: map[string]*ts{},
		gs:    gs,
		Mutex: &sync.Mutex{},
	}
}

func (s *syncer) init(topic string) {
	s.Lock()
	defer s.Unlock()
	s.tsMap[topic] = &ts{
		SenderId: s.id,
		LampTs:   time.Now().Unix(),
	}
}

// parse returns the didcomm-message content while updating the lamport timestamp
func (s *syncer) parse(topic, msg string) (data string, err error) {
	s.Lock()
	defer s.Unlock()
	storedTs, ok := s.tsMap[topic]
	if !ok {
		return msg, nil
	}

	var gm groupMsg
	if err = json.Unmarshal([]byte(msg), &gm); err != nil {
		return ``, fmt.Errorf(`unmarshalling message failed - %v`, err)
	}

	if storedTs.LampTs <= gm.Ts.LampTs {
		s.tsMap[topic] = &gm.Ts
	}

	return string(gm.Data), nil
}

// message builds the didcomm-message with updated lamport timestamp only if syncing enabled
// for the topic. If otherwise, didcomm-message is returned without any modification
func (s *syncer) message(topic string, data []byte) (msg []byte, err error) {
	s.Lock()
	defer s.Unlock()
	storedTs, ok := s.tsMap[topic]
	if !ok {
		return data, nil
	}

	curntTime := time.Now().Unix()
	// setting the updated lamport timestamp by selecting between current_timestamp and stored_timestamp+1
	if curntTime < storedTs.LampTs {
		storedTs.LampTs = storedTs.LampTs + 1
	} else {
		storedTs.LampTs = curntTime
	}

	return json.Marshal(groupMsg{Ts: *storedTs, Data: data})
}
