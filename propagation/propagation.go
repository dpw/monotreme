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

	// Records to which neighbors this state has been delivered.
	delivered bitset.BitSet
}

type Neighbor struct {
	*Propagation
	index uint

	// Updates that have yet to be delivered to this neighbor.
	// This is just a cache for the nodeState delivered bits, so
	// we don't have to search to find updates pending for a
	// neighbor.
	undelivered map[*nodeState]struct{}
}

type Propagation struct {
	neighbors []*Neighbor
	nodes     map[NodeID]*nodeState
	onChange  func()
}

func newPropagation(onChange func()) *Propagation {
	return &Propagation{
		nodes:    make(map[NodeID]*nodeState),
		onChange: onChange,
	}
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
	n.index = ^uint(0)
}

func (n *Neighbor) activate() {
	if n.undelivered != nil {
		return
	}

	undelivered := make(map[*nodeState]struct{})
	for _, ns := range n.nodes {
		if !ns.delivered.Test(n.index) {
			undelivered[ns] = struct{}{}
		}
	}

	n.undelivered = undelivered
}

func (n *Neighbor) deactivate() {
	n.undelivered = nil
}

func (p *Propagation) clearDelivered(ns *nodeState) {
	del := &ns.delivered
	for i, e := del.NextSet(0); e; i, e = del.NextSet(i + 1) {
		n := p.neighbors[i]
		if n.undelivered != nil {
			n.undelivered[ns] = struct{}{}
		}
		del.Clear(i)
	}
}

func (p *Propagation) addNodeState(u Update) *nodeState {
	ns := &nodeState{Update: u}
	p.nodes[u.Node] = ns

	for _, n := range p.neighbors {
		if n.undelivered != nil {
			n.undelivered[ns] = struct{}{}
		}
	}

	return ns
}

func (n *Neighbor) setDelivered(ns *nodeState) {
	if n.undelivered != nil {
		delete(n.undelivered, ns)
	}
	ns.delivered.Set(n.index)
}

// Register an update.  Returns true if this update is news.
func (p *Propagation) Set(n NodeID, state interface{}) {
	ns := p.nodes[n]
	if ns == nil {
		p.addNodeState(Update{n, 0, state})
	} else {
		ns.Version++
		ns.State = state
		p.clearDelivered(ns)
	}

	p.onChange()
}

// Register an update received from the neighbor.  Returns true if
// this update is news.
func (n *Neighbor) Incoming(updates []Update) {
	news := false

	for _, u := range updates {
		ns := n.nodes[u.Node]
		if ns == nil {
			ns = n.addNodeState(u)
		} else if ns.Version >= u.Version {
			continue
		} else {
			ns.Update = u
			n.clearDelivered(ns)
		}

		n.setDelivered(ns)
		news = true
	}

	if news {
		n.onChange()
	}
}

// Get the updates pending for the neighbor
func (n *Neighbor) Outgoing() []Update {
	n.activate()
	var res []Update

	for ns := range n.undelivered {
		res = append(res, ns.Update)
	}

	return res
}

// Are there pending updates for the neighbor?
func (n *Neighbor) HasOutgoing() bool {
	n.activate()
	return len(n.undelivered) > 0
}

// Register delivery of some updates to the neighbor
func (n *Neighbor) Delivered(updates []Update) {
	for _, u := range updates {
		ns := n.nodes[u.Node]
		if ns != nil && ns.Version == u.Version {
			n.setDelivered(ns)
		}
	}
}
