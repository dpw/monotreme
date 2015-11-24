package osedax

import (
	"sort"
)

// XXX determinism comments

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
	// from a witness node
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

type TreeNode struct {
	id       NodeID
	parent   *TreeNode
	children []*TreeNode
}

type Tree map[NodeID]*TreeNode

func (t Tree) Nodes() []NodeID {
	var res []NodeID
	for id, _ := range t {
		res = append(res, id)
	}
	return res
}

func (t Tree) Edges(id NodeID) []NodeID {
	var res []NodeID
	for _, tn := range t[id].children {
		res = append(res, tn.id)
	}
	return res
}

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
