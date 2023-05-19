package impl

import "go.dedis.ch/cs438/types"

func (a *Applicant) recvPK(key []byte) {
	a.keyMux.Lock()
	defer a.keyMux.Unlock()

	if a.hasPubKey {
		return
	}
	a.pubKey.UnmarshalBinary(key)
	a.hasPubKey = true
}

func (a *Applicant) sendEncryptedChunkMessage(key string, chunk []byte) {
	chunkMsg := types.EncryptedChunkMessage{
		ChunkKey: key,
		Chunk:    chunk,
	}
	transportMsg, _ := a.node.MessageRegistry.MarshalMessage(&chunkMsg)
	a.node.Broadcast(transportMsg)
}
