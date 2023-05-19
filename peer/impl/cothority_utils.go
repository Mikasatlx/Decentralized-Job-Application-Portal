package impl

import (
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.dedis.ch/cs438/types"
	dkg "go.dedis.ch/kyber/v3/share/dkg/pedersen"
)

func (n *CothorityNode) recvPubKey(m *types.PubKeyMessage) {
	n.dkgDoneMux.RLock()
	ok := n.dkgDone
	n.dkgDoneMux.RUnlock()
	if ok {
		return
	}

	pk := suite.Point().Null()
	pk.UnmarshalBinary(m.PubKey)
	n.pubKeyTable.Set(m.Index, pk)
	pubKeys, ok := n.pubKeyTable.GetKeys()
	if !ok {
		return
	}
	go n.initDKG(pubKeys)
}

func (n *CothorityNode) recvDeal(m *types.DealMessage) {
	n.dkgDoneMux.RLock()
	ok := n.dkgDone
	n.dkgDoneMux.RUnlock()
	if ok {
		return
	}

	ok = n.dealTable.Set(uint(m.Deal.Index))
	if !ok {
		return
	}

	ok = false
	for !ok {
		n.dkgMux.Lock()
		if n.dkg != nil {
			res, _ := n.dkg.ProcessDeal(&m.Deal)
			go n.broadcastDealResponse(res)
			ok = true
		}
		n.dkgMux.Unlock()
		if !ok {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (n *CothorityNode) broadcastDealResponse(res *dkg.Response) {
	resMsg := types.DealResponseMessage{
		Index: n.index,
		Res:   res,
	}
	transportResMsg, _ := n.node.MessageRegistry.MarshalMessage(&resMsg)
	n.node.Broadcast(transportResMsg)
}

func (n *CothorityNode) recvDealResponse(msg *types.DealResponseMessage) {
	n.dkgDoneMux.RLock()
	ok := n.dkgDone
	n.dkgDoneMux.RUnlock()
	if ok {
		return
	}
	ok = n.resTable.Set(msg.Index + uint(len(n.addresses))*(1+uint(msg.Res.Index)))
	if !ok {
		return
	}
	ok = false
	for !ok {
		var err error
		n.dkgMux.Lock()
		if n.dkg != nil {
			_, err = n.dkg.ProcessResponse(msg.Res)
		}
		n.dkgMux.Unlock()
		if err == nil {
			ok = true
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (n *CothorityNode) recvPKRequestMessage(msg *types.PKRequestMessage) {
	ok := false
	for !ok {
		n.dkgDoneMux.RLock()
		ok = n.dkgDone
		n.dkgDoneMux.RUnlock()
		if !ok {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	pkBuf, _ := n.pubKey.MarshalBinary()
	pkRes := types.PKResponseMessage{
		PubKey: pkBuf,
	}

	n.node.SendPrivateMsg(&pkRes, msg.Address)
}

func (n *CothorityNode) recvEncryptedFileMessage(msg *types.EncryptedFileMessage) {
	// type EncryptedFileMessage struct {
	// 	ID          string
	// 	C           []byte
	// 	U           []byte
	// }
	data := Data{
		C: msg.C,
		U: msg.U,
	}
	fmt.Println(n.addresses, "recvEncryptedFileMessage", msg.ID)
	n.dataTableMux.Lock()
	n.dataTable[msg.ID] = data
	n.dataTableMux.Unlock()
}

func (n *CothorityNode) recvEncryptedChunkMessage(msg *types.EncryptedChunkMessage) {
	store := n.node.Storage.GetDataBlobStore()
	store.Set(msg.ChunkKey, msg.Chunk)
	fmt.Println(store.Len())
}

func (n *CothorityNode) recvRequestFileMessage(msg *types.RequestFileMessage) {

	// 0. check signature
	if signMethod == 1 {
		id, _ := strconv.Atoi(msg.HRID)
		HRSecret, ok := n.HRSecretTable[uint(id)]
		if !ok {
			fmt.Println(msg.HRID, "is not a HR")
			return
		}
		messageMAC := msg.MAC
		msg.MAC = nil
		expectedMAC := GetHMAC(*msg, HRSecret)
		if !hmac.Equal(messageMAC, expectedMAC) {
			fmt.Println("HR", msg.HRID, "validated fail")
			return
		}
	} else {
		messageMAC := msg.MAC
		msg.MAC = nil
		reqBytes, _ := json.Marshal(msg)
		if !VerifyRSA(reqBytes, messageMAC, getPublicKey(msg.Address)) {
			fmt.Println("HR", msg.HRID, "validated fail")
			return
		}
	}

	// 1. build up the record that is ready to propose in consensus
	record := types.RequestRecord{
		Index:     n.index,
		SeenTime:  time.Now().String(),
		HRID:      msg.HRID,
		HRAddress: msg.Address,
	}

	// 2. we need to use consensus to record the request, if we are the indicated service index
	if msg.ServiceIndex == n.index {
		recordBytes, _ := json.Marshal(record)
		// first test not encrypted
		n.node.Tag(hex.EncodeToString(recordBytes), msg.MetaFileID)
		consensusStr := ""
		consensus := n.node.GetBlockchain()
		for _, c := range consensus {
			cStr := c.String()
			consensusStr += cStr + "; "
		}
		fmt.Println("consensus: ", consensusStr)
		// do we need to encrypt the request?
		// then test encrypted
		// enRecord, _ := RsaEncrypt(recordBytes, n.recordPubKey)
		// n.node.Tag(hex.EncodeToString(enRecord), "")
	}

	// 3. send back partial decryption
	Q := suite.Point().Null()
	Q.UnmarshalBinary(msg.PubKey)

	n.dataTableMux.RLock()
	ok := false
	var data Data
	for !ok {
		// TODO: we may need to search data
		data, ok = n.dataTable[msg.MetaFileID]
		if !ok {
			fmt.Println("dataTable", msg.MetaFileID, "Not Found")
			time.Sleep(time.Second)
		}
	}
	fmt.Println("dataTable", msg.MetaFileID, "Found")
	n.dataTableMux.RUnlock()

	UPoint := suite.Point().Null()
	println("n", n, "n.priShare", n.priShare, "n.priShare.V", n.priShare.V)
	UPoint.UnmarshalBinary(data.U)
	v := suite.Point().Add( // oU + oQ
		suite.Point().Mul(n.priShare.V, UPoint), // oU
		suite.Point().Mul(n.priShare.V, Q),      // oQ
	)
	// 	[signal SIGSEGV: segmentation violation code=0x1 addr=0x8 pc=0x79fdff]
	// at "suite.Point().Mul(n.priShare.V, UPoint), // oU"
	// goroutine 2464 [running]:
	// go.dedis.ch/cs438/peer/impl.(*CothorityNode).recvRequestFileMessage(0xc0001900f0, 0xc000c1baa0)
	//         /home/shi/cs438-2022-project/peer/impl/cothority_utils.go:178 +0x87f
	// created by go.dedis.ch/cs438/peer/impl.(*node).ExecRequestFileMessage
	//         /home/shi/cs438-2022-project/peer/impl/message_handler.go:595 +0x17c
	vBuf, _ := v.MarshalBinary()
	A := n.pubKey
	ABuf, _ := A.MarshalBinary()

	resMsg := types.ResponseFileMessage{
		I:          n.index,
		V:          vBuf,
		A:          ABuf,
		C:          data.C,
		MetaFileID: msg.MetaFileID,
	}
	n.node.SendPrivateMsg(&resMsg, msg.Address)
}
