package osedax

import (
	"sort"
)

// For routing

// Spanning tree:

// Pick N arbitrary nodes
// Calculate distance from those to all other nodes

// Calculate depth first

type NodeID string

// A graph is a function that produces the outgoing edges from a node
type Graph interface {
	// Get the list of nodes of the graph.  Callers should not
	// modify the result.
	Nodes() []NodeID
	Edges(NodeID) []NodeID
}

// The shortest path result for a particular node
type ShortestPath struct {
	Distance int
	Initial  NodeID
}

// Depth-first search to find shortest paths
func FindShortestPaths(g Graph, start NodeID) map[NodeID]ShortestPath {
	type todoItem struct {
		node    NodeID
		initial NodeID
	}

	res := map[NodeID]ShortestPath{start: {0, start}}
	var todo_next []todoItem
	dist := 0

	reached := func(n NodeID, initial NodeID) {
		if _, present := res[n]; present {
			return
		}

		res[n] = ShortestPath{dist, initial}
		todo_next = append(todo_next, todoItem{n, initial})
	}

	// Add a placeholder entry for the start node
	reached(start, start)

	// First iteration is special, because of Initial
	dist++
	for _, n := range g.Edges(start) {
		reached(n, n)
	}

	todo := todo_next
	todo_next = nil

	for len(todo) > 0 {
		dist++
		for _, i := range todo {
			for _, n := range g.Edges(i.node) {
				reached(n, i.initial)
			}
		}

		temp := todo
		todo = todo_next
		todo_next = temp[:0]
	}

	return res
}

const MaxInt int = int(^uint(0) >> 1)

func SortNodeIDs(ids []NodeID) []NodeID {
	strs := make([]string, len(ids))
	for i := range ids {
		strs[i] = string(ids[i])
	}

	sort.StringSlice(strs).Sort()
	ids = make([]NodeID, len(ids))
	for i := range ids {
		ids[i] = NodeID(strs[i])
	}

	return ids
}

func FindPseudoCentralNode(g Graph, witnesses int) NodeID {
	// The pseudo-eccentricity for each node: the maximum distance
	// from a witness
	eccs := make(map[NodeID]int)

	fillEccsFrom := func(n NodeID) {
		for m, sp := range FindShortestPaths(g, n) {
			if sp.Distance > eccs[m] {
				eccs[m] = sp.Distance
			}
		}
	}

	nodes := g.Nodes()
	if len(nodes) <= witnesses {
		for _, n := range nodes {
			fillEccsFrom(n)
		}
	} else {
		nodes = SortNodeIDs(nodes)

		a := 0
		b := len(nodes) - 1
		for i := witnesses; i > 0; i-- {
			fillEccsFrom(nodes[a])
			a++
			i--
			if i <= 0 {
				break
			}
			fillEccsFrom(nodes[b])
			b--
		}
	}

	// The pseudo-central node is the node with the minimal
	// pseudo-eccentricity and the lowest NodeID.
	minEcc := MaxInt
	var res NodeID

	for n, e := range eccs {
		if e < minEcc || (e == minEcc && n < res) {
			minEcc = e
			res = n
		}
	}

	return res
}
