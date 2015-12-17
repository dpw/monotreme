package propagation

import (
	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type Connection struct {
	c    *Connectivity
	node NodeID
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

func (c *Connectivity) Connect(node NodeID) *Connection {
	if _, present := c.conns[node]; present {
		panic("already connected")
	}

	conn := &Connection{
		c:        c,
		node:     node,
		Neighbor: c.prop.AddNeighbor(),
	}
	c.conns[node] = conn

	c.connectionsChanged()
	return conn
}

func (conn *Connection) Close() {
	conn.Neighbor.Remove()
	delete(conn.c.conns, conn.node)
	conn.c.connectionsChanged()
}

func (c *Connectivity) connectionsChanged() {
	c.version++
	c.prop.Update(Update{Node: c.id, Version: c.version,
		State: graph.SortNodeIDs(c.connNodeIDs())})
	c.propagate()
}

func (c *Connectivity) connNodeIDs() []NodeID {
	var conns []NodeID
	for n := range c.conns {
		conns = append(conns, n)
	}
	return conns
}

func (c *Connectivity) propagate() {
	// reachability prune
	g := graph.ReachableGraph(c.id, func(node NodeID) []NodeID {
		return c.prop.Get(node, []NodeID(nil)).([]NodeID)
	})

	// The graph g might not be symmetric, as we might hear that
	// one side of a connection was dropped or established
	// before/without hearing about the other side.
	//
	// We can't simply make it symmetric by adding edges to make
	// an undirected graph, because that might result in an edge in
	// the spanning tree that doesn't correspond to a working
	// connection.
	//
	// Alteratively, we can make an undirected graph by removing
	// edges that don't have a counterpart reverse edge.  This is
	// better, but introduces a bootstrapping problem: When we add
	// a connection to another node, we don't know that it is
	// connected to us, and so the edge won't feature in the
	// graph.  Which would mean that we can never learn anything
	// from other nodes.
	//
	// So we use the intersected graph, but enhance it with the
	// local graph whcih reflects connections from this node to
	// neighbouring nodes.
	local := graph.Graph{
		Nodes: graph.SortNodeIDs(append(c.connNodeIDs(), c.id)),
		Edges: func(node NodeID) []NodeID {
			if node == c.id {
				return c.connNodeIDs()
			} else {
				return nil
			}
		},
	}

	local = local.Union(local.Transpose())

	g = g.Intersect(g.Transpose()).Union(local)

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
		if conn.HasUpdates() && conn.pending != nil {
			conn.pending()
		}
	}
}

func (conn *Connection) SetPendingFunc(pending func()) {
	conn.pending = pending
	if conn.HasUpdates() && pending != nil {
		pending()
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

func (conn *Connection) UpdatesToSend() []Update {
	if !conn.buddy {
		return nil
	}

	return conn.Neighbor.UpdatesToSend()
}

// Dump the contents of a Connectivity to simple representation
func (c *Connectivity) Dump() map[NodeID]interface{} {
	res := make(map[NodeID]interface{})
	for n, state := range c.prop.nodes {
		res[n] = state.Update.State
	}
	return res
}
