package propagation

import (
	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type Connectivity struct {
	id       NodeID
	connProp *Propagation
	props    []*Propagation
	links    map[NodeID]*Link
}

type Link struct {
	c         *Connectivity
	node      NodeID
	neighbors map[*Propagation]*Neighbor
	pending   func()

	// pendingProps is non-nil when this is a tree link
	pendingProps map[*Propagation]*Neighbor
}

func NewConnectivity(id NodeID) *Connectivity {
	c := &Connectivity{
		id:    id,
		links: make(map[NodeID]*Link),
	}
	c.connProp = newPropagation(c.connectivityChange)
	return c
}

func (c *Connectivity) ConnectivityPropagation() *Propagation {
	return c.connProp
}

func (c *Connectivity) Link(node NodeID) *Link {
	if _, present := c.links[node]; present {
		panic("already linked")
	}

	link := &Link{
		c:    c,
		node: node,
		neighbors: map[*Propagation]*Neighbor{
			c.connProp: c.connProp.AddNeighbor(),
		},
	}

	for _, prop := range c.props {
		link.neighbors[prop] = prop.AddNeighbor()
	}

	c.links[node] = link
	c.linksChanged()
	return link
}

func (link *Link) Close() {
	for _, neighbor := range link.neighbors {
		neighbor.Remove()
	}

	delete(link.c.links, link.node)
	link.c.linksChanged()
}

func (c *Connectivity) linksChanged() {
	c.connProp.Set(c.id, graph.SortNodeIDs(c.linkNodeIDs()))
}

func (c *Connectivity) linkNodeIDs() []NodeID {
	var links []NodeID
	for n := range c.links {
		links = append(links, n)
	}
	return links
}

func (c *Connectivity) connectivityChange() {
	// reachability prune
	g := graph.ReachableGraph(c.id, func(node NodeID) []NodeID {
		return c.connProp.Get(node, []NodeID(nil)).([]NodeID)
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

	// recompute spanning tree
	pcn := graph.FindPseudoCentralNode(g, 10)
	t := graph.MakeBushySpanningTree(g, pcn, 4)

	treeLinks := make(map[NodeID]struct{})
	for _, n := range t.Undirected().Edges(c.id) {
		treeLinks[n] = struct{}{}
	}

	for n, link := range c.links {
		if _, present := treeLinks[n]; present {
			link.pendingProps = make(map[*Propagation]*Neighbor)
		} else {
			link.pendingProps = nil
			for _, neighbor := range link.neighbors {
				neighbor.deactivate()
			}
		}
	}

	c.checkPending(c.connProp)
}

func (c *Connectivity) checkPending(prop *Propagation) {
	// XXX store separate treeLink list
	for _, link := range c.links {
		if link.pending != nil && link.pendingProps != nil &&
			link.checkPending(prop) {
			link.pending()
		}
	}
}

func (link *Link) checkPending(prop *Propagation) bool {
	n := link.neighbors[prop]
	if !n.HasOutgoing() {
		return false
	}

	if _, present := link.pendingProps[prop]; present {
		return false
	}

	link.pendingProps[prop] = n
	return true
}

func (link *Link) SetPendingFunc(pending func()) {
	link.pending = pending
	if pending != nil && link.pendingProps != nil {
		p := false
		for prop := range link.neighbors {
			p = (p || link.checkPending(prop))
		}

		if p {
			pending()
		}
	}
}

func (link *Link) Outgoing() map[*Propagation][]Update {
	res := make(map[*Propagation][]Update)
	if link.pendingProps != nil {
		for prop, n := range link.pendingProps {
			o := n.Outgoing()
			if o != nil {
				res[prop] = o
			}
		}
	}

	return res
}

func (link *Link) Delivered(prop *Propagation, updates []Update) {
	n := link.neighbors[prop]
	if n != nil {
		n.Delivered(updates)
		if link.pendingProps != nil && !n.HasOutgoing() {
			delete(link.pendingProps, prop)
		}
	}
}

func (link *Link) Incoming(prop *Propagation, updates []Update) {
	n := link.neighbors[prop]
	if n != nil {
		n.Incoming(updates)
	}
}

// Dump the contents of a Linkectivity to simple representation
func (c *Connectivity) Dump() map[NodeID]interface{} {
	res := make(map[NodeID]interface{})
	for n, state := range c.connProp.nodes {
		res[n] = state.Update.State
	}
	return res
}
