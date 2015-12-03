package graph

import (
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "github.com/dpw/monotreme/rudiments"
)

func mapGraph(g map[NodeID][]NodeID) Graph {
	var res []NodeID

	for n := range g {
		res = append(res, n)
	}

	return Graph{Nodes: SortNodeIDs(res), Edges: func(id NodeID) []NodeID {
		return g[id]
	}}
}

type edge struct {
	a, b NodeID
}

// An undirected graph represented as a set of edges.  The edge pairs
// are sorted.
type undirected map[edge]struct{}

func makeEdge(a, b NodeID) edge {
	if a > b {
		t := a
		a = b
		b = t
	}
	return edge{a, b}
}

func (u undirected) add(a, b NodeID) {
	u[makeEdge(a, b)] = struct{}{}
}

func (u undirected) remove(a, b NodeID) {
	delete(u, makeEdge(a, b))
}

func (u undirected) toGraph() Graph {
	g := make(map[NodeID][]NodeID)

	// Symmetry
	for e := range u {
		g[e.a] = append(g[e.a], e.b)
		g[e.b] = append(g[e.b], e.a)
	}

	// Reflexivity
	for n := range g {
		g[n] = append(g[n], n)
	}

	return mapGraph(g)
}

func generateSparse(r *rand.Rand, size int) undirected {
	u := make(undirected)

	nodes := []NodeID{NodeID("0")}

	// Form a random tree
	for i := 1; i < size; i++ {
		n := NodeID(strconv.Itoa(i))
		u.add(n, nodes[r.Intn(len(nodes))])
		nodes = append(nodes, n)
	}

	// Add a few extra edges
	for i := r.Intn(size); i > 0; i-- {
		u.add(nodes[r.Intn(len(nodes))], nodes[r.Intn(len(nodes))])
	}

	return u
}

func generateDense(r *rand.Rand, size int) undirected {
	u := make(undirected)
	nodes := make([]NodeID, size)

	for i := 0; i < size; i++ {
		nodes[i] = NodeID(strconv.Itoa(i))
	}

	// Form a fully-connected graph
	for i := 0; i < size; i++ {
		for j := 0; j < i; j++ {
			u.add(nodes[i], nodes[j])
		}
	}

	// Remove some edges
	for i := r.Intn(size); i > 0; i-- {
		a := r.Intn(len(nodes)-1) + 1
		u.remove(nodes[r.Intn(len(nodes))], nodes[r.Intn(a)])
	}

	return u
}

func checkShortestPaths(t *testing.T, g Graph, sps map[NodeID]ShortestPath) {
	// This looks like a lot of code - far more than the shortest
	// path algorithm itself!  The main thing we are checking is
	// that the distance to each node is the lowest value
	// consistent with the distances to its predecessor nodes.

	isps := make(map[NodeID]struct {
		minDistance int
		initials    map[NodeID]struct{}
	})
	foundDistZero := false

	for _, n := range g.Nodes {
		nsp, present := sps[n]
		if !present {
			/// n was not reachable
			continue
		}

		if nsp.Distance == 0 {
			require.False(t, foundDistZero)
			foundDistZero = true
		}

		for _, m := range g.Edges(n) {
			isp, present := isps[m]
			if present {
				if isp.minDistance < nsp.Distance+1 {
					// already found a shorter
					// implied distance to m
					continue
				}

				if nsp.Distance+1 < isp.minDistance {
					isp.minDistance = nsp.Distance + 1
					isp.initials = nil
				}
			} else {
				isp.minDistance = nsp.Distance + 1
			}

			initial := nsp.Initial
			if nsp.Distance == 0 {
				initial = m
			}

			if isp.initials == nil {
				isp.initials = make(map[NodeID]struct{})
			}

			isp.initials[initial] = struct{}{}
			isps[m] = isp
		}
	}

	for n, isp := range isps {
		require.Contains(t, sps, n)
		if sps[n].Distance != 0 {
			require.Equal(t, isp.minDistance, sps[n].Distance,
				"Incorrect distance on %s", n)
			require.Contains(t, isp.initials, sps[n].Initial)
		} else {
			require.Equal(t, n, sps[n].Initial)
		}
	}

	// implied reachability should match
	for n, sp := range sps {
		if sp.Distance != 0 {
			require.Contains(t, isps, n)
		}
	}
}

func randomNode(g Graph, r *rand.Rand) NodeID {
	return g.Nodes[r.Intn(len(g.Nodes))]
}

