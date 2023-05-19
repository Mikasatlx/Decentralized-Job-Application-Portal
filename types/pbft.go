package types

import "fmt"

// -----------------------------------------------------------------------------
// PrePrepare

// NewEmpty implements types.Message.
func (p PrePrepareMessage) NewEmpty() Message {
	return &PrePrepareMessage{}
}

// Name implements types.Message.
func (p PrePrepareMessage) Name() string {
	return "preprepare"
}

// String implements types.Message.
func (p PrePrepareMessage) String() string {
	return fmt.Sprintf("{preprepare %s - %d - %s}", p.Digest, p.PbftID, p.SrcAddr)
}

// HTML implements types.Message.
func (p PrePrepareMessage) HTML() string {
	return p.String()
}



// -----------------------------------------------------------------------------
// PrepareMessage

// NewEmpty implements types.Message.
func (p PrepareMessage) NewEmpty() Message {
	return &PrepareMessage{}
}

// Name implements types.Message.
func (p PrepareMessage) Name() string {
	return "prepare"
}

// String implements types.Message.
func (p PrepareMessage) String() string {
	return fmt.Sprintf("{prepare %s - %d - %s}", p.Digest, p.PbftID, p.SrcAddr)
}

// HTML implements types.Message.
func (p PrepareMessage) HTML() string {
	return p.String()
}



// -----------------------------------------------------------------------------
// Commit

// NewEmpty implements types.Message.
func (c CommitMessage) NewEmpty() Message {
	return &CommitMessage{}
}

// Name implements types.Message.
func (c CommitMessage) Name() string {
	return "commit"
}

// String implements types.Message.
func (c CommitMessage) String() string {
	return fmt.Sprintf("{commit %s - %d - %s}", c.Digest, c.PbftID, c.SrcAddr)
}

// HTML implements types.Message.
func (c CommitMessage) HTML() string {
	return c.String()
}



// -----------------------------------------------------------------------------
// Checkpoint

// NewEmpty implements types.Message.
func (ch CheckpointMessage) NewEmpty() Message {
	return &CheckpointMessage{}
}

// Name implements types.Message.
func (ch CheckpointMessage) Name() string {
	return "checkpoint"
}

// String implements types.Message.
func (ch CheckpointMessage) String() string {
	return fmt.Sprintf("{checkpoint %s - %d - %s}", ch.Digest, ch.PbftID, ch.SrcAddr)
}

// HTML implements types.Message.
func (ch CheckpointMessage) HTML() string {
	return ch.String()
}


// -----------------------------------------------------------------------------
// TLC

// NewEmpty implements types.Message.
func (pt PbftTLCMessage) NewEmpty() Message {
	return &PbftTLCMessage{}
}

// Name implements types.Message.
func (pt PbftTLCMessage) Name() string {
	return "pbfttlc"
}

// String implements types.Message.
func (pt PbftTLCMessage) String() string {
	return fmt.Sprintf("{pbfttlc}")
}

// HTML implements types.Message.
func (pt PbftTLCMessage) HTML() string {
	return pt.String()
}