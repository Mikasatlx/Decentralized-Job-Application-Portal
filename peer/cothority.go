package peer

import "go.dedis.ch/cs438/types"

type Cothority interface {
	CreateCothorityNode(index uint, addresses []string, hrSecretTable map[uint]string)

	StartCothorityNode()

	GetBlockchain() []types.PaxosValue
}
