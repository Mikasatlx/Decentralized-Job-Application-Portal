package impl

import (
	"bytes"
	"encoding/hex"
	"errors"
	//"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// NewPeer creates a new peer. You can change the content and location of this
// function but you MUST NOT change its signature and package location.
func NewPeer(conf peer.Configuration) peer.Peer {
	// here you must return a struct that implements the peer.Peer functions.
	// Therefore, you are free to rename and change it as you want.

	// for project pbft
	myAddr := conf.Socket.GetAddress()
	getRsaKeys(myAddr)
	n := node{
		Configuration:       &conf,
		myAddr:              myAddr,
		routingTable:        NewSafeRoutingTable(myAddr),
		recvStop:            make(chan struct{}),
		recvWaitGroup:       &sync.WaitGroup{},
		rumorTable:          NewSafeRumorTable(),
		rumorQueue:          NewSafeRumorsQueue(conf.AckTimeout),
		sendStop:            make(chan struct{}),
		sendWaitGroup:       &sync.WaitGroup{},
		regStop:             make(chan struct{}),
		regWaitGroup:        &sync.WaitGroup{},
		procWaitGroup:       &sync.WaitGroup{},
		procQueue:           make(chan transport.Packet, procQueueSize),
		AntiEntropyInterval: conf.AntiEntropyInterval,
		HeartbeatInterval:   conf.HeartbeatInterval,
		ContinueMongering:   conf.ContinueMongering,
		AckTimeout:          conf.AckTimeout,
		catalog:             NewSafeCatalog(),
		channelTable:        NewSafeChannelTable(),
		dataRequestQueue:    NewSafeDataRequestQueue(),
		searchRequestQueue:  NewSafeSearchRequestQueue(),
		paxos:               NewSafePaxos(),
		pbft:                NewSafePbft(myAddr, conf.PaxosID),
		PaxosID:             conf.PaxosID,
		PaxosThreshold:      conf.PaxosThreshold,
		TotalPeers:          conf.TotalPeers,
		PaxosProposerRetry:  conf.PaxosProposerRetry,
		SendQueue:           make(chan transport.Packet, chanSize),
		plumTreeTable:       NewPlumTreeMap(),
		procStop:            make(chan struct{}),
	}
	conf.MessageRegistry.RegisterMessageCallback(types.ChatMessage{}, n.ExecChatMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.RumorsMessage{}, n.ExecRumorMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.AckMessage{}, n.ExecAckMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.StatusMessage{}, n.ExecStatusMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.EmptyMessage{}, n.ExecEmptyMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PrivateMessage{}, n.ExecPrivateMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.DataReplyMessage{}, n.ExecDataReplyMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.DataRequestMessage{}, n.ExecDataRequestMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.SearchRequestMessage{}, n.ExecSearchRequestMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.SearchReplyMessage{}, n.ExecSearchReplyMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PaxosPrepareMessage{}, n.ExecPaxosPrepareMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PaxosPromiseMessage{}, n.ExecPaxosPromiseMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PaxosProposeMessage{}, n.ExecPaxosProposeMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PaxosAcceptMessage{}, n.ExecPaxosAcceptMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.TLCMessage{}, n.ExecTLCMessage)

	// project
	conf.MessageRegistry.RegisterMessageCallback(types.PrePrepareMessage{}, n.ExecPrePrepareMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PrepareMessage{}, n.ExecPrepareMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.CommitMessage{}, n.ExecCommitMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.CheckpointMessage{}, n.ExecCheckpointMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PbftTLCMessage{}, n.ExecPbftTLCMessage)

	conf.MessageRegistry.RegisterMessageCallback(types.PubKeyMessage{}, n.ExecPubKeyMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.DealMessage{}, n.ExecDealMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.DealResponseMessage{}, n.ExecDealResponseMessage)

	conf.MessageRegistry.RegisterMessageCallback(types.PKRequestMessage{}, n.ExecPKRequestMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PKResponseMessage{}, n.ExecPKResponseMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.EncryptedFileMessage{}, n.ExecEncryptedFileMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.EncryptedChunkMessage{}, n.ExecEncryptedChunkMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.RequestFileMessage{}, n.ExecRequestFileMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.ResponseFileMessage{}, n.ExecResponseFileMessage)

	conf.MessageRegistry.RegisterMessageCallback(types.IHaveMessage{}, n.ExecIHaveMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.GraftMessage{}, n.ExecGraftMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.PruneMessage{}, n.ExecPruneMessage)
	conf.MessageRegistry.RegisterMessageCallback(types.AckIHaveMessage{}, n.ExecAckIHaveMessage)

	return &n
}

// node implements a peer to build a Peerster system
//
// - implements peer.Peer
type node struct {
	// You probably want to keep the peer.Configuration on this struct:
	peer.Peer
	*peer.Configuration
	myAddr              string
	routingTable        *SafeRoutingTable
	recvStop            chan struct{}
	recvWaitGroup       *sync.WaitGroup
	procWaitGroup       *sync.WaitGroup
	rumorTable          *SafeRumorTable
	rumorQueue          *SafeRumorsQueue
	sendStop            chan struct{}
	procStop            chan struct{}
	sendWaitGroup       *sync.WaitGroup
	regStop             chan struct{}
	regWaitGroup        *sync.WaitGroup
	AntiEntropyInterval time.Duration
	HeartbeatInterval   time.Duration
	ContinueMongering   float64
	AckTimeout          time.Duration
	catalog             *SafeCatalog
	channelTable        *SafeChannelTable
	dataRequestQueue    *SafeDataRequestQueue
	searchRequestQueue  *SafeSearchRequestQueue
	paxos               *SafePaxos
	pbft                *SafePbft
	PaxosID             uint
	PaxosThreshold      func(uint) int
	TotalPeers          uint
	PaxosProposerRetry  time.Duration
	SendQueue           chan transport.Packet
	plumTreeTable       plumTreeTable
	procQueue           chan transport.Packet
	// plumTreeBroadcaster *plumTree

	// Project
	cn *CothorityNode
	a  *Applicant
	h  *HR
}

// project---------------start

// A possible bug is that the nodes that join later may not finish initializing,
// so the message handel functions may nil pointers
func (n *node) CreateCothorityNode(index uint, addresses []string, hrSecretTable map[uint]string) {
	n.cn = NewCothorityNode(n, index, addresses, hrSecretTable)
}

func (n *node) StartCothorityNode() {
	n.cn.BroadcastPubKey()
}

func (n *node) CreateHR(id uint, secret string, num uint) {
	n.h = NewHR(n, id, secret, num)
}

func (n *node) CreateApplicant() {
	n.a = NewApplicant(n)
}

// to get the content of blockchain
func (n *node) GetBlockchain() []types.PaxosValue {
	if pbft {
		// the following is for visualization of pbft based implementation, implemented elsewhere
		return nil
	} else {
		// the following is for visualization of paxos based implementation
		chainMap := n.Storage.GetBlockchainStore().GetAll()
		cur, ok := chainMap["LastBlockKey"]
		if !ok {
			return nil
		}
		retList := make([]types.PaxosValue, len(chainMap)-1)
		delete(chainMap, "LastBlockKey")
		for len(chainMap) > 0 {
			key := hex.EncodeToString(cur)
			val := chainMap[key]
			block := types.BlockchainBlock{}
			block.Unmarshal(val)
			retList[block.Index] = block.Value
			cur = block.PrevHash
			delete(chainMap, key)
		}
		return retList
	}
}

// project---------------end

// Start implements peer.Service
func (n *node) Start() error {
	//panic("to be implemented in HW0")

	// process messages
	n.procWaitGroup.Add(ProcNum)
	for i := 0; i < ProcNum; i++ {
		go n.Processor()
	}
	// receive messages
	n.recvWaitGroup.Add(RecvNum)
	for i := 0; i < RecvNum; i++ {
		go n.Receiver()
	}
	// resend I have msg
	n.plumTreeTable.queue.lazyWaitGroup.Add(LazyNum)
	for i := 0; i < LazyNum; i++ {
		go n.LazyQueueSender()
	}

	// resend messages in the message queue
	if n.AckTimeout.Milliseconds() > 0 {
		n.sendWaitGroup.Add(SendNum)
		for i := 0; i < SendNum; i++ {
			go n.Sender()
		}
	}

	// regular daemons
	if n.AntiEntropyInterval != 0 {
		n.regWaitGroup.Add(1)
		go n.AntiEntropy()
	}
	if n.HeartbeatInterval != 0 {
		n.regWaitGroup.Add(1)
		go n.Heartbeat()
	}

	return nil
}

// Stop implements peer.Service
func (n *node) Stop() error {
	// if usePlumTree {
	// 	plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance("127.0.0.1:1")
	// 	plumTreeBroadcaster.Lock()
	// 	defer plumTreeBroadcaster.Unlock()
	// 	plumTreeBroadcaster.showEagerList(n.myAddr)
	// 	plumTreeBroadcaster.showLazyList(n.myAddr)
	// }
	//panic("to be implemented in HW0")

	close(n.plumTreeTable.queue.lazyQueueStop)
	n.plumTreeTable.queue.lazyWaitGroup.Wait()

	close(n.procStop)
	n.procWaitGroup.Wait()

	close(n.recvStop)
	n.recvWaitGroup.Wait()

	close(n.sendStop)
	n.sendWaitGroup.Wait()

	close(n.regStop)
	n.regWaitGroup.Wait()
	return nil
}

// Unicast implements peer.Messaging
func (n *node) Unicast(dest string, msg transport.Message) error {
	//panic("to be implemented in HW0")
	nextArr, ok := n.routingTable.Get(dest)
	if !ok {
		return errors.New("unicast error : could not find the address of next hop")
	}
	err := n.SendMsg(dest, nextArr, msg)
	return err
}

// Broadcast sends a packet asynchronously to all know destinations.
// The node must not send the message to itself (to its socket),
// but still process it.
//
// - implemented in HW1
func (n *node) Broadcast(msg transport.Message) error {
	//panic("to be implemented in HW1")
	//1.prepare message, increase self sequence number
	header := transport.NewHeader(n.myAddr, n.myAddr, n.myAddr, 0)
	pkt := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}
	rumorMsg := n.rumorTable.NewRumorsMessage(n.myAddr, msg)

	//2. send to a neighbor, in plumtree implement we send to all neighbors
	if usePlumTree {
		plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(n.myAddr)
		plumTreeBroadcaster.Lock()
		defer plumTreeBroadcaster.Unlock()

		msgHash := rumorMsg.Rumors[0].Origin + ";" + strconv.Itoa(int(rumorMsg.Rumors[0].Sequence))
		status, _ := plumTreeBroadcaster.computeIfAbsent(msgHash)
		status.receivedMsg = true
		for _, peer := range plumTreeBroadcaster.getEagerList() {
			if peer == n.myAddr {
				continue
			}
			header := transport.NewHeader(n.myAddr, n.myAddr, n.myAddr, 0)
			pkt := transport.Packet{
				Header: &header,
				Msg:    &msg,
			}
			err := n.PushRumorQueueAndSend(&rumorMsg, peer, pkt.Header.PacketID)
			if err != nil {
				return errors.New("broadcast error: " + err.Error())
			}
		}

		transportIhaveMsg, _ := n.MessageRegistry.MarshalMessage(plumTreeBroadcaster.prepareIHaveMsg(msgHash))

		go n.sendIHaveMsg(n.myAddr, transportIhaveMsg, plumTreeBroadcaster.iHaveInterval)

	} else {
		neighbor, _ := n.GetNeighbor("")

		err := n.PushRumorQueueAndSend(&rumorMsg, neighbor, pkt.Header.PacketID)

		if err != nil {
			return errors.New("broadcast error: " + err.Error())
		}
	}

	// 3. process message locally
	return n.MessageRegistry.ProcessPacket(pkt)
}

