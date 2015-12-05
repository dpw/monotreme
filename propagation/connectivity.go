package propagation

import (
	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type Connection struct {
	c *Connectivity
	*Neighbor
	buddy   bool
	pending func()
}

type Connectivity struct {
	id      NodeID
	version Version

	prop  *Propagation
	conns map[NodeID]*Connection
}

func NewConnectivity(id NodeID) *Connectivity {
	return &Connectivity{
		id:    id,
		prop:  NewPropagation(),
		conns: make(map[NodeID]*Connection),
	}
}

func (c *Connectivity) Connect(node NodeID, pending func()) *Connection {
	if _, present := c.conns[node]; present {
		panic("already connected")
	}

	conn := &Connection{
		c:        c,
		Neighbor: c.prop.AddNeighbor(),
		pending:  pending,
	}
	c.conns[node] = conn

	var conns []NodeID
	for n := range c.conns {
		conns = append(conns, n)
	}
	conns = graph.SortNodeIDs(conns)

	c.version++
	c.prop.Update(Update{Node: c.id, Version: c.version, State: conns})
	c.propagate()

	return conn
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

	for _, conn := range c.conns {
		conn.buddy = false
	}

	for _, b := range t.Undirected().Edges(c.id) {
		conn := c.conns[b]
		conn.buddy = true
		if conn.HasUpdates() {
			conn.pending()
		}
	}
}

func (conn *Connection) Receive(updates []Update) {
	news := false

	for _, u := range updates {
		if conn.Update(u) {
			news = true
		}
	}

	if news {
		conn.c.propagate()
	}
}

func (conn *Connection) Updates() []Update {
	if !conn.buddy {
		return nil
	}

	return conn.Neighbor.Updates()
}

// Operations:

// remove conn

// Delivered
// - pass through to propagation

// Scheduler callback: pending updates on a connection
