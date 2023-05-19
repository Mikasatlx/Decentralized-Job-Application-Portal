package impl

import (
	"crypto"
	"errors"
	"math/rand"
	"strings"

	"github.com/rs/xid"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/types"
	"golang.org/x/xerrors"
)

func SHA256(buf []byte) []byte {
	sha256 := crypto.SHA256.New()
	sha256.Write(buf)
	return sha256.Sum(nil)
}

func (n *node) SendDataRequest(msg *types.DataRequestMessage, dst string) error {
	neighbor, ok := n.routingTable.Get(dst)
	if !ok {
		return errors.New("SendDataRequest could not find neighbor")
	}
	transportDataRequest, err := n.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return err
	}
	return n.SendMsg(dst, neighbor, transportDataRequest)
}

func (n *node) SendDataReply(msg *types.DataReplyMessage, dst string) error {
	neighbor, ok := n.routingTable.Get(dst)
	if !ok {
		return errors.New("SendDataReply could not find neighbor")
	}
	transportDataReply, err := n.MessageRegistry.MarshalMessage(msg)
	if err != nil {
		return err
	}
	return n.SendMsg(dst, neighbor, transportDataReply)
}

func (n *node) FetchFile(hash string) ([]byte, error) {
	store := n.Storage.GetDataBlobStore()
	file := store.Get(hash)
	if file == nil {
		dst, err := n.catalog.GetRandPeer(hash)
		if err != nil {
			return nil, err
		}
		requestID := xid.New().String()
		dataRequestMsg := types.DataRequestMessage{
			RequestID: requestID,
			Key:       hash,
		}
		c := n.channelTable.Set(requestID)
		defer func() {
			n.channelTable.Delete(requestID)
		}()
		err = n.SendDataRequest(&dataRequestMsg, dst)
		if err != nil {
			return nil, err
		}
		backoff := n.Configuration.BackoffDataRequest
		n.dataRequestQueue.Push(&dataRequestMsg, dst, requestID, &backoff)
		msg, ok := <-c
		if !ok {
			return nil, errors.New("the channel is closed, retry fail")
		}
		dataReplyMsg, ok := msg.(*types.DataReplyMessage)
		if !ok {
			return nil, xerrors.Errorf("message wrong type: %T, we expect DataReplyMessage", msg)
		}
		file = dataReplyMsg.Value
	}
	return file, nil
}

func (n *node) GetNeighborsAndBudgets(without string, budget uint) map[string]uint {
	routingTable := n.routingTable.GetAll()
	neighbors := []string{}
	for key, val := range routingTable {
		if key == val && n.myAddr != key && key != without {
			neighbors = append(neighbors, key)
		}
	}
	if len(neighbors) == 0 {
		return nil
	}
	rand.Shuffle(len(neighbors), func(i, j int) { neighbors[i], neighbors[j] = neighbors[j], neighbors[i] })
	res := map[string]uint{}
	div := budget / uint(len(neighbors))
	mod := budget % uint(len(neighbors))
	if div > 0 {
		for i := 0; i < len(neighbors); i++ {
			res[neighbors[i]] = div
			if mod > 0 {
				res[neighbors[i]]++
				mod--
			}
		}
		return res
	}
	for i := 0; i < int(mod); i++ {
		res[neighbors[i]] = 1
	}
	return res
}

func (n *node) SearchFile(msg *types.SearchRequestMessage, prevNeighbor string) (bool, error) {
	neighbors := n.GetNeighborsAndBudgets(prevNeighbor, msg.Budget)
	if len(neighbors) == 0 {
		return false, nil
	}

	for neighbor, budget := range neighbors {
		msg := types.SearchRequestMessage{
			RequestID: msg.RequestID,
			Origin:    msg.Origin,
			Pattern:   msg.Pattern,
			Budget:    budget,
		}
		transportMsg, err := n.MessageRegistry.MarshalMessage(msg)
		if err != nil {
			return false, err
		}
		err = n.SendMsg(neighbor, neighbor, transportMsg)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (n *node) GetAllChunkHash(metahash []byte) ([][]byte, bool) {
	store := n.Storage.GetDataBlobStore()
	fileChunks := store.Get(string(metahash))
	if fileChunks != nil {
		chunkKeys := strings.Split(string(fileChunks), peer.MetafileSep)
		chunks := [][]byte{}
		for _, chunkKey := range chunkKeys {
			chunk := store.Get(chunkKey)
			if chunk != nil {
				chunks = append(chunks, []byte(chunkKey))
			} else {
				chunks = append(chunks, nil)
			}
		}
		return chunks, true
	}
	return nil, false
}

func (n *node) IsFullMatch(metahash []byte) bool {
	store := n.Storage.GetDataBlobStore()
	fileChunks := store.Get(string(metahash))
	if fileChunks == nil {
		return false
	}
	chunkKeys := strings.Split(string(fileChunks), peer.MetafileSep)
	for _, chunkKey := range chunkKeys {
		chunk := store.Get(chunkKey)
		if chunk == nil {
			return false
		}
	}
	return true
}

func (n *node) GetFullMatch(searchReplyMsg *types.SearchReplyMessage) (string, bool) {
	for _, info := range searchReplyMsg.Responses {
		fullMatch := true
		for _, chunkHash := range info.Chunks {
			if chunkHash == nil {
				fullMatch = false
				break
			}
		}
		if fullMatch {
			return info.Name, true
		}
	}
	return "", false
}
