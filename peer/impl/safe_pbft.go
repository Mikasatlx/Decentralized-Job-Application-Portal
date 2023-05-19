package impl

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

	"go.dedis.ch/cs438/storage"
	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"

	"bytes"
	"crypto/sha256"
	"io/ioutil"
)

type SafePbft struct {
	viewID			uint
	pbftID 			uint
	state  			map[uint]uint
	addr   			string
	lastCheckpoint	uint

	// in memory storage
	logRequest       map[string]types.PbftRequest
	logPrePrepareMsg map[uint]string
	logPrepareMsg    map[string]map[uint]string
	logCommitMsg     map[string]map[uint]string
	logCheckpointMsg map[uint]map[string]string

	tlcs			 map[uint][]types.PbftTLCMessage
	consensus     	 map[uint]types.PbftRequest

	// private key for signature
	privateKeyRSA 	 []byte

    // 0 -> good node; 1 -> malicious node 
	role		uint

	lock sync.Mutex
}

func NewSafePbft(myAddr string, initID uint) *SafePbft {

	return &SafePbft{
		viewID:			  0,
		pbftID: 		  initID,
		state:  		  make(map[uint]uint),
		addr:   		  myAddr,
		lastCheckpoint:   0,

		logRequest:       make(map[string]types.PbftRequest),
		logPrePrepareMsg: make(map[uint]string),
		logPrepareMsg:    make(map[string]map[uint]string),
		logCommitMsg:     make(map[string]map[uint]string),
		logCheckpointMsg: make(map[uint]map[string]string),

		tlcs:           map[uint][]types.PbftTLCMessage{},
		consensus:      map[uint]types.PbftRequest{},

        role:			0,

		privateKeyRSA: getPrivateKey(myAddr),
	}
}

// handler for PrePrepare message
func (n *node) ExecPrePrepareMessage(msg types.Message, pkt transport.Packet) error {
	pp, ok := msg.(*types.PrePrepareMessage)
	if !ok {
		return xerrors.Errorf("message wrong type")
	}


	// ignore message sent by itself
	if pp.SrcAddr == n.myAddr {
		return nil
	}


	//fmt.Printf("%s receive PrePrepare\n", n.myAddr)

	// verifying digest
	if digest := getDigest(pp.RequestMessage); digest != pp.Digest {
		return xerrors.Errorf("digest fail")
	}


	// verifying signature
	currPublicKey := getPublicKey(pp.SrcAddr)
	digestByte, _ := hex.DecodeString(pp.Digest)
	if !VerifyRSA(digestByte, pp.Signature, currPublicKey) {
		fmt.Println("fail in signature verifying")
		return nil
	}


	n.pbft.lock.Lock()


	// verifying preprepare record
	tmpDigest, valid := n.pbft.logPrePrepareMsg[pp.PbftID]
	if valid && (tmpDigest != pp.Digest) {
		fmt.Println("n, d exist")
		return xerrors.Errorf("correspondance fail")
	}


	// store the preprepare message locally
	n.pbft.logPrePrepareMsg[pp.PbftID] = pp.Digest
	n.pbft.logRequest[pp.Digest] = pp.RequestMessage


	if _, ok = n.pbft.logPrepareMsg[n.pbft.addr]; !ok {
		n.pbft.logPrepareMsg[n.pbft.addr] = make(map[uint]string)
	}
	// store the prepare message locally
	n.pbft.logPrepareMsg[n.pbft.addr][pp.PbftID] = pp.Digest


	// sign the signature
	signature := signRSA(digestByte, n.pbft.privateKeyRSA)


	n.pbft.lock.Unlock()


	// broadcast prepare message
	pre := types.PrepareMessage{
		ViewID:	   pp.ViewID,	
		Digest:    pp.Digest,
		PbftID:    pp.PbftID,
		SrcAddr:   n.pbft.addr,
		Signature: signature,
	}

	if n.pbft.role == 1 {
		pre.PbftID = pre.PbftID + 1
	}

	transPre, err := n.MessageRegistry.MarshalMessage(pre)
	if err != nil {
		return err
	}


	n.Broadcast(transPre)
	//fmt.Printf("%s, Prepare broadcast finished\n", n.myAddr)
	return nil
}

