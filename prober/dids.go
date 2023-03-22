package prober

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/messages"
	"sync"
)

type didStore struct {
	didMap    map[string]string
	didDocMap map[string]messages.DIDDocument
	*sync.RWMutex
}

func initDIDStore() *didStore {
	return &didStore{
		didMap:    map[string]string{},
		didDocMap: map[string]messages.DIDDocument{},
		RWMutex:   &sync.RWMutex{},
	}
}

func (d *didStore) add(label string, did string, doc messages.DIDDocument) {
	d.Lock()
	defer d.Unlock()
	d.didMap[label] = did
	d.didDocMap[label] = doc
}

func (d *didStore) get(label string) (did string, doc messages.DIDDocument, err error) {
	d.RLock()
	defer d.RUnlock()
	did, ok := d.didMap[label]
	if !ok {
		return ``, messages.DIDDocument{}, fmt.Errorf(`did does not exist for %s`, label)
	}

	didDoc, ok := d.didDocMap[label]
	if !ok {
		return ``, messages.DIDDocument{}, fmt.Errorf(`did-doc does not exist for %s`, label)
	}

	return did, didDoc, nil
}