func (n *node) Relay(pkt transport.Packet) error {
	dest := pkt.Header.Destination
	nextAddr, ok := n.routingTable.Get(dest)
	if !ok {
		return errors.New("relay error: can not find next hop in routing table")
	}
	newHeader := transport.NewHeader(
		pkt.Header.Source,
		n.myAddr,
		dest,
		pkt.Header.TTL-1,
	)
	newPkt := transport.Packet{
		Header: &newHeader,
		Msg:    pkt.Msg,
	}
	return n.Socket.Send(nextAddr, newPkt, time.Second)
}

// AddPeer implements peer.Service
func (n *node) AddPeer(addr ...string) {
	//panic("to be implemented in HW0")
	for i := 0; i < len(addr); i++ {
		n.routingTable.Set(addr[i], addr[i])
		if usePlumTree {
			if n.myAddr != addr[i] {
				n.plumTreeTable.addNeighbor(addr[i])
			}
		}
	}
}

// GetRoutingTable implements peer.Service
func (n *node) GetRoutingTable() peer.RoutingTable {
	//panic("to be implemented in HW0")
	return n.routingTable.GetAll()
}

// SetRoutingEntry implements peer.Service
func (n *node) SetRoutingEntry(origin, relayAddr string) {
	//panic("to be implemented in HW0")
	if relayAddr == "" {
		n.routingTable.Delete(origin)
	} else if origin == relayAddr {
		n.AddPeer(origin)
	} else {
		n.routingTable.Set(origin, relayAddr)
	}
}