// handler for Prepare message
func (n *node) ExecPrepareMessage(msg types.Message, pkt transport.Packet) error {
	pre, ok := msg.(*types.PrepareMessage)
	if !ok {
		return xerrors.Errorf("message wrong type")
	}


	// ignore message sent by itself
	if pre.SrcAddr == n.myAddr {
		return nil
	}


	// verifying signature
	currPublicKey := getPublicKey(pre.SrcAddr)
	digestByte, _ := hex.DecodeString(pre.Digest)
	if !VerifyRSA(digestByte, pre.Signature, currPublicKey) {
		fmt.Println("fail in signature verifying")
		return nil
	}

	
	//fmt.Printf("%s receive Prepare from %s\n", n.myAddr, pre.SrcAddr)
	n.pbft.lock.Lock()


	// store the prepare message locally
	if _, ok = n.pbft.logPrepareMsg[pre.SrcAddr]; !ok {
		n.pbft.logPrepareMsg[pre.SrcAddr] = make(map[uint]string)
	}
	n.pbft.logPrepareMsg[pre.SrcAddr][pre.PbftID] = pre.Digest


	// count the amount of prepare message corresponding to preprepare message
	count := 0
	for _, item := range n.pbft.logPrepareMsg {

		tmpDigest, ok := n.pbft.logPrePrepareMsg[pre.PbftID]
		if ok && (tmpDigest == item[pre.PbftID]) {
			count++
		}
	}


	// if 2f of prepare message received, state switches to prepared, broadcast commit message
	tmpState, valid := n.pbft.state[pre.PbftID]
	if uint(count) >= uint(n.TotalPeers/3*2) && (!valid || tmpState == 0) {

		n.pbft.state[pre.PbftID] = 1


		// store the commit message locally
		if _, ok = n.pbft.logCommitMsg[n.pbft.addr]; !ok {
			n.pbft.logCommitMsg[n.pbft.addr] = make(map[uint]string)
		}
		n.pbft.logCommitMsg[n.pbft.addr][pre.PbftID] = pre.Digest


		// sign the signare
		signature := signRSA(digestByte, n.pbft.privateKeyRSA)


		// broadcast the commit message
		c := types.CommitMessage{
			ViewID:	   pre.ViewID,	
			Digest:    pre.Digest,
			PbftID:    pre.PbftID,
			SrcAddr:   n.pbft.addr,
			Signature: signature,
		}

		if n.pbft.role == 1 {
			c.PbftID = c.PbftID + 1
		}


		transC, err := n.MessageRegistry.MarshalMessage(c)
		if err != nil {
			return err
		}


		n.Broadcast(transC)
		//fmt.Printf("2f of prepare reached, %s commit broadcast finished\n", n.myAddr)
	}


	n.pbft.lock.Unlock()

	return nil
}

