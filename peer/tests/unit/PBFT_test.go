package unit

import (
	"encoding/hex"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
	"fmt"

	"github.com/stretchr/testify/require"
	z "go.dedis.ch/cs438/internal/testing"
	//"go.dedis.ch/cs438/storage"
	//"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/transport/channel"
	//"go.dedis.ch/cs438/types"
)

// Call the Tag() function on multiple peers concurrently with no malicious nodes.
func Test_PBFT_no_malicious(t *testing.T) {

	numMessages := 10
	numNodes := 4

	transp := channel.NewTransport()
	nodes := make([]z.TestNode, numNodes)


	for i := range nodes {

		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i+1)))
		defer node.Stop()

		nodes[i] = node
	}


	for _, n1 := range nodes {
		for _, n2 := range nodes {
			n1.AddPeer(n2.GetAddr())
		}
	}


	wait := sync.WaitGroup{}
	wait.Add(numNodes)


	for _, node := range nodes {

		go func(n z.TestNode) {
			defer wait.Done()

			for i := 0; i < numMessages; i++ {

				name := make([]byte, 12)
				rand.Read(name)

				err := n.Tag(hex.EncodeToString(name), "1")
				require.NoError(t, err)

				time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))
			}
		}(node)
	}

	wait.Wait()
	fmt.Println("Waiting finished !!!")

	time.Sleep(time.Second * 1)


	for i, node := range nodes {
		t.Logf("node %d", i)
		z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:1", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:2", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:3", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:3", node.GetHrStorage().GetHrBlockchainStore())
	}
}


// Call the Tag() function on multiple peers concurrently with one malicious node.
func Test_PBFT_one_malicious(t *testing.T) {
	numMessages := 10
	numNodes := 4

	transp := channel.NewTransport()
	nodes := make([]z.TestNode, numNodes)

	for i := range nodes {

		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i+1)))
		defer node.Stop()

		nodes[i] = node
	}

	for _, n1 := range nodes {
		for _, n2 := range nodes {
			n1.AddPeer(n2.GetAddr())
		}
	}


	wait := sync.WaitGroup{}
	wait.Add(numNodes)


	for j, node := range nodes {

		if j == 3 {
			node.TurnBad()
		}

		go func(n z.TestNode) {

			defer wait.Done()
			for i := 0; i < numMessages; i++ {
				name := make([]byte, 12)
				rand.Read(name)

				err := n.Tag(hex.EncodeToString(name), "1")
				require.NoError(t, err)

				time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))
			}
		}(node)
	}

	wait.Wait()
	fmt.Println("Waiting finished !!!")

	time.Sleep(time.Second * 1)

	for i, node := range nodes {

		t.Logf("node %d", i)
		z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:1", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:2", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:3", node.GetHrStorage().GetHrBlockchainStore())
	}
}

// Call the Tag() function on multiple peers concurrently with one malicious node.
func Test_PBFT_two_malicious(t *testing.T) {
	numMessages := 4
	numNodes := 4

	transp := channel.NewTransport()
	nodes := make([]z.TestNode, numNodes)

	for i := range nodes {
		
		node := z.NewTestNode(t, peerFac, transp, "127.0.0.1:0", z.WithTotalPeers(uint(numNodes)), z.WithPaxosID(uint(i+1)))
		defer node.Stop()

		nodes[i] = node
	}

	for _, n1 := range nodes {
		for _, n2 := range nodes {
			n1.AddPeer(n2.GetAddr())
		}
	}

	wait := sync.WaitGroup{}
	wait.Add(numNodes)

	for j, node := range nodes {

		if j == 3 || j == 2 {
			node.TurnBad()
		}

		go func(n z.TestNode) {
			defer wait.Done()
			for i := 0; i < numMessages; i++ {
				name := make([]byte, 12)
				rand.Read(name)

				err := n.Tag(hex.EncodeToString(name), "1")
				require.NoError(t, err)

				time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))
			}
		}(node)
	}

	wait.Wait()

	fmt.Println("Waiting finished !!!")

	time.Sleep(time.Second * 1)

	for i, node := range nodes {
		t.Logf("node %d", i)
		z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:1", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:2", node.GetHrStorage().GetHrBlockchainStore())
		//z.DisplayPbftchainBlocks(t, os.Stdout, "127.0.0.1:3", node.GetHrStorage().GetHrBlockchainStore())
	}
}