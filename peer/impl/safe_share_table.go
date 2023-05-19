package impl

import (
	"math"
	"sync"

	"go.dedis.ch/kyber/v3/share"
)

type SafeShareTable struct {
	sync.RWMutex
	ShareTable map[uint][]byte
	threshold  int
	shares     []*share.PubShare
	num        uint
}

func NewSafeShareTable(num uint) *SafeShareTable {
	return &SafeShareTable{
		ShareTable: map[uint][]byte{},
		threshold:  int(math.Ceil(float64(num)/2) + 1),
		shares:     make([]*share.PubShare, num),
		num:        num,
	}
}

func (t *SafeShareTable) Set(I uint, V []byte) {
	t.Lock()
	defer t.Unlock()
	if len(t.ShareTable) >= t.threshold {
		return
	}
	_, ok := t.ShareTable[I]
	if ok {
		return
	}
	t.ShareTable[I] = V
	if len(t.ShareTable) >= t.threshold {
		for i, v := range t.ShareTable {
			// To check if it would work!!!!!!!!!!!!!!!!!!!!!!!
			vPoint := suite.Point()
			vPoint.UnmarshalBinary(v)
			t.shares[i] = &share.PubShare{
				I: int(i), V: vPoint,
			}
		}
	}
}

func (t *SafeShareTable) Get() ([]*share.PubShare, bool) {
	t.RLock()
	defer t.RUnlock()
	if len(t.ShareTable) >= t.threshold {
		return t.shares, true
	}
	return nil, false
}

func (t *SafeShareTable) Clean() {
	t.Lock()
	defer t.Unlock()
	t.ShareTable = map[uint][]byte{}
	t.shares = make([]*share.PubShare, t.num)
}
