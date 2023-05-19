package impl

// In this file, we can have an overview of settings and modify them if necessary
const (
	// the number of goCoroutines to listen messages
	RecvNum = 1
	// the number of goCoroutines to send messages
	SendNum = 1
	// the number of goCoroutines to proc messages
	ProcNum = 1
	// the number of lazyqueue resender
	LazyNum = 1
	// the buffer size of a channel
	chanSize = 1000
	// using plumTree
	usePlumTree = true //false //
	/// the buffer size of rumor table
	rumorTableSize  = 1000
	procQueueSize   = 1000
	resendQueueSize = 1000
	//using pbft
	pbft = true

	checkpointInterval = 1000

	// when verifying the file download request, we can select signature verify method: 1 is for HMAC, 2 is for RSA
	signMethod = 2
)
