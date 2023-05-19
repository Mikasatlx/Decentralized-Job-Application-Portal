package impl

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

// We will put all ExecXXX message mechanisms in this file

// ChatMessage mechanism
func (n *node) ExecChatMessage(msg types.Message, pkt transport.Packet) error {
	chatMsg, ok := msg.(*types.ChatMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect chat message", msg)
	}
	log.Info().Str("node", n.myAddr).Msgf("received a new chat message: %s from %s", chatMsg.Message, pkt.Header.Source)
	return nil
}

// AckIHaveMessage mechanism
func (n *node) ExecAckIHaveMessage(msg types.Message, pkt transport.Packet) error {
	_, ok := msg.(*types.AckIHaveMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect chat message", msg)
	}
	n.plumTreeTable.queue.Lock()
	defer n.plumTreeTable.queue.Unlock()
	// log.Info().Str("node", n.myAddr).Msg("Received Ihave ack for " + i.MsgHash)
	n.plumTreeTable.queue.revMap[pkt.Header.PacketID] = struct{}{}
	return nil
}

// PruneMessage mechanism

func (n *node) ExecPruneMessage(msg types.Message, pkt transport.Packet) error {
	pruneMsg, ok := msg.(*types.PruneMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect chat message", msg)
	}
	res := strings.Split(pruneMsg.MsgHash, ";")
	source := res[0]
	plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(source)

	plumTreeBroadcaster.Lock()
	defer plumTreeBroadcaster.Unlock()
	plumTreeBroadcaster.moveToLazy(pkt.Header.Source)
	return nil
}

// IHaveMessage mechanism

func (n *node) ExecIHaveMessage(msg types.Message, pkt transport.Packet) error {
	ihMsg, ok := msg.(*types.IHaveMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect chat message", msg)
	}
	res := strings.Split(ihMsg.MsgHash, ";")
	source := res[0]
	plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(source)
	plumTreeBroadcaster.Lock()
	defer plumTreeBroadcaster.Unlock()
	status, _ := plumTreeBroadcaster.computeIfAbsent(ihMsg.MsgHash)
	if !status.receivedMsg {
		// log.Info().Str("node", n.myAddr).Msg("receive i have " + ihMsg.MsgHash + " need ")
		// plumTreeBroadcaster.addLazyPeer(pkt.Header.Source)
		// plumTreeBroadcaster.showEagerList(n.myAddr)
		// plumTreeBroadcaster.showLazyList(n.myAddr)
		status.lazyPeers = append(status.lazyPeers, pkt.Header.Source)
		if !status.requestGraft {
			status.requestGraft = true
			go n.sendGraftMsg(ihMsg.MsgHash, status, plumTreeBroadcaster)
		}
	}
	//IHave Msg Ack Mechanism
	ackIHaveMsg := types.AckIHaveMessage{
		MsgHash: ihMsg.MsgHash,
	}
	transportAckIHaveMsg, _ := n.MessageRegistry.MarshalMessage(ackIHaveMsg)
	err := n.SendMsgWithFixedID(pkt.Header.Source, pkt.Header.Source, transportAckIHaveMsg, pkt.Header.PacketID)
	if err != nil {
		n.SendTimeout()
	}
	return nil
}

// GraftMessage mechanism

func (n *node) ExecGraftMessage(msg types.Message, pkt transport.Packet) error {
	graftMsg, ok := msg.(*types.GraftMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect chat message", msg)
	}
	res := strings.Split(graftMsg.MsgHash, ";")
	source := res[0]
	plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(source)

	plumTreeBroadcaster.Lock()
	defer plumTreeBroadcaster.Unlock()
	plumTreeBroadcaster.moveToEager(pkt.Header.Source)

	seq, _ := strconv.Atoi(res[1])

	rumors := n.rumorTable.GetInterval(source, uint(seq-1), uint(seq))
	if len(rumors) == 0 {
		return nil
	}

	rumorMsg := types.RumorsMessage{
		Rumors: rumors,
	}
	err := n.PushRumorQueueAndSend(&rumorMsg, pkt.Header.Source, pkt.Header.PacketID)
	if err != nil {
		return errors.New("broadcast error: " + err.Error())
	}
	return nil
}

