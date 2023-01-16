package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"sync"
)

/* Subscriber Store */

type subKey map[string][]byte // subscriber to public key map

// performance may be improved by using granular locks and
// trading off with complexity and memory utilization
type subStore struct {
	*sync.RWMutex
	subs map[string]subKey // can extend to multiple keys per peer
}

func newSubStore() *subStore {
	return &subStore{RWMutex: &sync.RWMutex{}, subs: map[string]subKey{}}
}

func (s *subStore) queryByTopic(topic string) (subKey, error) {
	s.RLock()
	defer s.RUnlock()
	subs, ok := s.subs[topic]
	if !ok {
		return nil, fmt.Errorf(`topic (%s) is not registered`, topic)
	}

	return subs, nil
}

// add replaces the key if already exists for the subscriber
func (s *subStore) add(topic, sub string, key []byte) {
	s.Lock()
	defer s.Unlock()
	if s.subs[topic] == nil {
		s.subs[topic] = subKey{}
	}
	s.subs[topic][sub] = key
}

func (s *subStore) delete(topic, label string) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs[topic], label)
}

func (s *subStore) deleteTopic(topic string) {
	s.Lock()
	defer s.Unlock()
	delete(s.subs, topic)
}

/* Group Store */

type groupStore struct {
	*sync.RWMutex
	// nested map for group state where primary key is the topic
	// and secondary is the member's label
	states map[string]map[string]models.Member
}

func newGroupStore() *groupStore {
	return &groupStore{
		RWMutex: &sync.RWMutex{},
		states:  map[string]map[string]models.Member{},
	}
}

func (g *groupStore) addMembr(topic string, m models.Member) {
	g.Lock()
	defer g.Unlock()
	if g.states[topic] == nil {
		g.states[topic] = make(map[string]models.Member)
	}
	g.states[topic][m.Label] = m
	//g.log.Trace(`group member added`, m.Label)
}

func (g *groupStore) deleteMembr(topic, membr string) {
	g.Lock()
	defer g.Unlock()
	if g.states[topic] == nil {
		return
	}
	delete(g.states[topic], membr)
}

func (g *groupStore) deleteTopic(topic string) {
	g.Lock()
	defer g.Unlock()
	delete(g.states, topic)
}

// joined checks if current member has already joined a group
// with the given topic
func (g *groupStore) joined(topic string) bool {
	g.RLock()
	defer g.RUnlock()
	if g.states[topic] == nil {
		return false
	}
	return true
}

func (g *groupStore) membrs(topic string) (m []models.Member) {
	g.RLock()
	defer g.RUnlock()
	if g.states[topic] == nil {
		return []models.Member{}
	}

	for _, mem := range g.states[topic] {
		m = append(m, mem)
	}
	return m
}

/* Exchange ID Store */

type exchIdStore struct {
	*sync.RWMutex
	ids map[string]bool
}

func newExchStore() *exchIdStore {
	return &exchIdStore{
		RWMutex: &sync.RWMutex{},
		ids:     map[string]bool{},
	}
}

func (e *exchIdStore) add(id string) {
	e.Lock()
	defer e.Unlock()
	e.ids[id] = true
}

func (e *exchIdStore) delete(id string) {
	e.Lock()
	defer e.Unlock()
	delete(e.ids, id)
}

func (e *exchIdStore) valid(id string) bool {
	e.RLock()
	defer e.RUnlock()
	if _, ok := e.ids[id]; !ok {
		return false
	}
	return true
}
