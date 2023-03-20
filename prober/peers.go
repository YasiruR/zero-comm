package prober

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/tryfix/log"
	"sync"
)

type peers struct {
	store *sync.Map
	log   log.Logger
}

func initPeerStore(l log.Logger) *peers {
	return &peers{
		store: &sync.Map{},
		log:   l,
	}
}

// name as the key may not be ideal
func (p *peers) add(label string, pr models.Peer) {
	p.store.Store(label, pr)
}

func (p *peers) peerByLabel(label string) (models.Peer, error) {
	val, ok := p.store.Load(label)
	if !ok {
		return models.Peer{}, fmt.Errorf(`requested peer (%s) does not exist in store`, label)
	}

	pr, ok := val.(models.Peer)
	if !ok {
		return models.Peer{}, fmt.Errorf(`invalid type found for peer %s (%v)`, label, val)
	}

	return pr, nil
}

func (p *peers) peerByExchId(exchId string) (name string, pr models.Peer, exists bool) {
	p.store.Range(func(key, val any) bool {
		tmpPr, ok := val.(models.Peer)
		if !ok {
			p.log.Error(fmt.Sprintf(`invalid type found for peer (%v)`, val))
			return true
		}

		if tmpPr.ExchangeThId == exchId {
			name, ok = key.(string)
			if !ok {
				p.log.Error(fmt.Sprintf(`peer name should be a string (%v)`, key))
				return false
			}

			pr = tmpPr
			exists = true
			return false
		}

		return true
	})

	return
}
