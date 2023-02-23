package pubsub

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
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
	groups map[string]*models.Group // todo changing this (eg: cons level) maliciously is a threat?
}

func newGroupStore() *groupStore {
	return &groupStore{
		RWMutex: &sync.RWMutex{},
		groups:  map[string]*models.Group{},
	}
}

func (g *groupStore) addMembr(topic string, m models.Member) {
	g.Lock()
	defer g.Unlock()
	if g.groups[topic] == nil {
		g.groups[topic] = &models.Group{Members: map[string]models.Member{}}
	}

	if g.groups[topic].Members == nil {
		g.groups[topic].Members = map[string]models.Member{}
	}

	g.groups[topic].Members[m.Label] = m
}

func (g *groupStore) deleteMembr(topic, membr string) {
	g.Lock()
	defer g.Unlock()
	if g.groups[topic] == nil {
		return
	}
	delete(g.groups[topic].Members, membr)
}

func (g *groupStore) deleteTopic(topic string) {
	g.Lock()
	defer g.Unlock()
	delete(g.groups, topic)
}

// joined checks if current member has already joined a group
// with the given topic
func (g *groupStore) joined(topic string) bool {
	g.RLock()
	defer g.RUnlock()
	if g.groups[topic] == nil {
		return false
	}
	return true
}

func (g *groupStore) membrs(topic string) (m []models.Member) {
	g.RLock()
	defer g.RUnlock()
	if g.groups[topic] == nil {
		return []models.Member{}
	}

	if g.groups[topic].Members == nil {
		return []models.Member{}
	}

	for _, mem := range g.groups[topic].Members {
		m = append(m, mem)
	}

	return m
}

func (g *groupStore) membr(topic, label string) *models.Member {
	g.RLock()
	defer g.RUnlock()
	if g.groups[topic] == nil {
		return nil
	}

	if g.groups[topic].Members == nil {
		return nil
	}

	for _, mem := range g.groups[topic].Members {
		if mem.Label == label {
			return &mem
		}
	}

	return nil
}

func (g *groupStore) setConsistLevel(topic string, cl domain.ConsistencyLevel) error {
	g.Lock()
	defer g.Unlock()

	if !cl.Valid() {
		return fmt.Errorf(`invalid consistency level - %s`, cl)
	}

	g.groups[topic].ConsistencyLevel = cl
	return nil
}

func (g *groupStore) consistLevel(topic string) domain.ConsistencyLevel {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].ConsistencyLevel
}
