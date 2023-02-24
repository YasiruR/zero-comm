package stores

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/pubsub/validator"
	"sync"
)

/* Group Store */

type Group struct {
	*sync.RWMutex
	groups map[string]*models.Group // todo changing this (eg: cons level) maliciously is a threat?
}

func NewGroupStore() *Group {
	return &Group{
		RWMutex: &sync.RWMutex{},
		groups:  map[string]*models.Group{},
	}
}

func (g *Group) AddMembr(topic string, m models.Member) {
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

func (g *Group) AddMembrs(topic string, mems ...models.Member) error {
	g.Lock()
	defer g.Unlock()
	for _, m := range mems {
		if g.groups[topic] == nil {
			g.groups[topic] = &models.Group{Members: map[string]models.Member{}}
		}

		if g.groups[topic].Members == nil {
			g.groups[topic].Members = map[string]models.Member{}
		}

		g.groups[topic].Members[m.Label] = m
	}

	var gm []models.Member
	for _, m := range g.groups[topic].Members {
		gm = append(gm, m)
	}

	hash, err := validator.Calculate(gm)
	if err != nil {
		return fmt.Errorf(`calculating group checksum failed - %v`, err)
	}

	g.groups[topic].Checksum = hash
	return nil
}

func (g *Group) DeleteMembr(topic, membr string) {
	g.Lock()
	defer g.Unlock()
	if g.groups[topic] == nil {
		return
	}
	delete(g.groups[topic].Members, membr)
}

func (g *Group) DeleteTopic(topic string) {
	g.Lock()
	defer g.Unlock()
	delete(g.groups, topic)
}

// Joined checks if current member has already Joined a group
// with the given topic
func (g *Group) Joined(topic string) bool {
	g.RLock()
	defer g.RUnlock()
	if g.groups[topic] == nil {
		return false
	}
	return true
}

func (g *Group) Membrs(topic string) (m []models.Member) {
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

func (g *Group) Membr(topic, label string) *models.Member {
	g.RLock()
	defer g.RUnlock()
	if g.groups[topic] == nil {
		return nil
	}

	for _, mem := range g.groups[topic].Members {
		if mem.Label == label {
			return &mem
		}
	}

	return nil
}

func (g *Group) SetGroupParams(topic string, cl domain.ConsistencyLevel, gm domain.GroupMode) error {
	g.Lock()
	defer g.Unlock()

	if cl == `` {
		cl = domain.NoConsistency
	}

	if !cl.Valid() {
		return fmt.Errorf(`invalid consistency level - %s`, cl)
	}

	if !gm.Valid() {
		return fmt.Errorf(`invalid group mode - %s`, gm)
	}

	if g.groups[topic] == nil {
		g.groups[topic] = &models.Group{}
	}

	g.groups[topic].ConsistencyLevel = cl
	g.groups[topic].GroupMode = gm
	return nil
}

func (g *Group) ConsistLevel(topic string) domain.ConsistencyLevel {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].ConsistencyLevel
}

func (g *Group) Mode(topic string) domain.GroupMode {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].GroupMode
}

func (g *Group) Checksum(topic string) string {
	g.RLock()
	g.RUnlock()
	return g.groups[topic].Checksum
}
