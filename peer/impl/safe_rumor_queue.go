package impl

import (
	"sync"
	"time"

	"go.dedis.ch/cs438/types"
)

func NewSafeRumorsQueue(t time.Duration) *SafeRumorsQueue {
	return &SafeRumorsQueue{
		list:    NewMsgList(),
		infoMap: map[string]*RumorInfo{},
		nodeMap: map[string]*MsgNode{},
		timeout: t.Milliseconds(),
	}
}

type SafeRumorsQueue struct {
	sync.Mutex
	list    *MsgList
	nodeMap map[string]*MsgNode
	infoMap map[string]*RumorInfo
	timeout int64
}

type RumorInfo struct {
	dest     string
	sendTime int64
	id       string
	rumor    types.RumorsMessage
	node     *MsgNode
}

func (q *SafeRumorsQueue) Push(msg *types.RumorsMessage, dest string, id string) {
	q.Lock()
	defer q.Unlock()
	n := &MsgNode{
		id: id,
	}
	q.nodeMap[id] = n
	q.list.Push(n)
	q.infoMap[id] = &RumorInfo{
		dest:     dest,
		sendTime: time.Now().UnixMilli(),
		id:       id,
		rumor:    *msg,
		node:     n,
	}
}

func (q *SafeRumorsQueue) GetRumor(n *node) (string, types.RumorsMessage, string, bool) {
	q.Lock()
	defer q.Unlock()
	msgNode, ok := q.list.GetFirst()
	if !ok {
		return "", types.RumorsMessage{}, "", false
	}
	id := msgNode.id
	sendTime := q.infoMap[id].sendTime

	if (time.Now().UnixMilli() - sendTime) >= q.timeout {
		q.list.Poll()
		q.list.Push(msgNode)
		q.infoMap[id].sendTime = time.Now().UnixMilli()
		if !usePlumTree {
			prevNeighbor := q.infoMap[id].dest
			neighbor, ok := n.GetNeighbor(prevNeighbor)
			if !ok {
				return "", types.RumorsMessage{}, "", false
			}
			q.infoMap[id].dest = neighbor
			return neighbor, q.infoMap[id].rumor, id, true
		} else {
			return q.infoMap[id].dest, q.infoMap[id].rumor, id, true
		}

	}
	return "", types.RumorsMessage{}, "", false
}

func (q *SafeRumorsQueue) Remove(id string) {
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