func (n *node) ExecRumorMessage(msg types.Message, pkt transport.Packet) error {
	rumorMsg, ok := msg.(*types.RumorsMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect rumor message", msg)
	}
	// 1. update status
	hasNew := false
	for _, rumor := range rumorMsg.Rumors {
		if !n.routingTable.Contains(rumor.Origin) {
			n.routingTable.Set(rumor.Origin, pkt.Header.RelayedBy)
		}
		res := n.rumorTable.TryToAcceptOrigin(rumor)

		if res == "prune" || res == "process" {
			if res == "process" {
				hasNew = true
				rumorPkt := transport.Packet{
					Header: pkt.Header,
					Msg:    rumor.Msg,
				}
				err := n.MessageRegistry.ProcessPacket(rumorPkt)
				if err != nil {
					return err
				}
			}

			if usePlumTree {
				plumTreeBroadcaster, _ := n.plumTreeTable.getPlumTreeInstance(rumor.Origin)
				plumTreeBroadcaster.Lock()
				defer plumTreeBroadcaster.Unlock()

				msgHash := rumor.Origin + ";" + strconv.Itoa(int(rumor.Sequence))
				status, _ := plumTreeBroadcaster.computeIfAbsent(msgHash)

				if !status.receivedMsg {
					// log.Info().Str("node", n.myAddr).Msg("Received rumor " + msgHash)
					plumTreeBroadcaster.considerNewEagerPeer(pkt.Header.Source)
					status.receivedMsg = true

					if status.requestGraft {
						close(status.received)
						// log.Info().Str("node", n.myAddr).Msg("Received graft ack for " + msgHash)
					}

					// send to all eager
					for _, peer := range plumTreeBroadcaster.getEagerList() {
						if peer == pkt.Header.Source {
							continue
						}
						header := transport.NewHeader(n.myAddr, n.myAddr, peer, 0)

						rumorMsg := types.RumorsMessage{
							Rumors: []types.Rumor{rumor},
						}

						err := n.PushRumorQueueAndSend(&rumorMsg, peer, header.PacketID)
						if err != nil {
							n.SendTimeout()
						}
					}

					transportIhaveMsg, _ := n.MessageRegistry.MarshalMessage(plumTreeBroadcaster.prepareIHaveMsg(msgHash))

					go n.sendIHaveMsg(rumor.Origin, transportIhaveMsg, plumTreeBroadcaster.iHaveInterval)

				} else {
					//send back prune msg
					plumTreeBroadcaster.considerNewLazyPeer(pkt.Header.Source)
					pruneMsg := plumTreeBroadcaster.preparePruneMsg(msgHash)
					transportPruneMsg, _ := n.MessageRegistry.MarshalMessage(pruneMsg)
					err := n.SendMsg(pkt.Header.Source, pkt.Header.Source, transportPruneMsg)
					if err != nil {
						n.SendTimeout()
					}
				}

			}
		}
	}
	// 2. send back ack msg
	if pkt.Header.Source != n.myAddr && hasNew {
		statusMsg := n.rumorTable.GetStatusMsg()
		ackMsg := types.AckMessage{
			AckedPacketID: pkt.Header.PacketID,
			Status:        *statusMsg,
		}
		transportAckMsg, err := n.MessageRegistry.MarshalMessage(ackMsg)
		if err != nil {
			return err
		}
		go func() {
			err = n.SendMsg(pkt.Header.Source, pkt.Header.Source, transportAckMsg)
			if err != nil {
				n.SendTimeout()
			}
		}()
	}

	if !usePlumTree {
		// 3. if it has new msg, then we send to a new neighbor
		if hasNew {
			neighbor, ok := n.GetNeighbor(pkt.Header.RelayedBy)
			if !ok {
				return nil
			}
			transportRumorMsg, err := n.MessageRegistry.MarshalMessage(rumorMsg)
			if err != nil {
				return err
			}
			go func() {
				err := n.SendMsg(neighbor, neighbor, transportRumorMsg)
				if err != nil {
					n.SendTimeout()
				}
			}()
		}
	}

	return nil
}

