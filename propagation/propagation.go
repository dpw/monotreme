package propagation

import (
	// Would use the bitset functionality of "math/big", but it
	// doesn't support in-place bit setting.
	"github.com/willf/bitset"

	. "github.com/dpw/monotreme/rudiments"
)

type Version uint64

type Update struct {
	Node    NodeID
	Version Version
	State   interface{}
}

type nodeState struct {
	Update
	delivered bitset.BitSet
}

type Neighbor struct {
	*Propagation
	index uint
}

type Propagation struct {
	neighbors []*Neighbor
	nodes     map[NodeID]*nodeState
}

func NewPropagation() *Propagation {
	return &Propagation{nodes: make(map[NodeID]*nodeState)}
}

func (p *Propagation) Get(node NodeID, def interface{}) interface{} {
	if ns := p.nodes[node]; ns != nil {
		return ns.State
	}

	return def
}

func (p *Propagation) AddNeighbor() *Neighbor {
	n := &Neighbor{Propagation: p, index: uint(len(p.neighbors))}
	p.neighbors = append(p.neighbors, n)
	return n
}

func (n *Neighbor) Remove() {
	n2 := n.neighbors[len(n.neighbors)-1]
	n.neighbors[n.index] = n2
	n.neighbors = n.neighbors[:len(n.neighbors)-1]

	for _, ns := range n.nodes {
		ns.delivered.SetTo(n.index, ns.delivered.Test(n2.index))
		ns.delivered.Clear(n2.index)
	}

	n2.index = n.index
}

func (p *Propagation) update(u Update) *nodeState {
	ns := p.nodes[u.Node]
	if ns == nil {
		ns = &nodeState{Update: u}
		p.nodes[u.Node] = ns
	} else if ns.Version >= u.Version {
		return nil
	} else {
		ns.Update = u
		ns.delivered.ClearAll()
	}

	return ns
}

// Register an update.  Returns true if this update is news.
func (p *Propagation) Update(u Update) bool {
	return p.update(u) != nil
}

// Register an update received from the neighbor.  Returns true if
// this update is news.
func (n *Neighbor) Update(u Update) bool {
	ns := n.update(u)
	if ns == nil {
		return false
	}

	ns.delivered.Set(n.index)
	return true
}

// Get the updates pending for the neighbor
func (n *Neighbor) UpdatesToSend() []Update {
	var res []Update

	for _, ns := range n.nodes {
		if !ns.delivered.Test(n.index) {
			res = append(res, ns.Update)
		}
	}

	return res
}

// Are there pending updates for the neighbor?
func (n *Neighbor) HasUpdates() bool {
	for _, ns := range n.nodes {
		if !ns.delivered.Test(n.index) {
			return true
		}
	}

	return false
}

// Register delivery of some updates to the neighbor
func (n *Neighbor) Delivered(updates []Update) {
	for _, u := range updates {
		ns := n.nodes[u.Node]
		if ns != nil && ns.Version == u.Version {
			ns.delivered.Set(n.index)
		}
	}
}
