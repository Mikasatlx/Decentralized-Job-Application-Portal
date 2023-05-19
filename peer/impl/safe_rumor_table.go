package impl

import (
	"math"
	"sync"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

type SafeRumorTable struct {
	sync.RWMutex
	rumorTable map[string][]types.Rumor
	curSeq     map[string]uint // seq that has been processed from origin so far
	revSeq     map[string]uint // max seq that has been recv from origin so far
}

func NewSafeRumorTable() *SafeRumorTable {
	return &SafeRumorTable{
		rumorTable: map[string][]types.Rumor{},
		curSeq:     make(map[string]uint),
		revSeq:     make(map[string]uint),
	}
}

func (t *SafeRumorTable) TryToAcceptOrigin(rumor types.Rumor) string {
	t.Lock()
	defer t.Unlock()
	origin := rumor.Origin
	seq := rumor.Sequence
	if uint(len(t.rumorTable[origin]))+1 == seq {
		t.rumorTable[origin] = append(t.rumorTable[origin], rumor)
		return "process"
	} else if uint(len(t.rumorTable[origin]))+1 > seq {
		return "prune"
	} else {
		return "future"
	}
}

// try to accept the rumor, if succeeding, return true, else return false
func (t *SafeRumorTable) TryToAccept(rumor types.Rumor) []types.Rumor {
	t.Lock()
	defer t.Unlock()
	origin := rumor.Origin
	seq := rumor.Sequence
	_, exist := t.rumorTable[origin]
	if !exist {
		t.rumorTable[origin] = make([]types.Rumor, rumorTableSize)
		t.curSeq[origin] = 0
		t.revSeq[origin] = 0
	}

	t.rumorTable[origin][seq-1] = rumor
	t.revSeq[origin] = uint(math.Max(float64(seq), float64(t.revSeq[origin])))

	if t.curSeq[origin]+1 == seq {

		for pos := t.curSeq[origin]; pos <= t.revSeq[origin]; pos++ {
			if t.rumorTable[origin][pos] == (types.Rumor{}) {
				res := make([]types.Rumor, pos-t.curSeq[origin])
				copy(res, t.rumorTable[origin][t.curSeq[origin]:pos])
				t.curSeq[origin] = pos
				return res
			}
		}
	}
	return []types.Rumor{}
}

func (t *SafeRumorTable) GetStatusMsg() *types.StatusMessage {
	t.RLock()
	defer t.RUnlock()
	statusMsg := types.StatusMessage{}
	for key, val := range t.rumorTable {
		statusMsg[key] = uint(len(val))
	}
	return &statusMsg
}

func (t *SafeRumorTable) GetStatusMsgNew() *types.StatusMessage {
	t.RLock()
	defer t.RUnlock()
	statusMsg := types.StatusMessage{}
	for key, _ := range t.rumorTable {
		statusMsg[key] = t.curSeq[key]
	}
	return &statusMsg
}

func (t *SafeRumorTable) GetInterval(origin string, from uint, to uint) []types.Rumor {
	t.RLock()
	defer t.RUnlock()
	res := make([]types.Rumor, to-from)
	copy(res, t.rumorTable[origin][from:to])
	return res
}

func (t *SafeRumorTable) GetIntervalNew(origin string, from uint, to uint) []types.Rumor {
	t.RLock()
	defer t.RUnlock()
	res := make([]types.Rumor, 0)
	for pos := from; pos < to; pos++ {
		if t.rumorTable[origin][pos] != (types.Rumor{}) {
			res = append(res, t.rumorTable[origin][pos])
		} else {
			return res
		}
	}
	return res
}

func (t *SafeRumorTable) NewRumorsMessage(origin string, msg transport.Message) types.RumorsMessage {
	t.Lock()
	defer t.Unlock()
	rumor := types.Rumor{
		Origin:   origin,
		Sequence: uint(len(t.rumorTable[origin])) + 1,
		Msg:      &msg,
	}
	t.rumorTable[origin] = append(t.rumorTable[origin], rumor)
	rumorMsg := types.RumorsMessage{
		Rumors: []types.Rumor{rumor},
	}
	return rumorMsg
}

func (t *SafeRumorTable) NewRumorsMessageNew(origin string, msg transport.Message) types.RumorsMessage {
	t.Lock()
	defer t.Unlock()
	_, exist := t.rumorTable[origin]
	if !exist {
		t.rumorTable[origin] = make([]types.Rumor, rumorTableSize)
		t.curSeq[origin] = 0
		t.revSeq[origin] = 0
	}
	rumor := types.Rumor{
		Origin:   origin,
		Sequence: t.curSeq[origin] + 1,
		Msg:      &msg,
	}
	t.rumorTable[origin][t.curSeq[origin]] = rumor
	t.curSeq[origin] += 1
	t.revSeq[origin] += 1
	rumorMsg := types.RumorsMessage{
		Rumors: []types.Rumor{rumor},
	}
	return rumorMsg
}