// Upload implements peer.DataSharing
func (n *node) Upload(data io.Reader) (metahash string, err error) {
	if n.a != nil {
		return n.a.Upload(data)
	}

	store := n.Storage.GetDataBlobStore()
	//--------bug-------------
	//bufCollector := bytes.Buffer{}
	//dataLen := 0
	// tmp := make([]byte, n.ChunkSize)
	// for {
	// 	l, err := data.Read(tmp)
	// 	dataLen += l
	// 	bufCollector.Write(tmp)
	// 	if err != nil {
	// 		break
	// 	}
	// }
	// buf := bufCollector.Bytes()
	//--------bug-------------
	buf, err := io.ReadAll(data)
	dataLen := len(buf)
	if err != nil {
		return "", err
	}
	metafileKeyBuf := bytes.Buffer{}
	metafileValueBuf := bytes.Buffer{}
	chunkSize := int(n.ChunkSize)
	curSize := 0
	bufPtr := 0
	for dataLen > 0 {
		if dataLen > chunkSize {
			dataLen -= chunkSize
			curSize = chunkSize
		} else {
			curSize = dataLen
			dataLen = 0
		}

		chunk := make([]byte, curSize)
		copy(chunk, buf[bufPtr:bufPtr+curSize])
		bufPtr += curSize

		chunkSHA256 := SHA256(chunk)
		_, err = metafileKeyBuf.Write(chunkSHA256)
		if err != nil {
			return "", err
		}
		chunkSHA256Str := hex.EncodeToString(chunkSHA256)
		_, err = metafileValueBuf.WriteString(chunkSHA256Str)
		if err != nil {
			return "", err
		}
		if dataLen > 0 {
			_, err = metafileValueBuf.WriteString(peer.MetafileSep)
			if err != nil {
				return "", err
			}
		}

		store.Set(chunkSHA256Str, chunk)
	}
	metafileValue := metafileValueBuf.Bytes()
	metafileKey := hex.EncodeToString(SHA256(metafileKeyBuf.Bytes()))
	store.Set(metafileKey, metafileValue)

	return metafileKey, nil
}