// handler for Commit message
func (n *node) ExecCommitMessage(msg types.Message, pkt transport.Packet) error {
	c, ok := msg.(*types.CommitMessage)
	if !ok {
		return xerrors.Errorf("message wrong type")
	}


	// ignore message sent by itself
	if c.SrcAddr == n.myAddr {
		return nil
	}


	// verifying signature
	currPublicKey := getPublicKey(c.SrcAddr)
	digestByte, _ := hex.DecodeString(c.Digest)
	if !VerifyRSA(digestByte, c.Signature, currPublicKey) {
		fmt.Println("fail in signature verifying")
		return nil
	}


	//fmt.Printf("%s receive commit from %s\n", n.myAddr, c.SrcAddr)

	n.pbft.lock.Lock()


	// store the commit message locally
	if _, ok = n.pbft.logCommitMsg[c.SrcAddr]; !ok {
		n.pbft.logCommitMsg[c.SrcAddr] = make(map[uint]string)
	}
	n.pbft.logCommitMsg[c.SrcAddr][c.PbftID] = c.Digest


	// count the amount of commit message corresponding to preprepare message
	count := 0
	for _, item := range n.pbft.logCommitMsg {

		tmpDigest, valid := n.pbft.logPrePrepareMsg[c.PbftID]
		if valid && (tmpDigest == item[c.PbftID]) {
			count++
		}
	}

	// if 2f+1 of commit message received, state switches to commited, store the blockchain
	_, ok = n.pbft.state[c.PbftID]
	if uint(count) > uint(n.TotalPeers/3*2) && ok && n.pbft.state[c.PbftID] == 1 {

		n.pbft.state[c.PbftID] = 2

		// define the pbft block
		request := n.pbft.logRequest[c.Digest]
		pbftBlock := n.GetPbftBlock(request, c.PbftID/n.TotalPeers)


		// update block chain
		chain := n.HrStorage.GetHrBlockchainStore()
		blockBytes, err := pbftBlock.Marshal()
		if err != nil {
			return err
		}


		chain.Set(pbftBlock.Request.IDhr, hex.EncodeToString(pbftBlock.Hash), blockBytes)
		chain.Set(pbftBlock.Request.IDhr, storage.LastBlockKey, pbftBlock.Hash)
		// update namestore
		n.HrStorage.GetHrNamingStore().Set(pbftBlock.Request.IDhr, pbftBlock.Request.Filename, []byte(pbftBlock.Request.Metahash))
		//fmt.Printf("2f + 1 Commit reached, %s add to blockchain with hr ID: %s, file name: %s, hash: %s\n", n.myAddr, pbftBlock.Request.IDhr, pbftBlock.Request.Filename, pbftBlock.Request.Metahash)
	}


	n.pbft.lock.Unlock()

	return nil
}


// handler for Checkpoint message
func (n *node) ExecCheckpointMessage(msg types.Message, pkt transport.Packet) error {
	cp, ok := msg.(*types.CheckpointMessage)
	if !ok {
		return xerrors.Errorf("message wrong type")
	}


	// ignore message sent by itself
	if cp.SrcAddr == n.myAddr {
		return nil
	}


	//fmt.Printf("%s receive checkpoint\n", n.myAddr)

	// verifying signature
	currPublicKey := getPublicKey(cp.SrcAddr)
	digestByte, _ := hex.DecodeString(cp.Digest)
	if !VerifyRSA(digestByte, cp.Signature, currPublicKey) {
		fmt.Println("fail in signature verifying")
		return nil
	}


	n.pbft.lock.Lock()


    if _, ok = n.pbft.logCheckpointMsg[cp.PbftID]; !ok {
		n.pbft.logCheckpointMsg[cp.PbftID] = make(map[string]string)
	}
	// store the checkpoint message locally
	n.pbft.logCheckpointMsg[cp.PbftID][cp.SrcAddr] = cp.Digest


    // count the amount of checkpoint message corresponding to the pbftID
	count := len(n.pbft.logCheckpointMsg[cp.PbftID])
	if uint(count) >= uint(n.TotalPeers/3*2+1) {
        n.pbft.lastCheckpoint = cp.PbftID
	}


	n.pbft.lock.Unlock()

	return nil
}

// get a PBFT block
func (n *node) GetPbftBlock(request types.PbftRequest, index uint) types.PbftchainBlock {
	prevHash := n.HrStorage.GetHrBlockchainStore().Get(request.IDhr, storage.LastBlockKey)
	if prevHash == nil {
		prevHash = make([]byte, 32)
	}


	hashBuf := bytes.Buffer{}
	// error will always be nil, no need to handle
	hashBuf.WriteString(strconv.FormatUint(uint64(index), 10))
	hashBuf.WriteString(request.IDhr)
	hashBuf.WriteString(request.Filename)
	hashBuf.WriteString(request.Metahash)
	hashBuf.Write(prevHash)


	hash := SHA256(hashBuf.Bytes())


	return types.PbftchainBlock{
		Index:    index,
		Hash:     hash,
		Request:  request,
		PrevHash: prevHash,
	}
}

// assign a new pbft id
func (p *SafePbft) AssignPbftID(increase uint) {
	p.pbftID = p.pbftID + increase
	p.state[p.pbftID] = 0
}

