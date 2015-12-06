package graph

import (
	"math/rand"
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
type Undirected map[Edge]struct{}

func makeEdge(a, b NodeID) Edge {
	if a > b {
		t := a
		a = b
		b = t
	}
	return Edge{a, b}
}

func (u Undirected) Add(a, b NodeID) {
	if a != b {
		u[makeEdge(a, b)] = struct{}{}
	}
}

func (u Undirected) Remove(a, b NodeID) {
	if a != b {
		delete(u, makeEdge(a, b))
	}
}

func (u Undirected) Graph() Graph {
	g := make(map[NodeID][]NodeID)

	// Symmetry
	for e := range u {
		g[e.A] = append(g[e.A], e.B)
		g[e.B] = append(g[e.B], e.A)
	}

	// Reflexivity
	for n := range g {
		g[n] = append(g[n], n)
	}

	return MapGraph(g)
}

func GenerateSparse(r *rand.Rand, size int) Undirected {
	u := make(Undirected)

	nodes := []NodeID{NodeID("0")}

	// Form a random tree
	for i := 1; i < size; i++ {
		n := NodeID(strconv.Itoa(i))
		u.Add(n, nodes[r.Intn(len(nodes))])
		nodes = append(nodes, n)
	}

	// Add a few extra edges
	for i := r.Intn(size); i > 0; i-- {
		u.Add(nodes[r.Intn(len(nodes))], nodes[r.Intn(len(nodes))])
	}

	return u
}

func GenerateDense(r *rand.Rand, size int) Undirected {
	u := make(Undirected)
	nodes := make([]NodeID, size)

	for i := 0; i < size; i++ {
		nodes[i] = NodeID(strconv.Itoa(i))
	}

	// Form a fully-connected graph
	for i := 0; i < size; i++ {
		for j := 0; j < i; j++ {
			u.Add(nodes[i], nodes[j])
		}
	}

	// Remove some edges
	for i := r.Intn(size); i > 0; i-- {
		a := r.Intn(len(nodes)-1) + 1
		u.Remove(nodes[r.Intn(len(nodes))], nodes[r.Intn(a)])
	}

	return u
}
