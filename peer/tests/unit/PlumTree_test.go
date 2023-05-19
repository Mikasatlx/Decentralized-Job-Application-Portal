package unit

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/internal/graph"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/channel"
	"go.dedis.ch/cs438/types"
)

// 1-1
//
// If I broadcast a message as a rumor and I have only one neighbor, then that
// neighbor should receive my message, answer with an ack message, and update
// its routing table.
func Test_PlumTree_Messaging_Broadcast_Rumor_Simple(t *testing.T) {

	transp := channel.NewTransport()

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:1", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:2", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:3", z.WithContinueMongering(0), z.WithAckTimeout(0))
	// defer node3.Stop()

	node4 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:4", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node4.Stop()

	node5 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:5", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node5.Stop()

	node1.AddPeer(node2.GetAddr())
	node1.AddPeer(node3.GetAddr())

	node2.AddPeer(node1.GetAddr())
	node2.AddPeer(node4.GetAddr())

	node3.AddPeer(node1.GetAddr())
	node3.AddPeer(node5.GetAddr())

	node4.AddPeer(node2.GetAddr())
	node4.AddPeer(node5.GetAddr())

	node5.AddPeer(node1.GetAddr())
	node5.AddPeer(node4.GetAddr())

	nodes := []z.TestNode{node1, node2, node3, node4, node5}
	msgNum := 20

	for _, node := range nodes {

		go func(node z.TestNode) {

			for num := 1; num <= msgNum; num++ {

				if num == 10 && node.GetAddr() == "127.0.0.1:3" {
					node.Stop()
					return
				}

				chat := types.ChatMessage{
					Message: fmt.Sprintf("Msg num %s from %s ", strconv.Itoa(num), node.GetAddr()),
				}
				data, _ := json.Marshal(&chat)

				msg := transport.Message{
					Type:    chat.Name(),
					Payload: data,
				}
				err := node.Broadcast(msg)
				require.NoError(t, err)
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(500)))
			}

		}(node)

	}

	time.Sleep(time.Millisecond * 20000)

	for _, node := range nodes {
		if node.GetAddr() == "127.0.0.1:3" {
			continue
		}
		actual := node.GetChatMsgs()
		require.Len(t, actual, len(nodes)*msgNum-11)
	}

}

func Test_PlumTree_Messaging_Broadcast_Rumor_Down(t *testing.T) {

	transp := channel.NewTransport()

	chat1 := types.ChatMessage{
		Message: "chat1",
	}
	data1, _ := json.Marshal(&chat1)

	msg1 := transport.Message{
		Type:    chat1.Name(),
		Payload: data1,
	}

	chat2 := types.ChatMessage{
		Message: "chat2",
	}
	data2, _ := json.Marshal(&chat2)

	msg2 := transport.Message{
		Type:    chat1.Name(),
		Payload: data2,
	}

	node1 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:1", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node1.Stop()

	node2 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:2", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node2.Stop()

	node3 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:3", z.WithContinueMongering(0), z.WithAckTimeout(0))
	// defer node3.Stop()

	node4 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:4", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node4.Stop()

	node5 := z.NewTestNode(t, peerFac, transp, "127.0.0.1:5", z.WithContinueMongering(0), z.WithAckTimeout(0))
	defer node5.Stop()

	node1.AddPeer(node2.GetAddr())
	node1.AddPeer(node3.GetAddr())

	node2.AddPeer(node1.GetAddr())
	node2.AddPeer(node4.GetAddr())

	node3.AddPeer(node1.GetAddr())
	node3.AddPeer(node5.GetAddr())

	node4.AddPeer(node2.GetAddr())
	node4.AddPeer(node5.GetAddr())

	node5.AddPeer(node1.GetAddr())
	node5.AddPeer(node4.GetAddr())

	err := node1.Broadcast(msg1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 1000)

	node3.Stop()

	err = node1.Broadcast(msg2)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 4000)

	n1c := node1.GetChatMsgs()
	require.Len(t, n1c, 2)
	n2c := node2.GetChatMsgs()
	require.Len(t, n2c, 2)
	n3c := node3.GetChatMsgs()
	require.Len(t, n3c, 1)
	n4c := node4.GetChatMsgs()
	require.Len(t, n4c, 2)
	n5c := node5.GetChatMsgs()
	require.Len(t, n5c, 1)

	time.Sleep(time.Millisecond * 7000)

	n1c = node1.GetChatMsgs()
	require.Len(t, n1c, 2)
	n2c = node2.GetChatMsgs()
	require.Len(t, n2c, 2)
	n3c = node3.GetChatMsgs()
	require.Len(t, n3c, 1)
	n4c = node4.GetChatMsgs()
	require.Len(t, n4c, 2)
	n5c = node5.GetChatMsgs()
	require.Len(t, n5c, 2)

}