// compute the digest of a pbft request
func getDigest(request types.PbftRequest) string {

	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}


	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

// load the public key according to an address
func getPublicKey(addr string) []byte {

	pubKey, err := ioutil.ReadFile("rsa_keys/" + addr + "/" + addr + "_rsa_pub")
	if err != nil {
		log.Panic(err)
	}


	return pubKey
}

// load the private key according to an address
func getPrivateKey(addr string) []byte {

	privKey, err := ioutil.ReadFile("rsa_keys/" + addr + "/" + addr + "_rsa_pri")
	if err != nil {
		log.Panic(err)
	}


	return privKey
}

// turn to malicious node
func (n *node) TurnBad() {
	n.pbft.lock.Lock()
	n.pbft.role = 1 
	n.pbft.lock.Unlock()
}


// get current pbftID
func (p *SafePbft) GetStep() uint {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.pbftID
}

// broadcast the PBFT TLC message
func (n *node) BroadcastPbftTLC(step uint, pbftBlock types.PbftchainBlock) error {

	PbftTLCMsg := types.PbftTLCMessage{
		Step:  step,
		Block: pbftBlock,
	}


	transPbftTLCMsg, err := n.MessageRegistry.MarshalMessage(PbftTLCMsg)
	if err != nil {
		return err
	}


	go func() {
		err := n.Broadcast(transPbftTLCMsg)
		if err != nil {
			n.BroadcastError()
		}
	}()


	return nil
}

// handler for PbftTLC message
func (n *node) ExecPbftTLCMessage(msg types.Message, pkt transport.Packet) error {

	tlc, ok := msg.(*types.PbftTLCMessage)
	if !ok {
		return xerrors.Errorf("message wrong type: %T, we expect PbftTLCMessage", msg)
	}


	// 1. update receied tlcs
	threshold := n.PaxosThreshold(n.TotalPeers)
	n.pbft.ReceivePbftTLC(tlc, uint(threshold))


	// 2. recursively update according to tcls
	return n.PbftTLCProcess(n.pbft.GetStep(), false)
}

// TLC counting
func (p *SafePbft) ReceivePbftTLC(tlc *types.PbftTLCMessage, threshold uint) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if tlc.Step >= p.pbftID {

		p.tlcs[tlc.Step] = append(p.tlcs[tlc.Step], *tlc)

		_, ok := p.consensus[tlc.Step]
		if len(p.tlcs[tlc.Step]) >= int(threshold) && !ok{
			p.consensus[tlc.Step] = tlc.Block.Request
		}
	}
}

// TLC catch up
func (n *node) PbftTLCProcess(step uint, catchUp bool) error {

	block, ok := n.pbft.GetPbftBlockAndAdvance(step)
	if !ok {
		return nil
	}

	// 1. update block chain
	chain := n.Storage.GetBlockchainStore()
	blockBytes, err := block.Marshal()
	if err != nil {
		return err
	}

	chain.Set(hex.EncodeToString(block.Hash), blockBytes)
	chain.Set(storage.LastBlockKey, block.Hash)

	// 2. update namestore
	n.Storage.GetNamingStore().Set(block.Request.Filename, []byte(block.Request.Metahash))

	fmt.Printf("%s add to blockchain with hr ID: %s, file name: %s, hash: %s\n", n.myAddr, block.Request.IDhr, block.Request.Filename, block.Request.Metahash)

	if !(catchUp) {

		err = n.BroadcastPbftTLC(step, block)
		if err != nil {
			return err
		}
	}

	return n.PbftTLCProcess(step+1, true)
}

// advance the step
func (p *SafePbft) GetPbftBlockAndAdvance(step uint) (types.PbftchainBlock, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// 0. we ignore the messages that do not belong to current step
	if step != p.pbftID {
		return types.PbftchainBlock{}, false
	}

	_, ok := p.consensus[step]
	if !ok {
		return types.PbftchainBlock{}, false
	}

	// 1. it means that we have consensus and we can move on
	p.pbftID++

	// 2. return blocks in tlc
	return p.tlcs[step][0].Block, true

}