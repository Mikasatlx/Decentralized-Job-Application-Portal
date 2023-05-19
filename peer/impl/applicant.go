package impl

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"go.dedis.ch/kyber/v3"
)

func NewApplicant(n *node) *Applicant {
	return &Applicant{
		node:      n,
		pubKey:    suite.Point().Null(), // store the public key of cothority nodes
		hasPubKey: false,
	}
}

type Applicant struct {
	node      *node
	pubKey    kyber.Point
	keyMux    sync.RWMutex
	hasPubKey bool
}

func (a *Applicant) RequestPK() {
	pkReqMsg := types.PKRequestMessage{
		Address: a.node.myAddr,
	}
	transportPkReqMsg, _ := a.node.MessageRegistry.MarshalMessage(&pkReqMsg)
	a.node.Broadcast(transportPkReqMsg)
}

func (a *Applicant) Upload(data io.Reader) (metahash string, err error) {
	a.keyMux.RLock()
	if !a.hasPubKey {
		go a.RequestPK()
	}
	a.keyMux.RUnlock()

	buf, err := io.ReadAll(data)
	dataLen := len(buf)
	if err != nil {
		return "", err
	}
	metafileKeyBuf := bytes.Buffer{}
	metafileValueBuf := bytes.Buffer{}
	chunkSize := int(a.node.ChunkSize)
	curSize := 0
	bufPtr := 0

	key := make([]byte, 24) //AES-192
	rand.Read(key)

	// 1. split file content into chunks
	for dataLen > 0 {
		if dataLen > chunkSize {
			dataLen -= chunkSize
			curSize = chunkSize
		} else {
			curSize = dataLen
			dataLen = 0
		}

		chunk := make([]byte, curSize)
		copy(chunk, buf[bufPtr:bufPtr+curSize])
		bufPtr += curSize

		// encryption happens here
		enChunk := AESEncrypt(key, chunk)

		chunkSHA256 := SHA256(enChunk)
		_, err = metafileKeyBuf.Write(chunkSHA256)
		if err != nil {
			return "", err
		}
		chunkSHA256Str := hex.EncodeToString(chunkSHA256)
		_, err = metafileValueBuf.WriteString(chunkSHA256Str)
		if err != nil {
			return "", err
		}
		if dataLen > 0 {
			_, err = metafileValueBuf.WriteString(peer.MetafileSep)
			if err != nil {
				return "", err
			}
		}
		a.sendEncryptedChunkMessage(chunkSHA256Str, enChunk)
	}
	metafileValue := metafileValueBuf.Bytes()
	metafileKey := hex.EncodeToString(SHA256(metafileKeyBuf.Bytes()))

	enChunk := AESEncrypt(key, metafileValue)

	//2. put the information of the file into the first chunk
	info := types.AppFileInfo{
		UserName: a.node.myAddr,
		FileName: "UnKnown",
		EntryKey: metafileKey,
	}
	infoByte, _ := json.Marshal(info)
	enInfo := AESEncrypt(key, infoByte)

	ID := hex.EncodeToString(SHA256(enInfo))

	a.sendEncryptedChunkMessage(ID, enInfo)

	a.sendEncryptedChunkMessage(metafileKey, enChunk)

	msg := types.EncryptedFileMessage{
		ID: ID,
	}

	//we need to ensure that the file info must be delivered by cothority nodes
	a.BroadcastEncryptedFileMessage(msg, key)

	return ID, nil
}

func (a *Applicant) BroadcastEncryptedFileMessage(msg types.EncryptedFileMessage, enKey []byte) {

	fmt.Println("enter BroadcastEncryptedMessage")
	ok := false
	for !ok {
		a.keyMux.RLock()
		ok = a.hasPubKey
		a.keyMux.RUnlock()
		if !ok {
			time.Sleep(time.Second)
		}
	}
	fmt.Println("BroadcastEncryptedMessage finish collecting public key")
	// follow the guidance in https://github.com/dedis/kyber/blob/master/examples/dkg_test.go#L167
	// (1) Message encryption:
	//
	// r: random point
	// A: dkg public key
	// G: curve's generator
	// M: message to encrypt
	// (C, U): encrypted message
	//
	// C = rA + M
	// U = rG

	// we would send C and U
	a.keyMux.RLock()
	A := a.pubKey
	a.keyMux.RUnlock()

	r := suite.Scalar().Pick(suite.RandomStream())
	M := suite.Point().Embed(enKey, suite.RandomStream())
	C := suite.Point().Add( // rA + M
		suite.Point().Mul(r, A), // rA
		M,
	)
	U := suite.Point().Mul(r, nil) // rG
	CBuf, _ := C.MarshalBinary()
	msg.C = CBuf
	UBuf, _ := U.MarshalBinary()
	msg.U = UBuf

	transportMsg, _ := a.node.MessageRegistry.MarshalMessage(&msg)

	err := a.node.Broadcast(transportMsg)
	for err != nil {
		err = a.node.Broadcast(transportMsg)
	}
	fmt.Println("leave BroadcastEncryptedMessage")
}
