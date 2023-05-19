package impl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

func NewHR(n *node, id uint, secret string, num uint) *HR {

	p := suite.Scalar().Pick(suite.RandomStream())
	Q := suite.Point().Mul(p, nil) // pG
	QBuf, _ := Q.MarshalBinary()
	return &HR{
		node:          n,
		id:            id,
		priKey:        p,
		Q:             QBuf,
		privateKeyRSA: getPrivateKey(n.myAddr),
		secret:        secret,
		shareTable:    NewSafeShareTable(num),
		num:           num,
	}
}

type HR struct {
	node        *node
	id          uint
	secret      string
	priKey      kyber.Scalar
	Q           []byte
	C           []byte // C from patient
	A           []byte // pubKey of cothority
	ACMux       sync.RWMutex
	shareTable  *SafeShareTable
	metaFileKey string
	keyMux      sync.RWMutex
	// private key for signature
	privateKeyRSA []byte
	// for indicating which node to serve
	num uint
}

func (h *HR) Download(metaFileKey string) ([]byte, error) {
	fmt.Println("Enter HR Download", metaFileKey, "h")

	// 1. request partial decryptions of decrypted key
	fmt.Println("1. request partial decryptions of decrypted key")
	h.keyMux.Lock()
	h.metaFileKey = metaFileKey
	h.keyMux.Unlock()

	h.RequestFile(metaFileKey)

	// 2. collect partial dcryptions to reconstruct decryption key
	fmt.Println("2. collect partial dcryptions to reconstruct decryption key")
	ok := false
	var shares []*share.PubShare
	for !ok {
		shares, ok = h.shareTable.Get()
		if !ok {
			time.Sleep(time.Second)
		}
	}
	decKey := h.GetDecKey(shares)
	fmt.Println("decKey", decKey)
	// 3. decrypt file information and reconstruct file
	fmt.Println("3. decrypt file information and reconstruct file", metaFileKey, "h")
	file, err := h.ReconstructFile(metaFileKey, decKey)

	// 4. do some cleaning
	fmt.Println("4. do some cleaning")
	h.keyMux.Lock()
	h.metaFileKey = ""
	h.keyMux.Unlock()
	h.shareTable.Clean()
	return file, err
}

func (h *HR) ReconstructFile(metaFileKey string, decKey []byte) ([]byte, error) {
	infoChunk, err := h.FetchChunk(metaFileKey, decKey)
	if err != nil {
		return nil, err
	}

	info := &types.AppFileInfo{}
	err = json.Unmarshal(infoChunk, info)
	if err != nil {
		fmt.Println("ReconstructFile json.Unmarshal", err)
	}
	metafile, err := h.FetchChunk(info.EntryKey, decKey)
	if err != nil {
		return nil, err
	}
	chunkKeys := strings.Split(string(metafile), peer.MetafileSep)
	file := bytes.Buffer{}
	for _, chunkKey := range chunkKeys {
		chunk, err := h.FetchChunk(chunkKey, decKey)
		if err != nil {
			return nil, err
		}
		file.Write(chunk)
	}
	return file.Bytes(), nil
}

func (h *HR) FetchChunk(metaHash string, key []byte) ([]byte, error) {
	chunk, err := h.node.FetchFile(metaHash)
	// we need to copy
	copyChunk := make([]byte, len(chunk))
	copy(copyChunk, chunk)
	if err != nil {
		fmt.Println("h.node.FetchFile", err)
		return nil, err
	}
	// the following implemnetation is in place!
	return AESDecrypt(key, copyChunk), nil
}

func (h *HR) GetDecKey(shares []*share.PubShare) []byte {
	// the following is referenced from https://github.com/dedis/kyber/blob/master/examples/dkg_test.go#L227

	R, err := share.RecoverCommit(suite, shares, h.shareTable.threshold, len(shares))
	if err != nil {
		fmt.Println("share.RecoverCommit", err)
	}
	C := suite.Point()
	C.UnmarshalBinary(h.C)

	A := suite.Point()
	A.UnmarshalBinary(h.A)
	decryptedPoint := suite.Point().Sub( // C - (R - pA)
		C,
		suite.Point().Sub( // R - pA
			R,
			suite.Point().Mul(h.priKey, A), // pA
		),
	)
	decKey, err := decryptedPoint.Data()
	if err != nil {
		fmt.Println("decryptedPoint.Data", err)
	}
	return decKey
}
func (h *HR) RequestFile(metaFileKey string) {
	// this is based on HMAC
	reqMsg := types.RequestFileMessage{
		MetaFileID:   metaFileKey,
		PubKey:       h.Q,
		Address:      h.node.myAddr,
		MAC:          nil,
		ServiceIndex: uint(rand.Intn(int(h.num))),
		HRID:         fmt.Sprint(h.id),
	}

	if signMethod == 1 {
		// calculate HMAC
		reqMsg.MAC = GetHMAC(reqMsg, h.secret)
		transreqMsg, _ := h.node.MessageRegistry.MarshalMessage(&reqMsg)
		h.node.Broadcast(transreqMsg)
	} else {
		// sign with rsa
		reqBytes, _ := json.Marshal(reqMsg)
		reqMsg.MAC = signRSA(reqBytes, h.privateKeyRSA)
		transreqMsg, _ := h.node.MessageRegistry.MarshalMessage(&reqMsg)
		h.node.Broadcast(transreqMsg)
	}

}
