package types

import (
	"fmt"
)

// -----------------------project--------------------------(top)

// PubKeyMessage is a message to exchange public keys among cothority nodes

func (p PubKeyMessage) NewEmpty() Message {
	return &PubKeyMessage{}
}

// Name implements types.Message.
func (p PubKeyMessage) Name() string {
	return "pubkey"
}

// String implements types.Message.
func (p PubKeyMessage) String() string {
	return fmt.Sprintf("<%T> <%s>", p.Index, p.PubKey)
}

// HTML implements types.Message.
func (p PubKeyMessage) HTML() string {
	return p.String()
}

// DealMessage-------------------------------------

func (DealMessage) NewEmpty() Message {
	return &DealMessage{}
}

// Name implements types.Message.
func (DealMessage) Name() string {
	return "deal"
}

// String implements types.Message.
func (d DealMessage) String() string {
	buf, _ := d.Deal.MarshalBinary()
	return fmt.Sprintf("<%s>", string(buf))
}

// HTML implements types.Message.
func (d DealMessage) HTML() string {
	return d.String()
}

// DealResponseMessage-------------------------------------

func (DealResponseMessage) NewEmpty() Message {
	return &DealResponseMessage{}
}

// Name implements types.Message.
func (DealResponseMessage) Name() string {
	return "dealres"
}

// String implements types.Message.
func (d DealResponseMessage) String() string {
	return fmt.Sprintf("for <%T>, signature: <%s>", d.Res.Index, d.Res.Response.Signature)
}

// HTML implements types.Message.
func (d DealResponseMessage) HTML() string {
	return d.String()
}

// PKRequestMessage-------------------------------------

func (PKRequestMessage) NewEmpty() Message {
	return &PKRequestMessage{}
}

// Name implements types.Message.
func (PKRequestMessage) Name() string {
	return "pkrequest"
}

// String implements types.Message.
func (p PKRequestMessage) String() string {
	return fmt.Sprintf("PKRequestMessage from %T", p.Address)
}

// HTML implements types.Message.
func (p PKRequestMessage) HTML() string {
	return p.String()
}

// PKResponseMessage-------------------------------------

func (PKResponseMessage) NewEmpty() Message {
	return &PKResponseMessage{}
}

// Name implements types.Message.
func (PKResponseMessage) Name() string {
	return "pkresponse"
}

// String implements types.Message.
func (p PKResponseMessage) String() string {
	return fmt.Sprintf("PKResponseMessage: %T", p.PubKey)
}

// HTML implements types.Message.
func (p PKResponseMessage) HTML() string {
	return p.String()
}

// EncryptedFileMessage-------------------------------------

func (EncryptedFileMessage) NewEmpty() Message {
	return &EncryptedFileMessage{}
}

// Name implements types.Message.
func (EncryptedFileMessage) Name() string {
	return "enfile"
}

// String implements types.Message.
func (e EncryptedFileMessage) String() string {
	return fmt.Sprintf("EncryptedFileMessage key: %T", e.ID)
}

// HTML implements types.Message.
func (e EncryptedFileMessage) HTML() string {
	return e.String()
}

// EncryptedFileMessage-------------------------------------

func (EncryptedChunkMessage) NewEmpty() Message {
	return &EncryptedChunkMessage{}
}

// Name implements types.Message.
func (EncryptedChunkMessage) Name() string {
	return "enchunk"
}

// String implements types.Message.
func (e EncryptedChunkMessage) String() string {
	return fmt.Sprintf("EncryptedChunkMessage key: %T", e.ChunkKey)
}

// HTML implements types.Message.
func (e EncryptedChunkMessage) HTML() string {
	return e.String()
}

// RequestFileMessage-------------------------------------

func (RequestFileMessage) NewEmpty() Message {
	return &RequestFileMessage{}
}

// Name implements types.Message.
func (RequestFileMessage) Name() string {
	return "reqfile"
}

// String implements types.Message.
func (r RequestFileMessage) String() string {
	return fmt.Sprintf("RequestFileMessage hr: %T, file : %T", r.HRID, r.MetaFileID)
}

// HTML implements types.Message.
func (r RequestFileMessage) HTML() string {
	return r.String()
}

// ResponseFileMessage-------------------------------------
func (ResponseFileMessage) NewEmpty() Message {
	return &ResponseFileMessage{}
}

// Name implements types.Message.
func (ResponseFileMessage) Name() string {
	return "resfile"
}

// String implements types.Message.
func (r ResponseFileMessage) String() string {
	return fmt.Sprintf("ResponseFileMessage: From %T, V: %T", r.I, r.V)
}

// HTML implements types.Message.
func (r ResponseFileMessage) HTML() string {
	return r.String()
}

// -----------------------project--------------------------(down)
