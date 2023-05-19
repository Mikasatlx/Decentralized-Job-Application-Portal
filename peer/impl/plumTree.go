package impl

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func NewPlumTree(neighbors []string) *plumTree {
	return &plumTree{
		peers:         newPeerList(neighbors),
		msgMap:        make(msgStatusMap),
		graftInterval: 300 * time.Millisecond,
		iHaveInterval: 5000 * time.Millisecond,
	}
}

type plumTreeTable struct {
	table     map[string]*plumTree
	neighbors []string
	queue     ResendQueue
	sync.Mutex
}

func NewPlumTreeMap() plumTreeTable {
	return plumTreeTable{
		table: make(map[string]*plumTree),
		queue: ResendQueue{
			queue:         make(chan resendTask, resendQueueSize),
			revMap:        make(map[string]struct{}),
			lazyQueueStop: make(chan struct{}),
			lazyWaitGroup: &sync.WaitGroup{},
		},
	}
}

func (p *plumTreeTable) addNeighbor(addr string) {
	p.Lock()
	defer p.Unlock()
	p.neighbors = append(p.neighbors, addr)

}

func (p *plumTreeTable) getPlumTreeInstance(addr string) (*plumTree, bool) {
	p.Lock()
	defer p.Unlock()
	_, ext := p.table[addr]
	if !ext {
		p.table[addr] = NewPlumTree(p.neighbors) // initialize
	}
	return p.table[addr], ext
}

type resendTask struct {
	lastTime time.Duration
	interval time.Duration
	transMsg transport.Message
	origin   string
	list     []string
}

type ResendQueue struct {
	queue  chan resendTask
	revMap map[string]struct{}
	sync.Mutex
	lazyQueueStop chan struct{}
	lazyWaitGroup *sync.WaitGroup
}

func (n *node) LazyQueueSender() {
	defer n.plumTreeTable.queue.lazyWaitGroup.Done()
	for {
		<-time.After(time.Millisecond * 2)

		select {
		case <-n.plumTreeTable.queue.lazyQueueStop:
			return
		case task := <-n.plumTreeTable.queue.queue:

			if time.Duration(time.Now().UnixMilli())-task.lastTime > task.interval {

				for _, peer := range task.list {
					err := n.SendMsg(peer, peer, task.transMsg)
					if err != nil {
						n.SendTimeout()
					}
				}
			} else {
				n.plumTreeTable.queue.queue <- task
			}

		}
	}
}

func (n *node) sendIHaveMsg(origin string, transMsg transport.Message, interval time.Duration) {
	plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(origin)
	plumTreeBroadcaster.Lock()
	lazyList := plumTreeBroadcaster.getLazyList()
	plumTreeBroadcaster.Unlock()
	n.plumTreeTable.queue.queue <- resendTask{
		lastTime: time.Duration(time.Now().UnixMilli()),
		interval: time.Duration(interval.Milliseconds()),
		transMsg: transMsg,
		origin:   origin,
		list:     lazyList,
	}

}

func (n *node) sendGraftMsg(msgHash string, status *msgStatus, plumTreeBroadcaster *plumTree) {
	index := 0
	graftMsg := plumTreeBroadcaster.prepareGraftMsg(msgHash)
	transportGraftMsg, _ := n.MessageRegistry.MarshalMessage(graftMsg)
	for {
		select {
		case <-time.After(plumTreeBroadcaster.graftInterval):
			plumTreeBroadcaster.Lock()
			lazyList := status.lazyPeers
			index = index % len(lazyList)
			peer := lazyList[index]
			index += 1
			plumTreeBroadcaster.moveToEager(peer)
			plumTreeBroadcaster.Unlock()

			err := n.SendMsg(peer, peer, transportGraftMsg)

			// log.Info().Str("node", n.myAddr).Msg("Send graft msg for " + msgHash)
			if err != nil {
				n.SendTimeout()
			}
		case <-status.received:
			return
		}
	}
}