// GetCatalog implements peer.DataSharing
func (n *node) GetCatalog() peer.Catalog {
	return n.catalog.GetAll()
}

// UpdateCatalog implements peer.DataSharing
func (n *node) UpdateCatalog(key string, peer string) {
	n.catalog.Set(key, peer)
}

// Download implements peer.DataSharing
func (n *node) Download(metahash string) ([]byte, error) {
	if n.h != nil {
		return n.h.Download(metahash)
	}

	metafile, err := n.FetchFile(metahash)
	if err != nil {
		return nil, err
	}
	chunkKeys := strings.Split(string(metafile), peer.MetafileSep)
	file := bytes.Buffer{}
	for _, chunkKey := range chunkKeys {
		chunk, err := n.FetchFile(chunkKey)
		if err != nil {
			return nil, err
		}
		file.Write(chunk)
	}
	return file.Bytes(), nil
}

// Tag implements peer.DataSharing
func (n *node) Tag(name string, mh string) error {
	if n.TotalPeers <= 1 {
		n.HrStorage.GetHrNamingStore().Set(n.myAddr, name, []byte(mh))
		return nil
	}


	if n.HrStorage.GetHrNamingStore().Get(n.myAddr, name) != nil {
		return xerrors.Errorf("tag error: name already exists in naming store")
	}


	if pbft { // pbft is on

		flag := false

		for {

			// verifying tagging before exit
			mhTmp := n.HrStorage.GetHrNamingStore().Get(n.myAddr, name)
			if string(mhTmp) == mh {
				//fmt.Println("tag finished !!!!!!!!!!!!!!!!!!!!!!!!!!")
				return nil
			}

			// define pbft request message
			request := types.PbftRequest{
				IDhr:     n.myAddr,
				Filename: name,
				Metahash: mh,
			}


			n.pbft.lock.Lock()


			// assign PBFT ID for a new tagging
			if !flag {
				n.pbft.AssignPbftID(n.TotalPeers)
				//fmt.Printf("at step %d, hr id: %s, file name: %s and hash: %s\n", n.pbft.pbftID, n.myAddr, name, mh)
			}

			// sign the signature
			digest := getDigest(request)
			digestByte, _ := hex.DecodeString(digest)
			signature := signRSA(digestByte, n.pbft.privateKeyRSA)

			// store the preprepare message locally
			n.pbft.logPrePrepareMsg[n.pbft.pbftID] = digest
			n.pbft.logRequest[digest] = request


			// broadcast preprepare message
			pp := types.PrePrepareMessage{
				ViewID:			n.pbft.viewID,
				RequestMessage: request,
				Digest:         digest,
				PbftID:         n.pbft.pbftID,
				SrcAddr:        n.myAddr,
				Signature:      signature,
			}


			transPp, err := n.MessageRegistry.MarshalMessage(pp)
			if err != nil {
				return err
			}

			//fmt.Printf("%s, PrePrepare broadcast finished\n", n.myAddr)
			n.Broadcast(transPp)

            // checkpoint 
			if !flag && (pp.PbftID % checkpointInterval == 0){
				// broadcast preprepare message
				cp := types.CheckpointMessage{
					Digest:         digest,
					PbftID:         pp.PbftID,
					SrcAddr:        n.myAddr,
					Signature:      signature,
				}


				transCp, err := n.MessageRegistry.MarshalMessage(cp)
				if err != nil {
					return err
				}

				n.Broadcast(transCp)

                if _, ok := n.pbft.logCheckpointMsg[pp.PbftID]; !ok {
					n.pbft.logCheckpointMsg[pp.PbftID] = make(map[string]string)
				}
			
				// store the checkpoint message locally
				n.pbft.logCheckpointMsg[pp.PbftID][n.pbft.addr] = pp.Digest
			}

			n.pbft.lock.Unlock()

			flag = true

			// broadcast interval
			<-time.After(10 * time.Millisecond)

		}
	} else if !pbft {
		step := n.paxos.GetStep()

		// 1. if we have proposed, we need to wait consensus and try again
		if n.paxos.Proposed() {
			for {
				_, ok := n.paxos.GetConsensus(step)
				if ok {
					return n.Tag(name, mh)
				}
			}
		}

		// 2. if we have not proposed, we need to propose
		proposedVal := types.PaxosValue{
			UniqID:   xid.New().String(),
			Filename: name,
			Metahash: mh,
		}
		consensus, err := n.Propose(0, proposedVal)
		if err != nil {
			return err
		}
		if consensus == proposedVal {
			return nil
		}
		return n.Tag(name, mh)
	}

	return nil
}

