package impl

import (
	"math/rand"
	"time"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// For the first time, we would send to a random neighbor
// For the next time, we would send to new random neighbor, different from the last one
func (n *node) GetNeighbor(preNeighbor string) (string, bool) {
	t := n.routingTable.GetAll()
	neighbors := []string{}
	for k, v := range t {
		if k == v && v != preNeighbor && k != n.myAddr {
			neighbors = append(neighbors, k)
		}
	}
	if len(neighbors) == 0 {
		return "", false
	}
	randNeighbor := neighbors[rand.Intn(len(neighbors))]
	return randNeighbor, true
}

func (n *node) GetNeighborList() []string {
	t := n.routingTable.GetAll()
	neighbors := []string{}
	for k, v := range t {
		if k == v && k != n.myAddr {
			neighbors = append(neighbors, k)
		}
	}
	return neighbors
}

func (n *node) PushRumorQueueAndSend(rumorMsg *types.RumorsMessage, neighbor string, id string) error {
	if neighbor == "" {
		n.rumorQueue.Push(rumorMsg, "", id)
		return nil
	}
	n.rumorQueue.Push(rumorMsg, neighbor, id)
	return n.SendRumor(rumorMsg, neighbor, id)
}

func (n *node) SendRumor(rumorMsg *types.RumorsMessage, neighbor string, id string) error {
	transportRumorMsg, err := n.MessageRegistry.MarshalMessage(rumorMsg)
	if err != nil {
		return err
	}
	// we should use the same packet ID
	header := transport.NewHeader(
		n.myAddr,
		n.myAddr,
		neighbor,
		0)
	header.PacketID = id
	pkg := transport.Packet{
		Header: &header,
		Msg:    &transportRumorMsg,
	}
	err = n.Socket.Send(neighbor, pkg, time.Second)
	if err != nil {
		n.SendTimeout()
	}
	return nil
	// we should not new a packet as following
	//return n.SendMsg(neighbor, neighbor, transportRumorMsg)
}

func (n *node) GetStatusPkt(statusMsg *types.StatusMessage, header *transport.Header) (*transport.Packet, error) {
	transpotStatusMsg, err := n.MessageRegistry.MarshalMessage(statusMsg)
	if err != nil {
		return nil, err
	}
	statusMsgPkt := transport.Packet{
		Header: header,
		Msg:    &transpotStatusMsg,
	}
	return &statusMsgPkt, nil
}

func (n *node) findMyNew(myStatus types.StatusMessage, remoteStatus types.StatusMessage) []types.Rumor {
	rumors := []types.Rumor{}
	for origin, mySeq := range myStatus {
		remoteSeq, ok := remoteStatus[origin]
		if !ok {
			remoteSeq = 0
		}
		if mySeq > remoteSeq {
			rumors = append(rumors, n.rumorTable.GetInterval(origin, remoteSeq, mySeq)...)
		}
	}
	return rumors
}

func (n *node) findRemoteNew(myStatus types.StatusMessage, remoteStatus types.StatusMessage) bool {
	remoteHasNew := false
	for origin, remoteSeq := range remoteStatus {
		mySeq, ok := myStatus[origin]
		if !ok {
			mySeq = 0
		}
		if mySeq < remoteSeq {
			remoteHasNew = true
			break
		}
	}
	return remoteHasNew
}

func (n *node) sendBack(msg types.Message, pkt transport.Packet) error {
	transpotStatusMsg, err := n.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return err
	}
	err = n.SendMsg(pkt.Header.Source, pkt.Header.RelayedBy, transpotStatusMsg)
	if err != nil {
		return err
	}
	return nil
}
