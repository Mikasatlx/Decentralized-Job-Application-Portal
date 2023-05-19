package impl

import (
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
)

func NewSafeDataRequestQueue() *SafeDataRequestQueue {
	return &SafeDataRequestQueue{
		list:    NewMsgList(),
		nodeMap: map[string]*MsgNode{},
		infoMap: map[string]*DataRequestInfo{},
	}
}

type SafeDataRequestQueue struct {
	sync.Mutex
	list    *MsgList
	nodeMap map[string]*MsgNode
	infoMap map[string]*DataRequestInfo
}

type DataRequestInfo struct {
	dest        string
	sendTime    int64
	id          string
	dataRequest types.DataRequestMessage
	node        *MsgNode
	timeout     time.Duration
	retry       uint
	factor      uint
}

func (q *SafeDataRequestQueue) Push(msg *types.DataRequestMessage, dest string, id string, backoff *peer.Backoff) {
	q.Lock()
	defer q.Unlock()
	n := &MsgNode{
		id: id,
	}
	q.nodeMap[id] = n
	q.list.Push(n)
	q.infoMap[id] = &DataRequestInfo{
		dest:     dest,
		sendTime: time.Now().UnixMilli(),
		id:       id,
		node:     n,
		timeout:  backoff.Initial,
		retry:    backoff.Retry - 1,
		factor:   backoff.Factor,
	}
}

func (q *SafeDataRequestQueue) GetDataRequest(n *node) (*types.DataRequestMessage, string, bool) {
	q.Lock()
	defer q.Unlock()
	msgNode, ok := q.list.GetFirst()
	if !ok {
		return nil, "", false
	}
	id := msgNode.id
	info := q.infoMap[id]
	sendTime := info.sendTime
	if (time.Now().UnixMilli() - sendTime) >= info.timeout.Milliseconds() {
		q.list.Poll()
		if info.retry == 0 {
			n.channelTable.Close(id)
		}
		info.retry--
		info.timeout *= time.Duration(info.factor)
		info.sendTime = time.Now().UnixMilli()
		q.list.Push(msgNode)
		return &info.dataRequest, info.dest, true

	}
	return nil, "", false
}

func (q *SafeDataRequestQueue) Remove(id string) {
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
