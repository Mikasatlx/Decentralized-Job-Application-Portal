package unit

import (
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/peer/impl"
)

// var udpFac transport.Factory = udp.NewUDP
var studentFac peer.Factory = impl.NewPeer

func Test_DKG(t *testing.T) {

	rand.Seed(1)
	nStudent := 3

	stopped := false

	studentTransp := udpFac()

	antiEntropy := time.Second * 10
	ackTimeout := time.Second * 10
	initTime := time.Second * 10
	//waitPerNode := time.Second * 5

	if runtime.GOOS == "windows" {
		antiEntropy = time.Second * 20
		ackTimeout = time.Second * 90
		//waitPerNode = time.Second * 15
	}

	nodes := make([]z.TestNode, nStudent)
	addresses := make([]string, nStudent)
	for i := 0; i < nStudent; i++ {
		node := z.NewTestNode(t, studentFac, studentTransp, "127.0.0.1:0",
			z.WithAntiEntropy(antiEntropy),
			// since everyone is sending a rumor, there is no need to have route
			// rumors
			z.WithHeartbeat(0),
			z.WithAckTimeout(ackTimeout))
		addresses[i] = node.GetAddr()
		nodes[i] = node
	}

	// fully connected
	for i := 0; i < nStudent; i++ {
		for j := 0; j < nStudent; j++ {
			nodes[i].AddPeer(nodes[j].GetAddr())
		}
	}

	// create cothority
	for i := 0; i < nStudent; i++ {
		nodes[i].CreateCothorityNode(uint(i), addresses, nil)
	}

	// start cothority
	for i := 0; i < nStudent; i++ {
		//time.Sleep(time.Second * time.Duration(i))
		go nodes[i].StartCothorityNode()
	}

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

	defer stopNodes()
	//time.Sleep(waitPerNode * time.Duration(5)) //nStudent
	time.Sleep(initTime + time.Second*20)
}
