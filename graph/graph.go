package graph

import (
	"sort"

	. "github.com/dpw/monotreme/rudiments"
)

// // A graph is a set of nodes, and a function that produces the
// outgoing edges from a node.  A Graph is /stable/ if the NodeID
// arrays returned by the Edge function always appear in the same
// order.
type Graph struct {
	// Get the list of nodes of the graph.  Callers should not
	// modify the result.
	Nodes []NodeID

	// Get the edges from the given node.  Returns nil for a node
	// not in the graph.
	Edges func(NodeID) []NodeID
}

func Transpose(g Graph) Graph {
	tg := make(map[NodeID][]NodeID)

	for _, n := range g.Nodes {
		for _, m := range g.Edges(n) {
			tg[m] = append(tg[m], n)
		}
	}

	return Graph{Nodes: g.Nodes, Edges: func(n NodeID) []NodeID {
		return tg[n]
	}}
}

// The shortest path result for a particular node
type ShortestPath struct {
	Distance int
	Initial  NodeID
}

// Depth-first search to find shortest paths
//
// Stable if the Graph g is stable.
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

// Find a pseudo-centrol node: The node with lowest eccentricity with
// respect to a set of witness nodes.
func FindPseudoCentralNode(g Graph, witnesses int) NodeID {
	// The pseudo-eccentricity for each node: the maximum distance
	// to a witness node
	eccs := make(map[NodeID]int)

	// Transpose the graph in order to find shortest paths from
	// candidate pseudo-central nodes to the witnesses:
	tg := Transpose(g)
	fillEccsFrom := func(n NodeID) {
		for m, sp := range FindShortestPaths(tg, n) {
			if sp.Distance > eccs[m] {
				eccs[m] = sp.Distance
			}
		}
	}

	if len(g.Nodes) <= witnesses {
		for _, n := range g.Nodes {
			fillEccsFrom(n)
		}
	} else {
		nodes := SortNodeIDs(g.Nodes)

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

type TreeNode struct {
	id       NodeID
	parent   *TreeNode
	children []*TreeNode
}

type Tree map[NodeID]*TreeNode

func (t Tree) nodes() []NodeID {
	var res []NodeID
	for id, _ := range t {
		res = append(res, id)
	}
	return SortNodeIDs(res)
}

func (t Tree) Directed() Graph {
	return Graph{Nodes: t.nodes(), Edges: func(id NodeID) []NodeID {
		tn := t[id]
		if tn == nil {
			return nil
		}

		var res []NodeID
		for _, child := range tn.children {
			res = append(res, child.id)
		}

		return res
	}}
}

func (t Tree) Undirected() Graph {
	return Graph{Nodes: t.nodes(), Edges: func(id NodeID) []NodeID {
		tn := t[id]
		if tn == nil {
			return nil
		}

		var res []NodeID
		if tn.parent != nil {
			res = []NodeID{tn.parent.id}
		}

		for _, child := range tn.children {
			res = append(res, child.id)
		}

		return res
	}}
}

// Produce a spanning tree for the graph that attempts to:
//
// - minimise maximum distance from the given root node, and
//
// - constrain the degree of each node according to softChildLimit
//
// Stable if the graph is stable.
func MakeBushySpanningTree(g Graph, root NodeID, softChildLimit int) Tree {
	type nodeState struct {
		id NodeID

		// If the node has been added to the tree:
		treeNode *TreeNode
		depth    int

		// For a node reached but not added, which nodes this
		// was reached from, i.e. potential parents
		reachedFrom  []*nodeState
		reachedIndex int
	}

	rootNode := &nodeState{id: root, treeNode: &TreeNode{id: root}}
	nodes := map[NodeID]*nodeState{root: rootNode}
	todo := []*nodeState{rootNode}
	var todo_next []*nodeState

	// nodes reached but not yet added to the tree
	var reached []*nodeState

	removeReached := func(index int) {
		l := len(reached) - 1
		reached[index] = reached[l]
		reached[index].reachedIndex = index
		reached = reached[:l]
	}

	attachTreeNode := func(node *nodeState, parent *nodeState) {
		tn := &TreeNode{id: node.id, parent: parent.treeNode}
		tn.parent.children = append(tn.parent.children, tn)
		node.treeNode = tn
		node.depth = parent.depth + 1
		todo_next = append(todo_next, node)
	}

	visit := func(id NodeID, parent *nodeState) {
		node := nodes[id]
		if node == nil {
			node = &nodeState{id: id}
			nodes[id] = node
		}

		if node.treeNode != nil {
			// already added
			return
		}

		// Does the candidate parent already have too many
		// children to add this one?
		if len(parent.treeNode.children) >= softChildLimit {
			if node.reachedFrom == nil {
				node.reachedIndex = len(reached)
				reached = append(reached, node)
			}

			node.reachedFrom = append(node.reachedFrom, parent)
			return
		}

		if node.reachedFrom != nil {
			// node has been reached already, but that
			// gets overridden by adding it now
			node.reachedFrom = nil
			removeReached(node.reachedIndex)
		}

		attachTreeNode(node, parent)
	}

	for {
		for _, node := range todo {
			for _, child := range g.Edges(node.treeNode.id) {
				visit(child, node)
			}
		}

		if len(todo_next) > 0 {
			temp := todo
			todo = todo_next
			todo_next = temp[:0]
			continue
		}

		if len(reached) == 0 {
			break
		}

		// Have some reached-but-not-added nodes to add
		node := reached[0]

		// heuristically choose a parent node
		bestParent := node.reachedFrom[0]
		bestScore := bestParent.depth +
			len(bestParent.treeNode.children)
		for _, parent := range node.reachedFrom[1:] {
			score := parent.depth + len(parent.treeNode.children)
			if score < bestScore {
				bestScore = score
				bestParent = parent
			}
		}

		node.reachedFrom = nil
		removeReached(0)
		attachTreeNode(node, bestParent)
	}

	res := make(Tree)
	for id, node := range nodes {
		res[id] = node.treeNode
	}

	return res
}

type reachable struct {
	nodes []NodeID
	edges map[NodeID][]NodeID
}

func (g *reachable) Nodes() []NodeID {
	return g.nodes
}

func (g *reachable) Edges(n NodeID) []NodeID {
	return g.edges[n]
}

func (g *reachable) visit(n NodeID, edges func(NodeID) []NodeID) {
	if _, present := g.edges[n]; present {
		return
	}

	g.nodes = append(g.nodes, n)
	e := edges(n)
	g.edges[n] = e
	for _, m := range e {
		g.visit(m, edges)
	}
}

func ReachableGraph(start NodeID, edges func(NodeID) []NodeID) Graph {
	res := reachable{edges: make(map[NodeID][]NodeID)}
	res.visit(start, edges)
	return Graph{Nodes: res.nodes, Edges: func(id NodeID) []NodeID {
		return res.edges[id]
	}}
}
