package impl

import (
	"fmt"
	"math"
	"sync"
	"time"

	"go.dedis.ch/cs438/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"go.dedis.ch/kyber/v3/share"
	dkg "go.dedis.ch/kyber/v3/share/dkg/pedersen"
)

var suite = edwards25519.NewBlakeSHA256Ed25519()

func NewCothorityNode(n *node, myIndex uint, addresses []string, HRSecretTable map[uint]string) *CothorityNode {
	myPriKey := suite.Scalar().Pick(suite.RandomStream())
	return &CothorityNode{
		node:          n,
		index:         myIndex,
		priKey:        myPriKey,
		pubKey:        suite.Point().Mul(myPriKey, nil),
		pubKeyTable:   NewSafePubKeyTable(len(addresses)),
		addresses:     addresses,
		dealTable:     NewSafeIndexTable(),
		resTable:      NewSafeIndexTable(),
		dkgDone:       false,
		dataTable:     map[string]Data{},
		HRSecretTable: HRSecretTable,
	}
}

type CothorityNode struct {
	node          *node
	index         uint
	pubKey        kyber.Point
	priKey        kyber.Scalar
	priShare      *share.PriShare
	pubKeyTable   *SafePubKeyTable
	dkg           *dkg.DistKeyGenerator
	addresses     []string
	dealTable     *SafeIndexTable
	resTable      *SafeIndexTable
	dkgMux        sync.RWMutex
	dkgDoneMux    sync.RWMutex
	dkgDone       bool
	dataTable     map[string]Data
	dataTableMux  sync.RWMutex
	HRSecretTable map[uint]string
	//recordPubKey  []byte
}

type Data struct {
	AppFileInfo []byte
	AppFileIV   []byte
	C           []byte
	U           []byte
}

// broadcast public key
func (n *CothorityNode) BroadcastPubKey() error {
	time.Sleep(time.Duration(10-n.index) * time.Second)
	pubKeyBuf, _ := n.pubKey.MarshalBinary()
	pubKeyMsg := types.PubKeyMessage{
		Index:  n.index,
		PubKey: pubKeyBuf,
	}
	transportPubKeyMessage, err := n.node.MessageRegistry.MarshalMessage(&pubKeyMsg)
	if err != nil {
		return err
	}
	fmt.Println("index", n.index, "BroadcastPubKey: ", n.pubKey)
	return n.node.Broadcast(transportPubKeyMessage)
}

// initailize dkg
func (n *CothorityNode) initDKG(pks []kyber.Point) error {
	t := int(math.Ceil(float64(n.pubKeyTable.num)/2) + 1)
	fmt.Println("initDKG: ", n.index, "threshold", t)
	dkg, err := dkg.NewDistKeyGenerator(suite, n.priKey, pks, t)
	if err != nil {
		fmt.Println("initDKG:", err)
		return err
	}

	n.dkgMux.Lock()
	n.dkg = dkg
	deals, err := n.dkg.Deals()
	n.dkgMux.Unlock()
	if err != nil {
		return err
	}
	go n.SendDeals(deals)

	go n.CheckCertified()
	return nil
}

// send deals
func (n *CothorityNode) SendDeals(deals map[int]*dkg.Deal) error {
	for i, deal := range deals {
		deal := types.DealMessage{
			Deal: *deal,
		}
		n.node.SendPrivateMsg(&deal, n.addresses[i])
	}
	return nil
}

func (cn *CothorityNode) CheckCertified() {
	cap := len(cn.addresses)
	eps := 1
	if cap > 10 {
		eps = 2
	}
	if cap > 15 {
		eps = 10
	}
	if cap > 20 {
		eps = 15
	}
	timeout := time.Duration(cap*eps) * time.Second
	ticker := time.NewTicker(timeout)
	<-ticker.C
	t := false
	for !t {
		cn.dkgMux.Lock()
		fmt.Println("id:", cn.index, "QUAL:", cn.dkg.QUAL(), "Receive Deals:", cn.dealTable.Size(), "Receive DealResponse:", cn.resTable.Size())
		t = cn.dkg.ThresholdCertified()
		if t {
			cn.dkg.SetTimeout()
			distrKey, _ := cn.dkg.DistKeyShare()
			cn.priShare = distrKey.PriShare()
			cn.pubKey = distrKey.Public()
			ticker.Stop()
			fmt.Println("id:", cn.index, "Receive Deals:", cn.dealTable.Size(), "Receive DealResponse:", cn.resTable.Size(), "QUAL:", cn.dkg.QualifiedShares(), "Public key: ", cn.pubKey)
			cn.dkgMux.Unlock()

			cn.dkgDoneMux.Lock()
			cn.dkgDone = true
			cn.dkgDoneMux.Unlock()
		} else {
			cn.dkgMux.Unlock()
			time.Sleep(time.Second)
		}
	}
}
