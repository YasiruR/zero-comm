package stores

import (
	"fmt"
	"sync"
)

/* Subscriber Store */

// Subscriber store performance may be improved by using granular locks and
// trading off with complexity and memory utilization
type Subscriber struct {
	*sync.RWMutex
	subs map[string]subKey // can extend to multiple keys per peer
}

func NewSubStore() *Subscriber {
	return &Subscriber{RWMutex: &sync.RWMutex{}, subs: map[string]subKey{}}
}

type subKey map[string][]byte // subscriber to public key map

func (s *Subscriber) QueryByTopic(topic string) (subKey, error) {
	s.RLock()
	defer s.RUnlock()
	subs, ok := s.subs[topic]
	if !ok {
		return nil, fmt.Errorf(`topic (%s) is not registered`, topic)
	}

	return subs, nil
}

// Add replaces the key if already exists for the subscriber
func (s *Subscriber) Add(topic, sub string, key []byte) {
	s.Lock()
	defer s.Unlock()
	if s.subs[topic] == nil {
		s.subs[topic] = subKey{}
	}
	s.subs[topic][sub] = key
}

func (s *Subscriber) Delete(topic, label string) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs[topic], label)
}

func (s *Subscriber) DeleteTopic(topic string) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs, topic)
}
