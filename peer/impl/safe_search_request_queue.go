package impl

import (
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
)

func NewSafeSearchRequestQueue() *SafeSearchRequestQueue {
	return &SafeSearchRequestQueue{
		list:    NewMsgList(),
		nodeMap: map[string]*MsgNode{},
		infoMap: map[string]*SearchRequestInfo{},
	}
}

type SafeSearchRequestQueue struct {
	sync.Mutex
	list    *MsgList
	nodeMap map[string]*MsgNode
	infoMap map[string]*SearchRequestInfo
}

type SearchRequestInfo struct {
	sendTime      int64
	id            string
	SearchRequest types.SearchRequestMessage
	node          *MsgNode
	timeout       time.Duration
	retry         uint
	factor        uint
}

func (q *SafeSearchRequestQueue) Push(msg *types.SearchRequestMessage, id string, ring *peer.ExpandingRing) {
	q.Lock()
	defer q.Unlock()
	n := &MsgNode{
		id: id,
	}
	q.nodeMap[id] = n
	q.list.Push(n)
	q.infoMap[id] = &SearchRequestInfo{
		sendTime:      time.Now().UnixMilli(),
		id:            id,
		SearchRequest: *msg,
		node:          n,
		timeout:       ring.Timeout,
		retry:         ring.Retry - 1,
		factor:        ring.Factor,
	}
}

func (q *SafeSearchRequestQueue) GetSearchRequest(n *node) (*types.SearchRequestMessage, bool) {
	q.Lock()
	defer q.Unlock()
	msgNode, ok := q.list.GetFirst()
	if !ok {
		return nil, false
	}
	id := msgNode.id
	info := q.infoMap[id]
	sendTime := info.sendTime
	if (time.Now().UnixMilli() - sendTime) >= info.timeout.Milliseconds() {
		q.list.Poll()
		if info.retry == 0 {
			n.channelTable.Close(id)
			delete(q.nodeMap, id)
			delete(q.infoMap, id)
			return nil, false
		}
		info.retry--
		info.SearchRequest.Budget *= info.factor
		info.sendTime = time.Now().UnixMilli()
		q.list.Push(msgNode)
		return &info.SearchRequest, true
	}
	return nil, false
}

func (q *SafeSearchRequestQueue) Remove(id string) {
	q.Lock()
	defer q.Unlock()
	n, ok := q.nodeMap[id]
	if !ok {
		return
	}
	q.list.Remove(n)
	delete(q.nodeMap, id)
	delete(q.infoMap, id)
}