func (n *node) ExecAckMessage(msg types.Message, pkt transport.Packet) error {
	ackMsg, ok := msg.(*types.AckMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect ack message", msg)
	}
	// 1. stop waiting ack
	n.rumorQueue.Remove(ackMsg.AckedPacketID)

	if usePlumTree {
		return nil // we don't need to process status msg in plum tree implement
	}
	// 2. process statusMsg inside ack
	statusMsgPkt, err := n.GetStatusPkt(&ackMsg.Status, pkt.Header)
	if err != nil {
		return err
	}
	return n.MessageRegistry.ProcessPacket(*statusMsgPkt)
}

func (n *node) ExecStatusMessage(msg types.Message, pkt transport.Packet) error {
	statusMsg, ok := msg.(*types.StatusMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect status message", msg)
	}

	remoteStatus := *statusMsg
	myStatus := *(n.rumorTable.GetStatusMsg())

	// 1. find out if there are new rumors at my side
	myHasNew := false
	rumors := n.findMyNew(myStatus, remoteStatus)
	if len(rumors) > 0 {
		myHasNew = true
		rumorMsg := types.RumorsMessage{
			Rumors: rumors,
		}
		go func() {
			err := n.sendBack(rumorMsg, pkt)
			if err != nil {
				n.SendTimeout()
			}
		}()
	}

	// 2. find out that if there are new rumors at remote side
	remoteHasNew := n.findRemoteNew(myStatus, remoteStatus)
	if remoteHasNew {
		go func() {
			err := n.sendBack(myStatus, pkt)
			if err != nil {
				n.SendTimeout()
			}
		}()
	}

	//3. for a possibility that this peer will send its status to a NEW neighbor
	if !myHasNew && !remoteHasNew && rand.Float64() < n.ContinueMongering {
		transportStatusMsg, err := n.MessageRegistry.MarshalMessage(myStatus)
		if err != nil {
			return err
		}
		neigbor, ok := n.GetNeighbor(pkt.Header.RelayedBy)
		if !ok {
			return nil
		}
		go func() {
			err := n.SendMsg(neigbor, neigbor, transportStatusMsg)
			if err != nil {
				n.SendTimeout()
			}
		}()
	}
	return nil
}

func (n *node) ExecPrivateMessage(msg types.Message, pkt transport.Packet) error {
	privateMsg, ok := msg.(*types.PrivateMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect private message", msg)
	}
	_, ok = privateMsg.Recipients[n.myAddr]
	if ok {
		privateMsgPkt := transport.Packet{
			Header: pkt.Header,
			Msg:    privateMsg.Msg,
		}
		return n.MessageRegistry.ProcessPacket(privateMsgPkt)
	}
	return nil
}

func (n *node) ExecEmptyMessage(msg types.Message, pkt transport.Packet) error {
	_, ok := msg.(*types.EmptyMessage)
	if !ok {
		return xerrors.Errorf("wrong type: %T, we expect empty message", msg)
	}
	// do nothing
	return nil
}

func (n *node) ExecDataRequestMessage(msg types.Message, pkt transport.Packet) error {
	dataRequestMsg, ok := msg.(*types.DataRequestMessage)
	if !ok {
		return xerrors.Errorf("wrong type: %T", msg)
	}
	dataReplyMsg := types.DataReplyMessage{
		RequestID: dataRequestMsg.RequestID,
		Key:       dataRequestMsg.Key,
		Value:     n.Storage.GetDataBlobStore().Get(dataRequestMsg.Key),
	}
	return n.SendDataReply(&dataReplyMsg, pkt.Header.Source)
}

func (n *node) ExecDataReplyMessage(msg types.Message, pkt transport.Packet) error {
	dataReplyMsg, ok := msg.(*types.DataReplyMessage)
	if !ok {
		return xerrors.Errorf("wrong type: %T", msg)
	}
	// 1. notify the FetchFile function
	n.channelTable.FeedMsg(dataReplyMsg.RequestID, msg)

	// 2. stop waiting ack
	n.dataRequestQueue.Remove(dataReplyMsg.RequestID)

	// 3. update the local store
	n.Storage.GetDataBlobStore().Set(dataReplyMsg.Key, dataReplyMsg.Value)
	return nil
}

