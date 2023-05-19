package types

import (
	dkg "go.dedis.ch/kyber/v3/share/dkg/pedersen"
)

// -----------------------project--------------------------(top)
// PubKeyMessage is a message to exchange public keys among cothority nodes

type PubKeyMessage struct {
	Index  uint
	PubKey []byte
}

type DealMessage struct {
	Deal dkg.Deal
}

type DealResponseMessage struct {
	Index uint
	Res   *dkg.Response
}

type PKRequestMessage struct {
	Address string
}

type PKResponseMessage struct {
	PubKey []byte
}
type AppFileInfo struct {
	UserName string
	FileName string
	EntryKey string
}
type EncryptedFileMessage struct {
	ID string
	C  []byte
	U  []byte
}

type EncryptedChunkMessage struct {
	ChunkKey string
	Chunk    []byte
}

type RequestFileMessage struct {
	HRID         string
	Address      string
	MetaFileID   string
	PubKey       []byte
	MAC          []byte
	ServiceIndex uint
}

type ResponseFileMessage struct {
	MetaFileID string
	I          uint
	V          []byte
	A          []byte
	C          []byte
}

// the following structure will recorded via consensus
type RequestRecord struct {
	Index     uint   `json:"Service Node"` // the ID of the cothority node that handles the request
	SeenTime  string `json:"Handle time"`  // the time that the cothority node views the request
	HRID      string `json:"HR ID"`
	HRAddress string `json:"HR addr"`
}

// -----------------------project--------------------------(down)
