package propagation

import (
	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type Link struct {
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
	links map[NodeID]*Link
}

func NewConnectivity(id NodeID) *Connectivity {
	return &Connectivity{
		id:    id,
		prop:  NewPropagation(),
		links: make(map[NodeID]*Link),
	}
}

func (c *Connectivity) Link(node NodeID) *Link {
	if _, present := c.links[node]; present {
		panic("already linked")
	}

	link := &Link{
		c:        c,
		node:     node,
		Neighbor: c.prop.AddNeighbor(),
	}
	c.links[node] = link

	c.linksChanged()
	return link
}

func (link *Link) Close() {
	link.Neighbor.Remove()
	delete(link.c.links, link.node)
	link.c.linksChanged()
}

func (c *Connectivity) linksChanged() {
	c.version++
	c.prop.Set(Update{Node: c.id, Version: c.version,
		State: graph.SortNodeIDs(c.linkNodeIDs())})
	c.propagate()
}

func (c *Connectivity) linkNodeIDs() []NodeID {
	var links []NodeID
	for n := range c.links {
		links = append(links, n)
	}
	return links
}

func (c *Connectivity) propagate() {
	// reachability prune
	g := graph.ReachableGraph(c.id, func(node NodeID) []NodeID {
		return c.prop.Get(node, []NodeID(nil)).([]NodeID)
	})

	// The graph g might not be symmetric, as we might hear that
	// one side of a link was dropped or established
	// before/without hearing about the other side.
	//
	// We can't simply make it symmetric by adding edges to make
	// an undirected graph, because that might result in an edge in
	// the spanning tree that doesn't correspond to a working
	// link.
	//
	// Alteratively, we can make an undirected graph by removing
	// edges that don't have a counterpart reverse edge.  This is
	// better, but introduces a bootstrapping problem: When we add
	// a link to another node, we don't know that it is linked to
	// us, and so the edge won't feature in the graph.  Which
	// would mean that we can never learn anything from other
	// nodes.
	//
	// So we use the intersected graph, but enhance it with the
	// local graph whcih reflects links from this node to
	// neighbouring nodes.
	local := graph.Graph{
		Nodes: graph.SortNodeIDs(append(c.linkNodeIDs(), c.id)),
		Edges: func(node NodeID) []NodeID {
			if node == c.id {
				return c.linkNodeIDs()
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

	for _, link := range c.links {
		link.buddy = false
	}

	for _, b := range t.Undirected().Edges(c.id) {
		link := c.links[b]
		link.buddy = true
		if link.HasOutgoing() && link.pending != nil {
			link.pending()
		}
	}
}

func (link *Link) SetPendingFunc(pending func()) {
	link.pending = pending
	if link.HasOutgoing() && pending != nil {
		pending()
	}
}

func (link *Link) Receive(updates []Update) {
	news := false

	for _, u := range updates {
		if link.Incoming(u) {
			news = true
		}
	}

	if news {
		link.c.propagate()
	}
}

func (link *Link) Outgoing() []Update {
	if !link.buddy {
		return nil
	}

	return link.Neighbor.Outgoing()
}

// Dump the contents of a Linkectivity to simple representation
func (c *Connectivity) Dump() map[NodeID]interface{} {
	res := make(map[NodeID]interface{})
	for n, state := range c.prop.nodes {
		res[n] = state.Update.State
	}
	return res
}