func rng() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestFindShortestPaths(t *testing.T) {
	r := rng()

	check := func(g Graph) {
		n := randomNode(g, r)
		sp := FindShortestPaths(g, n)
		checkShortestPaths(t, g, sp)

		// FindShortestPaths is stable:
		require.True(t, reflect.DeepEqual(sp, FindShortestPaths(g, n)))
	}

	for i := 0; i < 100; i++ {
		check(generateSparse(r, 10).toGraph())
		check(generateDense(r, 10).toGraph())
	}
}

// Find the eccentricity of a node: the maximum shortest path to
// another node.
func eccentricity(g Graph, n NodeID) int {
	max := 0

	for _, sp := range FindShortestPaths(g, n) {
		if sp.Distance > max {
			max = sp.Distance
		}
	}

	return max
}

// Find the true central nodes: those with minimal eccentricity.
func centralNodes(g Graph) []NodeID {
	minEcc := MaxInt
	var res []NodeID

	for _, n := range g.Nodes {
		ecc := eccentricity(g, n)
		if ecc <= minEcc {
			if ecc < minEcc {
				minEcc = ecc
				res = nil
			}
			res = append(res, n)
		}
	}

	return res
}

func TestFindPseudoCentralNode(t *testing.T) {
	r := rng()

	// Test the case where the all nodes are witnesses, in which
	// case the pseudo-central node is the central node with
	// lowest id

	check := func(g Graph) {
		require.Contains(t, SortNodeIDs(centralNodes(g))[0],
			FindPseudoCentralNode(g, 10))

		// check stability
		require.Equal(t, FindPseudoCentralNode(g, 3),
			FindPseudoCentralNode(g, 3))
	}

	for i := 0; i < 100; i++ {
		check(generateSparse(r, 10).toGraph())
		check(generateDense(r, 10).toGraph())
	}
}

func linearGraph(n int) Graph {
	g := make(map[NodeID][]NodeID)
	prev := NodeID(strconv.Itoa(0))

	for i := 1; i < n; i++ {
		next := NodeID(strconv.Itoa(i))
		g[prev] = append(g[prev], next)
		g[next] = append(g[next], prev)
		prev = next
	}

	return mapGraph(g)
}

func TestFindPseudoCentralNodeOfLinearGraph(t *testing.T) {
	require.Equal(t, NodeID("50"), FindPseudoCentralNode(linearGraph(101),
		10))
	require.Equal(t, NodeID("50"), FindPseudoCentralNode(linearGraph(101),
		11))
}

func graphsEqual(g, h Graph) bool {
	if !reflect.DeepEqual(SortNodeIDs(g.Nodes), SortNodeIDs(h.Nodes)) {
		return false
	}

	for _, n := range g.Nodes {
		if !reflect.DeepEqual(SortNodeIDs(g.Edges(n)),
			SortNodeIDs(h.Edges(n))) {
			return false
		}
	}

	return true
}

// Validate a tree
func checkTree(t *testing.T, tree Tree, root NodeID) {
	copy := make(Tree)
	for k, v := range tree {
		copy[k] = v
	}

	checkTreeNode(t, tree[root], nil, copy)
	require.Len(t, copy, 0)
}

func checkTreeNode(t *testing.T, node, parent *TreeNode, tree Tree) {
	require.Equal(t, node.parent, parent)
	require.Equal(t, node, tree[node.id])
	delete(tree, node.id)

	for _, child := range node.children {
		checkTreeNode(t, child, node, tree)
	}
}

func TestMakeBushySpanningTree(t *testing.T) {
	r := rng()

	check := func(g Graph, witnesses, limit int) {
		root := FindPseudoCentralNode(g, witnesses)
		tr := MakeBushySpanningTree(g, root, limit)
		gnodes := SortNodeIDs(g.Nodes)
		require.Equal(t, gnodes, SortNodeIDs(tr.Directed().Nodes))
		r := ReachableGraph(root, tr.Undirected().Edges)
		require.Equal(t, gnodes, SortNodeIDs(r.Nodes))
		checkTree(t, tr, root)

		// Stability
		require.True(t, graphsEqual(tr.Directed(),
			MakeBushySpanningTree(g, root, limit).Directed()))
	}

	for i := 0; i < 100; i++ {
		check(generateSparse(r, 10).toGraph(), 4, 2)
		check(generateDense(r, 10).toGraph(), 4, 2)
	}

	// Test on some nice big graphs, for luck
	check(generateSparse(r, 100).toGraph(), 10, 3)
	check(generateDense(r, 100).toGraph(), 10, 3)
}
