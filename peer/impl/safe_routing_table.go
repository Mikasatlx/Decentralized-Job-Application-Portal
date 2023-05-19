package impl

import (
	"sync"

	"go.dedis.ch/cs438/peer"
)

// It is a thread safe routing table

type SafeRoutingTable struct {
	sync.RWMutex
	routingTable peer.RoutingTable
}

func NewSafeRoutingTable(origin string) *SafeRoutingTable {
	return &SafeRoutingTable{
		routingTable: peer.RoutingTable{origin: origin},
	}
}

func (t *SafeRoutingTable) Set(origin string, relayAddr string) {
	t.Lock()
	defer t.Unlock()
	t.routingTable[origin] = relayAddr
}

func (t *SafeRoutingTable) Delete(origin string) {
	t.Lock()
	defer t.Unlock()
	delete(t.routingTable, origin)
}

func (t *SafeRoutingTable) GetAll() peer.RoutingTable {
	t.RLock()
	defer t.RUnlock()
	copyTable := peer.RoutingTable{}
	for key, val := range t.routingTable {
		copyTable[key] = val
	}
	return copyTable
}

func (t *SafeRoutingTable) Contains(origin string) bool {
	t.RLock()
	defer t.RUnlock()
	_, ok := t.routingTable[origin]
	return ok
}

func (t *SafeRoutingTable) Get(origin string) (string, bool) {
	t.RLock()
	defer t.RUnlock()
	val, ok := t.routingTable[origin]
	return val, ok
}
