package graph

import (
	"math/rand"
	"sort"
	"strconv"

	. "github.com/dpw/monotreme/rudiments"
)

func MapGraph(g map[NodeID][]NodeID) Graph {
	var res []NodeID

	for n := range g {
		res = append(res, n)
	}

	return Graph{
		Nodes: SortNodeIDs(res),
		Edges: func(id NodeID) []NodeID {
			return g[id]
		},
	}
}

type Edge struct {
	A, B NodeID
}

// An undirected graph represented as a set of edges.  The edge pairs
// are sorted.
type Undirected struct {
	Nodes []NodeID
	Edges map[Edge]struct{}
}

func (e Edge) Reverse() Edge {
	return Edge{e.B, e.A}
}

func (e Edge) Reflexive() bool {
	return e.A == e.B
}

func (e Edge) Canonical() Edge {
	if e.A <= e.B {
		return e
	} else {
		return e.Reverse()
	}
}

func (u Undirected) Add(e Edge) {
	if !e.Reflexive() {
		u.Edges[e.Canonical()] = struct{}{}
	}
}

func (u Undirected) Remove(e Edge) {
	if !e.Reflexive() {
		delete(u.Edges, e.Canonical())
	}
}

func (u Undirected) Contains(e Edge) bool {
	if e.Reflexive() {
		return true
	}

	_, present := u.Edges[e.Canonical()]
	return present
}

type edges []Edge

func (es edges) Len() int { return len(es) }

func (es edges) Less(i, j int) bool {
	if es[i].A < es[j].A {
		return true
	} else if es[i].A > es[j].A {
		return false
	} else {
		return es[i].B < es[j].B
	}
}

func (es edges) Swap(i, j int) {
	t := es[i]
	es[i] = es[j]
	es[j] = t
}

func (u Undirected) SortedEdges() []Edge {
	var es edges

	for e := range u.Edges {
		es = append(es, e)
	}

	sort.Sort(es)
	return es
}

func (u Undirected) RandomEdge(r *rand.Rand) Edge {
	a := r.Intn(len(u.Nodes)-1) + 1
	return Edge{u.Nodes[r.Intn(a)], u.Nodes[a]}
}

func (u Undirected) Graph() Graph {
	g := make(map[NodeID][]NodeID)

	// Symmetry
	for e := range u.Edges {
		g[e.A] = append(g[e.A], e.B)
		g[e.B] = append(g[e.B], e.A)
	}

	// Reflexivity
	for _, n := range u.Nodes {
		g[n] = append(g[n], n)
	}

	return Graph{
		Nodes: u.Nodes,
		Edges: func(id NodeID) []NodeID {
			return SortNodeIDs(g[id])
		},
	}
}

func GenerateSparse(r *rand.Rand, size int) Undirected {
	u := Undirected{
		Nodes: []NodeID{NodeID("0")},
		Edges: make(map[Edge]struct{}),
	}

	// Form a random tree
	for i := 1; i < size; i++ {
		n := NodeID(strconv.Itoa(i))
		u.Add(Edge{n, u.Nodes[r.Intn(len(u.Nodes))]})
		u.Nodes = append(u.Nodes, n)
	}

	// Add a few extra edges
	for i := r.Intn(size); i > 0; i-- {
		u.Add(u.RandomEdge(r))
	}

	return u
}

func GenerateDense(r *rand.Rand, size int) Undirected {
	for {
		u := Undirected{
			Nodes: make([]NodeID, size),
			Edges: make(map[Edge]struct{}),
		}

		for i := 0; i < size; i++ {
			u.Nodes[i] = NodeID(strconv.Itoa(i))
		}

		// Form a fully-connected graph
		for i := 0; i < size; i++ {
			for j := 0; j < i; j++ {
				u.Add(Edge{u.Nodes[i], u.Nodes[j]})
			}
		}

		// Remove some edges
		for i := r.Intn(size); i > 0; i-- {
			u.Remove(u.RandomEdge(r))
		}

		if u.Graph().Connected() {
			return u
		}
	}
}