func (n *node) ExecSearchRequestMessage(msg types.Message, pkt transport.Packet) error {
	searchRequestMsg, ok := msg.(*types.SearchRequestMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect SearchRequestMessage", msg)
	}

	// 1. update routing table(do we need to do that?)
	if !n.routingTable.Contains(searchRequestMsg.Origin) {
		n.SetRoutingEntry(searchRequestMsg.Origin, pkt.Header.RelayedBy)
	}

	// 2. forward the request message
	searchRequestMsg.Budget--
	if searchRequestMsg.Budget > 0 {
		_, err := n.SearchFile(searchRequestMsg, pkt.Header.RelayedBy)
		if err != nil {
			return err
		}
	}

	// 3. search locally
	infos := []types.FileInfo{}
	reg := regexp.MustCompile(searchRequestMsg.Pattern)
	n.Storage.GetNamingStore().ForEach(func(name string, metahash []byte) bool {
		if reg.MatchString(name) {
			chunks, ok := n.GetAllChunkHash(metahash)
			if ok {
				info := types.FileInfo{
					Name:     name,
					Metahash: string(metahash),
					Chunks:   chunks,
				}
				infos = append(infos, info)
			}
		}
		return true
	})

	// 3. reply to the origin of the request
	searchReplyMsg := types.SearchReplyMessage{
		RequestID: searchRequestMsg.RequestID,
		Responses: infos,
	}
	transporMsg, err := n.MessageRegistry.MarshalMessage(searchReplyMsg)
	if err != nil {
		return err
	}
	return n.SendMsg(searchRequestMsg.Origin, pkt.Header.RelayedBy, transporMsg)
}

func (n *node) ExecSearchReplyMessage(msg types.Message, pkt transport.Packet) error {
	searchReplyMsg, ok := msg.(*types.SearchReplyMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect SearchReplyMessage", msg)
	}

	// 1. update naming store and catalog
	for _, info := range searchReplyMsg.Responses {
		n.Storage.GetNamingStore().Set(info.Name, []byte(info.Metahash))
		n.UpdateCatalog(info.Metahash, pkt.Header.Source)
		for _, chunk := range info.Chunks {
			if chunk != nil {
				n.UpdateCatalog(string(chunk), pkt.Header.Source)
			}
		}
	}

	// 2. notify the waiting request
	n.channelTable.FeedMsg(searchReplyMsg.RequestID, msg)

	// // 3. stop retransmitting request
	// n.searchRequestQueue.Remove(searchReplyMsg.RequestID)

	return nil
}

func (n *node) ExecPaxosPrepareMessage(msg types.Message, pkt transport.Packet) error {
	prepare, ok := msg.(*types.PaxosPrepareMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PaxosPrepareMessage", msg)
	}
	acceptedValue, acceptedID, ok := n.paxos.ReceivePrepare(*prepare)

	// 1. ignore
	if !ok {
		return nil
	}

	//2. response with paxos promise message
	return n.ResponsePrepare(prepare, acceptedValue, acceptedID)
}

func (n *node) ExecPaxosProposeMessage(msg types.Message, pkt transport.Packet) error {
	propose, ok := msg.(*types.PaxosProposeMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PaxosProposeMessage", msg)
	}
	ok = n.paxos.ReceivePropose(*propose)

	// 1. ignore
	if !ok {
		return nil
	}

	// 2. response with paxos accept message
	return n.ResponsePropose(propose)
}

func (n *node) ExecPaxosPromiseMessage(msg types.Message, pkt transport.Packet) error {
	promise, ok := msg.(*types.PaxosPromiseMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PaxosPromiseMessage", msg)
	}
	threshold := n.PaxosThreshold(n.TotalPeers)
	n.paxos.ReceivePromise(promise, uint(threshold))

	return nil
}

func (n *node) ExecPaxosAcceptMessage(msg types.Message, pkt transport.Packet) error {
	accept, ok := msg.(*types.PaxosAcceptMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PaxosAcceptMessage", msg)
	}
	threshold := n.PaxosThreshold(n.TotalPeers)
	consensus, step := n.paxos.ReceiveAccept(accept, uint(threshold))

	// 0. step != -1 if we collect enough accept messages
	if step == -1 {
		return nil
	}

	// 1. create a block
	block := n.GetBlock(consensus, uint(step))

	// 2. broadcast TLC message
	err := n.BroadcastTLC(block.Index, block)
	if err != nil {
		return err
	}

	return nil
}

