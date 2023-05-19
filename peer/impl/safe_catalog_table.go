package impl

import (
	"errors"
	"math/rand"
	"sync"

	"go.dedis.ch/cs438/peer"
)

// It is a thread safe routing table

type SafeCatalog struct {
	sync.RWMutex
	Catalog peer.Catalog
}

func NewSafeCatalog() *SafeCatalog {
	return &SafeCatalog{
		Catalog: peer.Catalog{},
	}
}

func (t *SafeCatalog) Set(key string, peer string) {
	t.Lock()
	defer t.Unlock()
	_, ok := t.Catalog[key]
	if !ok {
		t.Catalog[key] = map[string]struct{}{
			peer: {},
		}
	} else {
		t.Catalog[key][peer] = struct{}{}
	}
}

func (t *SafeCatalog) GetAll() peer.Catalog {
	t.RLock()
	defer t.RUnlock()
	copyCatalog := peer.Catalog{}
	for key, peers := range t.Catalog {
		copyCatalog[key] = map[string]struct{}{}
		for peer := range peers {
			copyCatalog[key][peer] = struct{}{}
		}
	}
	return copyCatalog
}

func (t *SafeCatalog) GetRandPeer(hash string) (string, error) {
	t.RLock()
	defer t.RUnlock()
	match, ok := t.Catalog[hash]
	if !ok {
		return "", errors.New("could not find random pair to request data")
	}
	matchList := make([]string, 0, len(match))
	for k := range match {
		matchList = append(matchList, k)
	}
	return matchList[rand.Intn(len(match))], nil
}
