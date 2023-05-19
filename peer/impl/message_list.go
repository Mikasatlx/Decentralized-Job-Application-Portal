package impl

type MsgList struct {
	head *MsgNode
	tail *MsgNode
}

func NewMsgList() *MsgList {
	head := MsgNode{}
	tail := MsgNode{}
	head.next = &tail
	tail.prev = &head
	return &MsgList{
		head: &head,
		tail: &tail,
	}
}

type MsgNode struct {
	id   string
	prev *MsgNode
	next *MsgNode
}

func (l *MsgList) Push(n *MsgNode) {
	l.tail.prev.next = n
	n.prev = l.tail.prev
	n.next = l.tail
	l.tail.prev = n
}

func (l *MsgList) Poll() (*MsgNode, bool) {
	if l.head.next == l.tail {
		return nil, false
	}
	ret := l.head.next
	l.head.next = ret.next
	ret.next.prev = l.head
	return ret, true
}

func (l *MsgList) Remove(n *MsgNode) {
	prev := n.prev
	next := n.next
	prev.next = next
	next.prev = prev
}

func (l *MsgList) GetFirst() (*MsgNode, bool) {
	if l.head.next == l.tail {
		return nil, false
	}
	return l.head.next, true
}
