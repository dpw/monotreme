package rudiments

import (
	"crypto/rand"
	"encoding/hex"
)

// A node ID is a string, so that it can be used as a map key. It is
// not necessrily human-readable.
type NodeID string

const GUID_LEN uint = 8

func NewNodeID() NodeID {
	bs := make([]byte, GUID_LEN)
	_, err := rand.Read(bs)
	if err != nil {
		panic("unable to get random bytes for NodeID")
	}
	return NodeID(hex.EncodeToString(bs))
}
