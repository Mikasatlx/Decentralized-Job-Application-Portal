package types

import "fmt"

// -----------------------------------------------------------------------------
// PruneMessage

// PruneMessage implements types.Message.
func (p PruneMessage) NewEmpty() Message {
	return &PruneMessage{}
}

// Name implements types.Message.
func (PruneMessage) Name() string {
	return "Prune"
}

// String implements types.Message.
func (p PruneMessage) String() string {
	return "{∅}"
}

// HTML implements types.Message.
func (p PruneMessage) HTML() string {
	return "{∅}"
}

// -----------------------------------------------------------------------------
// IHaveMessage

// NewEmpty implements types.Message.
func (i IHaveMessage) NewEmpty() Message {
	return &IHaveMessage{}
}

// Name implements types.Message.
func (IHaveMessage) Name() string {
	return "ihave"
}

// String implements types.Message.
func (i IHaveMessage) String() string {
	return fmt.Sprintf("<%s>", i.MsgHash)
}

// HTML implements types.Message.
func (i IHaveMessage) HTML() string {
	return i.String()
}

// -----------------------------------------------------------------------------
// GraftMessage

// NewEmpty implements types.Message.
func (g GraftMessage) NewEmpty() Message {
	return &GraftMessage{}
}

// Name implements types.Message.
func (GraftMessage) Name() string {
	return "graft"
}

// String implements types.Message.
func (g GraftMessage) String() string {
	return fmt.Sprintf("<%s>", g.MsgHash)
}

// HTML implements types.Message.
func (g GraftMessage) HTML() string {
	return g.String()
}

// -----------------------------------------------------------------------------
// AckIHaveMessage

// NewEmpty implements types.Message.
func (g AckIHaveMessage) NewEmpty() Message {
	return &AckIHaveMessage{}
}

// Name implements types.Message.
func (AckIHaveMessage) Name() string {
	return "ackihavemsg"
}

// String implements types.Message.
func (g AckIHaveMessage) String() string {
	return fmt.Sprintf("<%s>", g.MsgHash)
}

// HTML implements types.Message.
func (g AckIHaveMessage) HTML() string {
	return g.String()
}