// Resolve implements peer.DataSharing
func (n *node) Resolve(name string) (metahash string) {
	return string(n.Storage.GetNamingStore().Get(name))
}

// SearchAll implements peer.DataSharing
func (n *node) SearchAll(reg regexp.Regexp, budget uint, timeout time.Duration) (names []string, err error) {

	res := map[string]struct{}{} // remove duplication

	// 1. search locally
	n.Storage.GetNamingStore().ForEach(func(filename string, metahash []byte) bool {
		if reg.MatchString(filename) {
			res[filename] = struct{}{}
		}
		return true
	})

	// 2. search remotely
	requestID := xid.New().String()
	c := n.channelTable.Set(requestID)
	defer func() {
		n.channelTable.Delete(requestID)
	}()
	msg := types.SearchRequestMessage{
		RequestID: requestID,
		Origin:    n.myAddr,
		Pattern:   reg.String(),
		Budget:    budget,
	}
	ok, err := n.SearchFile(&msg, "")
	if !ok {
		return SetToArr(res), nil
	}

	// 3. wait for the reply
	stopFlag := time.After(timeout)
	for {
		select {
		case <-stopFlag:
			return SetToArr(res), nil
		case msg, ok := <-c:
			if !ok {
				return SetToArr(res), nil
			}
			searchReplyMsg, ok := msg.(*types.SearchReplyMessage)
			if !ok {
				return SetToArr(res), errors.New("in SearchAll, undefined message")
			}
			for _, info := range searchReplyMsg.Responses {
				res[info.Name] = struct{}{}
			}
		}
	}
}