// Test the sending of chat messages in rumor on a "big" network. The topology
// is generated randomly, we expect every node to receive chat messages from
// every other nodes.
func Test_PlumTree_Messaging_Broadcast_BigGraph(t *testing.T) {

	rand.Seed(1)

	n := 10
	chatMsg := "hi from %s index %s"
	stopped := false

	transp := channel.NewTransport()

	nodes := make([]z.TestNode, n)

	stopNodes := func() {
		if stopped {
			return
		}

		defer func() {
			stopped = true
		}()

		wait := sync.WaitGroup{}
		wait.Add(len(nodes))

		for i := range nodes {
			go func(node z.TestNode) {
				defer wait.Done()
				node.Stop()
			}(nodes[i])
		}

		t.Log("stopping nodes...")

		done := make(chan struct{})

		go func() {
			select {
			case <-done:
			case <-time.After(time.Minute * 5):
				t.Error("timeout on node stop")
			}
		}()

		wait.Wait()
		close(done)
	}

	for i := 0; i < n; i++ {
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0",
			z.WithAntiEntropy(0),
			// since everyone is sending a rumor, there is no need to have route
			// rumors
			z.WithHeartbeat(0),
			z.WithAckTimeout(3))

		nodes[i] = node
	}

	defer stopNodes()

	// out, err := os.Create("topology.dot")
	// require.NoError(t, err)
	// graph.NewGraph(0.3).Generate(io.Discard, nodes)
	graph.NewGraph(0.3).Generate(os.Stdout, nodes)

	// > make each node broadcast a rumor, each node should eventually get
	// rumors from all the other nodes.

	wait := sync.WaitGroup{}
	wait.Add(len(nodes))
	msgnum := 5

	for i := range nodes {
		go func(node z.TestNode) {
			defer wait.Done()
			for j := 1; j <= msgnum; j++ {
				chat := types.ChatMessage{
					Message: fmt.Sprintf(chatMsg, node.GetAddr(), strconv.Itoa(j)),
				}
				data, err := json.Marshal(&chat)
				require.NoError(t, err)

				msg := transport.Message{
					Type:    chat.Name(),
					Payload: data,
				}

				// this is a key factor: the later a message is sent, the more time
				// it takes to be propagated in the network.
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))

				err = node.Broadcast(msg)
				require.NoError(t, err)
			}

		}(nodes[i])
	}

	time.Sleep(time.Millisecond * 2000 * time.Duration(n))

	done := make(chan struct{})

	go func() {
		select {
		case <-done:
		case <-time.After(time.Minute * 5):
			t.Error("timeout on node broadcast")
		}
	}()

	wait.Wait()
	close(done)

	stopNodes()

	// > check that each node got all the chat messages

	nodesChatMsgs := make([][]*types.ChatMessage, len(nodes))

	for i, node := range nodes {
		chatMsgs := node.GetChatMsgs()
		nodesChatMsgs[i] = chatMsgs
	}

	// > each nodes should get the same messages as the first node. We sort the
	// messages to compare them.

	expected := nodesChatMsgs[0]
	sort.Sort(types.ChatByMessage(expected))

	t.Logf("expected chat messages: %v", expected)
	require.Len(t, expected, len(nodes)*msgnum)

	for i := 1; i < len(nodesChatMsgs); i++ {
		compare := nodesChatMsgs[i]
		sort.Sort(types.ChatByMessage(compare))

		require.Equal(t, expected, compare)
	}

	// > every node should have an entry to every other nodes in their routing
	// tables.

	for _, node := range nodes {
		table := node.GetRoutingTable()
		require.Len(t, table, len(nodes))

		for _, otherNode := range nodes {
			_, ok := table[otherNode.GetAddr()]
			require.True(t, ok)
		}

		// uncomment the following to generate the routing table graphs
		// out, err := os.Create(fmt.Sprintf("node-%s.dot", node.GetAddr()))
		// require.NoError(t, err)

		// table.DisplayGraph(out)
	}
}