type plumTree struct {
	sync.Mutex
	peers         peerList
	msgMap        msgStatusMap
	graftInterval time.Duration
	iHaveInterval time.Duration
}

// type sendTask struct{
// 	msgType string
// 	sendTime uint64

// }

// return msgStatus of a specific hash and whether it exists
// if exists, return status & true
// if not, return a new status with two attributes false & false
func (t *plumTree) computeIfAbsent(msgHash string) (*msgStatus, bool) {
	_, ext := t.msgMap[msgHash]
	if !ext {
		t.msgMap[msgHash] = &msgStatus{
			receivedMsg:  false,
			requestGraft: false,
			received:     make(chan bool),
			lazyPeers:    make([]string, 0),
		}
	}
	return t.msgMap[msgHash], ext
}

func (t *plumTree) prepareGraftMsg(msgHash string) types.GraftMessage {
	graftMsg := types.GraftMessage{
		MsgHash: msgHash,
	}
	return graftMsg
}

func (t *plumTree) prepareIHaveMsg(msgHash string) types.IHaveMessage {
	iHaveMsg := types.IHaveMessage{
		MsgHash: msgHash,
	}
	return iHaveMsg
}

func (t *plumTree) preparePruneMsg(msgHash string) types.PruneMessage {
	pruneMsg := types.PruneMessage{
		MsgHash: msgHash,
	}
	return pruneMsg
}

type msgStatus struct {
	receivedMsg  bool
	requestGraft bool
	received     chan bool
	lazyPeers    []string
}

// MsgHash format: "ip:port;seq" 192.0.0.1:1;1 A Rumor with seq number 1 from 192.0.0.1:1
type msgStatusMap map[string]*msgStatus

type peerList struct {
	lazys  map[string]struct{}
	eagers map[string]struct{}
}

func newPeerList(neighbors []string) peerList {
	res := peerList{
		lazys:  make(map[string]struct{}),
		eagers: make(map[string]struct{}),
	}
	for _, peer := range neighbors {
		res.eagers[peer] = struct{}{}
	}
	return res
}

func (p *plumTree) addPeer(peer string) {
	p.peers.eagers[peer] = struct{}{}
}

func (p *plumTree) addLazyPeer(peer string) {
	p.peers.lazys[peer] = struct{}{}
}

func (p *plumTree) removePeer(peer string) {
	delete(p.peers.eagers, peer)
	delete(p.peers.lazys, peer)
}

func (p *plumTree) moveToLazy(peer string) {
	delete(p.peers.eagers, peer)
	p.peers.lazys[peer] = struct{}{}
}

func (p *plumTree) moveToEager(peer string) {
	delete(p.peers.lazys, peer)
	p.peers.eagers[peer] = struct{}{}
}

func (p *plumTree) considerNewEagerPeer(peer string) {
	_, ext := p.peers.lazys[peer]
	if !ext {
		p.peers.eagers[peer] = struct{}{}
	}
}

func (p *plumTree) considerNewLazyPeer(peer string) {
	_, ext := p.peers.eagers[peer]
	if !ext {
		p.peers.lazys[peer] = struct{}{}
	}
}

func (p *plumTree) getEagerList() []string {
	res := make([]string, 0)
	for k, _ := range p.peers.eagers {
		res = append(res, k)
	}
	return res
}

func (p *plumTree) getLazyList() []string {
	res := make([]string, 0)
	for k, _ := range p.peers.lazys {
		res = append(res, k)
	}
	return res
}

func (p *plumTree) showEagerList(node string) {
	log.Info().Str("node", node).Msg("Eager list")

	for k, _ := range p.peers.eagers {
		log.Info().Msg(k)
	}
}

func (p *plumTree) showLazyList(node string) {
	log.Info().Str("node", node).Msg("Lazy list")

	for k, _ := range p.peers.lazys {
		log.Info().Msg(k)
	}
}
