package impl

import (
	"time"

	"github.com/rs/zerolog/log"
	"go.dedis.ch/cs438/types"
)

// We will put all goroutines in this file

func (n *node) Processor() {
	defer n.procWaitGroup.Done()
	for {
		select {
		case <-n.procStop:
			return
		case pkt := <-n.procQueue:
			err := n.MessageRegistry.ProcessPacket(pkt)
			if err != nil {
				log.Err(err).Msg("processing packet fails:   " + pkt.String())
			}
		}
	}
}

func (n *node) Receiver() {
	defer n.recvWaitGroup.Done()
	for {
		select {
		case <-n.recvStop:
			return
		default:
			pkt, err := n.Socket.Recv(time.Second * 1)
			if err != nil {
				//log.Err(err).Msg("socket recv error(timeout)!")
				continue
			}
			if pkt.Header.Destination == n.myAddr {
				n.procQueue <- pkt
			} else {
				err = n.Relay(pkt)
				if err != nil {
					log.Err(err).Msg("relaying fails")
				}
			}
		}
	}
}

func (n *node) Sender() {
	defer n.sendWaitGroup.Done()
	for {
		<-time.After(1 * time.Millisecond)
		select {
		case <-n.sendStop:
			return
		default:
			n.ReSendRumor()
			n.ReSendDataRequest()
			n.ReSendSearchRequest()
		}
	}
}

func (n *node) ReSendRumor() {
	neighbor, rumor, id, ok := n.rumorQueue.GetRumor(n)
	if ok {
		// we should use the same Packet ID
		err := n.SendRumor(&rumor, neighbor, id)
		if err != nil {
			return
		}
		// the following is wrong, since it will renew a packet ID
		// transportRumorMsg, err := n.MessageRegistry.MarshalMessage(rumor)
		// if err != nil {
		// 	return
		// }
		//err = n.SendMsg(neighbor, neighbor, transportRumorMsg)
		// if err != nil {
		// 	return
		// }
	}
}

func (n *node) ReSendDataRequest() {
	dataRequest, dest, ok := n.dataRequestQueue.GetDataRequest(n)
	if ok {
		err := n.SendDataRequest(dataRequest, dest)
		if err != nil {
			return
		}
	}
}

func (n *node) ReSendSearchRequest() {
	searchRequest, ok := n.searchRequestQueue.GetSearchRequest(n)
	if ok {
		_, err := n.SearchFile(searchRequest, "")
		if err != nil {
			return
		}
	}
}

func (n *node) AntiEntropy() {
	defer n.regWaitGroup.Done()
	for {
		select {
		case <-n.regStop:
			return
		case <-time.After(n.AntiEntropyInterval):
			neighbor, ok := n.GetNeighbor("")
			if !ok {
				continue
			}
			statusMsg := n.rumorTable.GetStatusMsg()
			transportStatusMsg, err := n.MessageRegistry.MarshalMessage(statusMsg)
			if err != nil {
				log.Err(err).Msg("AntiEntropy MarshalMessage error")
				continue
			}
			err = n.SendMsg(neighbor, neighbor, transportStatusMsg)
			if err != nil {
				log.Err(err).Msg("AntiEntropy SendMsg error")
			}
		}
	}
}

func (n *node) Heartbeat() {
	defer n.regWaitGroup.Done()
	for {
		err := n.SendEmptyMsg()
		if err != nil {
			log.Err(err).Msg("Heartbeat MarshalMessage error")
		}
		select {
		case <-n.regStop:
			return
		case <-time.After(n.HeartbeatInterval):
			err := n.SendEmptyMsg()
			if err != nil {
				log.Err(err).Msg("Heartbeat MarshalMessage error")
			}
		}
	}
}

func (n *node) SendEmptyMsg() error {
	emptyMsg := types.EmptyMessage{}
	transportEmptyMsg, err := n.MessageRegistry.MarshalMessage(emptyMsg)
	if err != nil {
		return err
	}
	return n.Broadcast(transportEmptyMsg)
}
