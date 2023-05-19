package impl

import (
	"sync"

	"go.dedis.ch/cs438/types"
)

type SafeChannelTable struct {
	sync.RWMutex
	channelTable map[string]chan types.Message
}

func NewSafeChannelTable() *SafeChannelTable {
	return &SafeChannelTable{
		channelTable: map[string]chan types.Message{},
	}
}

func (t *SafeChannelTable) Set(requestID string) <-chan types.Message {
	t.Lock()
	defer t.Unlock()
	c := make(chan types.Message, chanSize)
	t.channelTable[requestID] = c
	return c
}

func (t *SafeChannelTable) Delete(requestID string) {
	t.Lock()
	defer t.Unlock()
	delete(t.channelTable, requestID)
}

func (t *SafeChannelTable) Close(requestID string) {
	t.Lock()
	defer t.Unlock()
	if t.channelTable[requestID] != nil {
		close(t.channelTable[requestID])
	}
	delete(t.channelTable, requestID)
}

func (t *SafeChannelTable) FeedMsg(requestID string, msg types.Message) {
	t.RLock()
	defer t.RUnlock()
	c, ok := t.channelTable[requestID]
	if !ok {
		return
	}
	c <- msg
}