func (n *node) ExecTLCMessage(msg types.Message, pkt transport.Packet) error {
	tlc, ok := msg.(*types.TLCMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect TLCMessage", msg)
	}
	// 1. update receied tlcs
	threshold := n.PaxosThreshold(n.TotalPeers)
	n.paxos.ReceiveTLC(tlc, uint(threshold))

	// 2. recursively update according to tcls
	return n.TLCProcess(n.paxos.GetStep(), false)
}

func (n *node) ExecPubKeyMessage(msg types.Message, pkt transport.Packet) error {
	if n.cn == nil {
		return nil
	}
	pub, ok := msg.(*types.PubKeyMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PubKeyMessage", msg)
	}
	go n.cn.recvPubKey(pub)
	return nil
}

func (n *node) ExecDealMessage(msg types.Message, pkt transport.Packet) error {
	if n.cn == nil {
		return nil
	}
	deal, ok := msg.(*types.DealMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect DealMessage", msg)
	}
	go n.cn.recvDeal(deal)
	return nil
}

func (n *node) ExecDealResponseMessage(msg types.Message, pkt transport.Packet) error {
	if n.cn == nil {
		return nil
	}
	dealRes, ok := msg.(*types.DealResponseMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect DealResponseMessage", msg)
	}
	go n.cn.recvDealResponse(dealRes)
	return nil
}

func (n *node) ExecPKRequestMessage(msg types.Message, pkt transport.Packet) error {
	if n.cn == nil {
		return nil
	}
	pkReq, ok := msg.(*types.PKRequestMessage)

	fmt.Println("ExecPKRequestMessage", pkReq.Address)

	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PKRequestMessage", msg)
	}
	go n.cn.recvPKRequestMessage(pkReq)
	return nil
}

func (n *node) ExecPKResponseMessage(msg types.Message, pkt transport.Packet) error {
	if n.a == nil {
		return nil
	}

	pkRes, ok := msg.(*types.PKResponseMessage)
	fmt.Println("ExecPKResponseMessage", pkRes.PubKey)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PKResponseMessage", msg)
	}
	go n.a.recvPK(pkRes.PubKey)
	return nil
}

func (n *node) ExecEncryptedFileMessage(msg types.Message, pkt transport.Packet) error {
	fmt.Println("ExecEncryptedFileMessage")
	if n.cn == nil {
		return nil
	}

	enFile, ok := msg.(*types.EncryptedFileMessage)

	fmt.Println("ExecEncryptedFileMessage", enFile.ID)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect EncryptedFileMessage", msg)
	}
	go n.cn.recvEncryptedFileMessage(enFile)
	return nil
}

func (n *node) ExecEncryptedChunkMessage(msg types.Message, pkt transport.Packet) error {
	fmt.Println("ExecEncryptedChunkMessage")
	if n.cn == nil {
		return nil
	}

	enChunk, ok := msg.(*types.EncryptedChunkMessage)

	fmt.Println("ExecEncryptedChunkMessage", enChunk.ChunkKey)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect EncryptedChunkMessage", msg)
	}
	go n.cn.recvEncryptedChunkMessage(enChunk)
	return nil
}

func (n *node) ExecRequestFileMessage(msg types.Message, pkt transport.Packet) error {
	if n.cn == nil {
		return nil
	}

	reqFile, ok := msg.(*types.RequestFileMessage)

	fmt.Println("ExecRequestFileMessage", reqFile.HRID)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect RequestFileMessage", msg)
	}
	go n.cn.recvRequestFileMessage(reqFile)
	return nil
}

func (n *node) ExecResponseFileMessage(msg types.Message, pkt transport.Packet) error {
	if n.h == nil {
		return nil
	}
	resFile, ok := msg.(*types.ResponseFileMessage)
	n.h.keyMux.RLock()
	curKey := n.h.metaFileKey
	n.h.keyMux.RUnlock()

	if curKey != resFile.MetaFileID {
		return nil
	}

	fmt.Println("ExecResponseFileMessage", resFile.MetaFileID)

	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect ResponseFileMessage", msg)
	}
	go n.h.recvResponseFileMessage(resFile)
	return nil
}
