package propagation

import (
	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type connection struct {
	*Neighbor
	pending func()
}

type Connectivity struct {
	id      NodeID
	version Version

	prop    *Propagation
	conns   map[NodeID]connection
	buddies map[NodeID]struct{}
}

func NewConnectivity(id NodeID) *Connectivity {
	return &Connectivity{
		id:    id,
		prop:  NewPropagation(),
		conns: make(map[NodeID]connection),
	}
}

func (c *Connectivity) Connect(node NodeID, pending func()) {
	if _, present := c.conns[node]; present {
		panic("already connected")
	}

	c.conns[node] = connection{c.prop.AddNeighbor(), pending}

	var conns []NodeID
	for n := range c.conns {
		conns = append(conns, n)
	}
	conns = graph.SortNodeIDs(conns)

	c.version++
	c.prop.Update(Update{Node: c.id, Version: c.version, State: conns})
	c.propagate()
}

func (c *Connectivity) propagate() {
	// reachability prune
	g := graph.ReachableGraph(c.id,
		func(node NodeID) []NodeID {
			return (c.prop.Get(node, []NodeID(nil))).([]NodeID)
		})

	// XXX Prune the propagation according to g

	// recompute spanning tree to find buddies
	pcn := graph.FindPseudoCentralNode(g, 10)
	t := graph.MakeBushySpanningTree(g, pcn, 4)
	c.buddies = make(map[NodeID]struct{})
	for _, b := range t.Undirected().Edges(c.id) {
		c.buddies[b] = struct{}{}
		conn := c.conns[b]
		if conn.HasUpdates() {
			conn.pending()
		}
	}
}

func (c *Connectivity) Receive(from NodeID, updates []Update) {
	conn := c.conns[from]
	news := false

	for _, u := range updates {
		if conn.Update(u) {
			news = true
		}
	}

	if news {
		c.propagate()
	}
}

func (c *Connectivity) UpdatesFor(to NodeID) []Update {
	if _, isBuddy := c.buddies[to]; !isBuddy {
		return nil
	}

	return c.conns[to].Updates()
}

func (c *Connectivity) Delivered(to NodeID, updates []Update) {
	if conn, present := c.conns[to]; present {
		conn.Delivered(updates)
	}
}

// Operations:

// remove conn

// Delivered
// - pass through to propagation

// Scheduler callback: pending updates on a connection