// SearchFirst implements peer.DataSharing
func (n *node) SearchFirst(reg regexp.Regexp, conf peer.ExpandingRing) (name string, err error) {
	res := ""
	// 1. search locally
	n.Storage.GetNamingStore().ForEach(func(filename string, metahash []byte) bool {
		if reg.MatchString(filename) && res == "" && n.IsFullMatch(metahash) {
			res = filename
		}
		return true
	})
	if res != "" {
		return res, nil
	}

	// 2. search remotely
	requestID := xid.New().String()
	c := n.channelTable.Set(requestID)
	defer func() {
		n.channelTable.Delete(requestID)
	}()
	msg := types.SearchRequestMessage{
		RequestID: requestID,
		Origin:    n.myAddr,
		Pattern:   reg.String(),
		Budget:    conf.Initial,
	}
	ok, err := n.SearchFile(&msg, "")
	if !ok {
		return res, nil
	}
	n.searchRequestQueue.Push(&msg, requestID, &conf)

	// 3. wait for the reply
	for {
		select {
		case msg, ok := <-c:
			if !ok {
				return res, nil
			}
			searchReplyMsg, ok := msg.(*types.SearchReplyMessage)
			if !ok {
				return res, errors.New("in SearchAll, undefined message")
			}
			res, ok = n.GetFullMatch(searchReplyMsg)
			if ok {
				return res, nil
			}
		default:
			{
			}
		}

	}
}
