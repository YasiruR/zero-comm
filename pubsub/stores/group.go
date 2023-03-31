package stores

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/pubsub/validator"
	"sync"
)

const (
	notifierSleepMs = 200
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

func (g *Group) DeleteMembr(topic, membr string) error {
	g.Lock()
	defer g.Unlock()
	if g.groups[topic] == nil {
		return nil
	}
	delete(g.groups[topic].Members, membr)

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

func (g *Group) SetParams(topic string, gp models.GroupParams) error {
	g.Lock()
	defer g.Unlock()

	if !gp.Mode.Valid() {
		return fmt.Errorf(`invalid group mode - %s`, gp.Mode)
	}

	if g.groups[topic] == nil {
		g.groups[topic] = &models.Group{}
	}

	g.groups[topic].GroupParams = &gp
	return nil
}

func (g *Group) Params(topic string) *models.GroupParams {
	g.RLock()
	defer g.RUnlock()

	if g.groups[topic] == nil {
		return nil
	}

	return g.groups[topic].GroupParams
}

func (g *Group) Mode(topic string) domain.GroupMode {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].Mode
}

func (g *Group) OrderEnabled(topic string) bool {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].OrderEnabled
}

func (g *Group) JoinConsistent(topic string) bool {
	g.RLock()
	defer g.RUnlock()
	return g.groups[topic].JoinConsistent
}

func (g *Group) Checksum(topic string) string {
	g.RLock()
	g.RUnlock()
	return g.groups[topic].Checksum
}
