package impl

import (
	"sync"
)

type SafeIndexTable struct {
	sync.RWMutex
	indexTable map[uint]uint
}

func NewSafeIndexTable() *SafeIndexTable {
	return &SafeIndexTable{
		indexTable: map[uint]uint{},
	}
}

func (t *SafeIndexTable) Set(index uint) bool {
	t.Lock()
	defer t.Unlock()
	_, ok := t.indexTable[index]
	if ok {
		return false
	}
	t.indexTable[index] = 0
	return true
}

func (t *SafeIndexTable) Size() int {
	t.RLock()
	defer t.RUnlock()
	return len(t.indexTable)
}
