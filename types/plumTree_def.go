package types

// PruneMessage means remove me from eager list to lazylist

// - implements types.Message
// - implemented in GroupHW
type PruneMessage struct {
	MsgHash string
}

// IHaveMessage indicates msg that should be sent to lazylist

// - implements types.Message
// - implemented in GroupHW
type IHaveMessage struct {
	MsgHash string
}

// AckIHaveMessage indicates msg received

// - implements types.Message
// - implemented in GroupHW
type AckIHaveMessage struct {
	MsgHash string
}

// GraftMessage indicates node does not have that msg and should be moved to eager list

// - implements types.Message
// - implemented in GroupHW
type GraftMessage struct {
	MsgHash string
}
