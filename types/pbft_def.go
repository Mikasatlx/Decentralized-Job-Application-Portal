package types

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)


// PBFT request
type PbftRequest struct {
	IDhr		string
	Filename	string
	Metahash	string
}


// PBFT pre-prepare message 
type PrePrepareMessage struct {
	ViewID		   uint
	RequestMessage PbftRequest
	Digest         string
	PbftID    	   uint
	SrcAddr		   string
	Signature      []byte
}


// PBFT prepare message
type PrepareMessage struct {
	ViewID	   uint
	Digest     string
	PbftID     uint
	SrcAddr    string
	Signature  []byte
}


// PBFT commit message
type CommitMessage struct {
	ViewID	   uint
	Digest     string
	PbftID     uint
	SrcAddr    string
	Signature  []byte
}


// PBFT checkpoint message
type CheckpointMessage struct {
	Digest     string
	PbftID     uint
	SrcAddr    string
	Signature  []byte
}


// BlockchainBlock defines the content of a block in the blockchain.
type PbftchainBlock struct {
	// Index is the index of the block in the blockchain, starting at 0 for the
	// first block.
	Index uint

	// Hash is SHA256(Index || Request.IDhr || Request.Filename || Request.Metahash || Prevhash)
	// use crypto/sha256
	Hash []byte

	Request PbftRequest

	// PrevHash is the SHA256 hash of the previous block
	PrevHash []byte
}


// TLCMessage defines a TLC message
//
// - implements types.Message
// - implemented in HW3
type PbftTLCMessage struct {
	Step  uint
	Block PbftchainBlock
}


// Marshal marshals the BlobkchainBlock into a byte representation. Must be used
// to store blocks in the blockchain store.
func (p *PbftchainBlock) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// Unmarshal unmarshals the data into the current instance. To unmarshal a
// block:
//
//	var block BlockchainBlock
//	err := block.Unmarshal(buf)
func (p *PbftchainBlock) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}


// DisplayBlock writes a rich string representation of a block
func (p PbftchainBlock) DisplayPbftBlock(out io.Writer) {
	crop := func(s string) string {
		if len(s) > 10 {
			return s[:8] + "..."
		}
		return s
	}

	max := func(s ...string) int {
		max := 0
		for _, se := range s {
			if len(se) > max {
				max = len(se)
			}
		}
		return max
	}

	pad := func(n int, s ...*string) {
		for _, se := range s {
			*se = fmt.Sprintf("%-*s", n, *se)
		}
	}

	row1 := fmt.Sprintf("%d | %x", p.Index, p.Hash[:6])
	row2 := fmt.Sprintf("I | %s", crop(p.Request.IDhr))
	row3 := fmt.Sprintf("F | %s", crop(p.Request.Filename))
	row4 := fmt.Sprintf("M | %s", crop(p.Request.Metahash))
	row5 := fmt.Sprintf("<- %x", p.PrevHash[:6])

	m := max(row1, row2, row3, row4, row5)
	pad(m, &row1, &row2, &row3, &row4, &row5)

	fmt.Fprintf(out, "\n┌%s┐\n", strings.Repeat("─", m+2))
	fmt.Fprintf(out, "│ %s │\n", row1)
	fmt.Fprintf(out, "│%s│\n", strings.Repeat("─", m+2))
	fmt.Fprintf(out, "│ %s │\n", row2)
	fmt.Fprintf(out, "│ %s │\n", row3)
	fmt.Fprintf(out, "│ %s │\n", row4)
	fmt.Fprintf(out, "│%s│\n", strings.Repeat("─", m+2))
	fmt.Fprintf(out, "│ %s │\n", row5)
	fmt.Fprintf(out, "└%s┘\n", strings.Repeat("─", m+2))
}
