package impl

import (
	"sync"

	"go.dedis.ch/kyber/v3"
)

type SafePubKeyTable struct {
	sync.RWMutex
	num      int
	keys     []kyber.Point
	keyTable map[uint]kyber.Point
}

func NewSafePubKeyTable(num int) *SafePubKeyTable {
	return &SafePubKeyTable{
		num:      num,
		keys:     make([]kyber.Point, num),
		keyTable: map[uint]kyber.Point{},
	}
}

func (t *SafePubKeyTable) Set(index uint, key kyber.Point) {
	t.Lock()
	defer t.Unlock()
	_, ok := t.keyTable[index]
	if ok {
		return
	}
	t.keyTable[index] = key
	if len(t.keyTable) == t.num {
		for i, key := range t.keyTable {
			t.keys[i] = key
		}
	}
}

func (t *SafePubKeyTable) GetKeys() ([]kyber.Point, bool) {
	t.RLock()
	defer t.RUnlock()
	if len(t.keyTable) == t.num {
		return t.keys, true
	}
	return nil, false
}
