package impl

import (
	"time"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

func (n *node) SendMsg(dest string, nextArr string, msg transport.Message) error {
	header := transport.NewHeader(
		n.myAddr,
		n.myAddr,
		dest,
		0)
	pkg := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}
	err := n.Socket.Send(nextArr, pkg, time.Second)
	if err != nil {
		n.SendTimeout()
	}
	return nil
}

func (n *node) SendMsgWithFixedID(dest string, nextArr string, msg transport.Message, pktID string) error {
	header := transport.NewHeader(
		n.myAddr,
		n.myAddr,
		dest,
		0)
	pkg := transport.Packet{
		Header: &header,
		Msg:    &msg,
	}
	pkg.Header.PacketID = pktID
	err := n.Socket.Send(nextArr, pkg, time.Second)
	if err != nil {
		n.SendTimeout()
	}
	return nil
}

func SetToArr(set map[string]struct{}) []string {
	res := []string{}
	for k := range set {
		res = append(res, k)
	}
	return res
}
func (n *node) SendTimeout() {
	//do nothing
}

func (n *node) BroadcastError() {
	//do nothing
}

func (n *node) SendPrivateMsg(msg types.Message, address string) {
	transportMsg, _ := n.MessageRegistry.MarshalMessage(msg)
	private := types.PrivateMessage{
		Recipients: map[string]struct{}{address: {}},
		Msg:        &transportMsg,
	}
	transPrivate, _ := n.MessageRegistry.MarshalMessage(&private)
	n.Broadcast(transPrivate)
}
