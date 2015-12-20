package propagation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type link struct {
	sender, receiver *Link
	closed           bool
}

type sim struct {
	graph   graph.Undirected
	cs      map[NodeID]*Connectivity
	links   map[graph.Edge]*link
	pending []*link
}

func (s *sim) link(e graph.Edge) {
	a := s.cs[e.A]
	b := s.cs[e.B]

	aconn := a.Link(b.id)
	bconn := b.Link(a.id)

	alink := &link{aconn, bconn, false}
	blink := &link{bconn, aconn, false}

	s.links[e] = alink
	s.links[e.Reverse()] = blink

	aconn.SetPendingFunc(func() { s.pending = append(s.pending, alink) })
	bconn.SetPendingFunc(func() { s.pending = append(s.pending, blink) })
}

func (s *sim) disconnect(e graph.Edge) {
	l := s.links[e]
	l.sender.Close()
	l.receiver.Close()

	l.closed = true
	s.links[e.Reverse()].closed = true

	delete(s.links, e)
	delete(s.links, e.Reverse())
}

func makeSim(g graph.Undirected) *sim {
	sim := &sim{
		graph: g,
		cs:    make(map[NodeID]*Connectivity),
		links: make(map[graph.Edge]*link),
	}

	for _, n := range g.Nodes {
		sim.cs[n] = NewConnectivity(n)
	}

	for _, e := range g.SortedEdges() {
		sim.link(e)
	}

	return sim
}

func dbg(msg ...interface{}) {
	//fmt.Println(msg...)
}

func (s *sim) run(t *testing.T, rng *rand.Rand) {
	dbg(s.graph.Graph().Map())

	step := 0
	for len(s.pending) > 0 {
		if step > 1000000 {
			t.Fatal("non-convergence")
		}
		step++

		// Maybe add or remove a link
		if rng.Intn(100) == 0 {
			e := s.graph.RandomEdge(rng)
			if s.graph.Contains(e) {
				s.graph.Remove(e)

				// Check that the graph did not become
				// disconnected.
				if s.graph.Graph().Connected() {
					dbg("Disconnecting", e)
					s.disconnect(e)
				} else {
					s.graph.Add(e)
				}
			} else {
				dbg("Connecting", e)
				s.graph.Add(e)
				s.link(e)
			}
		}

		// Propagate an update
		i := rng.Intn(len(s.pending))
		l := s.pending[i]
		s.pending[i] = s.pending[len(s.pending)-1]
		s.pending = s.pending[:len(s.pending)-1]

		if l.closed {
			continue
		}

		for prop, updates := range l.sender.Outgoing() {
			dbg(l.sender.c.id, "->", l.receiver.c.id, ":", updates)
			l.receiver.Incoming(l.receiver.c.ConnectivityPropagation(), updates)
			l.sender.Delivered(prop, updates)
		}
	}

	var expect map[NodeID]interface{}
	var expectNode NodeID
	for _, node := range s.graph.Nodes {
		c := s.cs[node]
		if expect == nil {
			expect = c.Dump()
			expectNode = node
		} else {
			require.Equal(t, expect, c.Dump(), "mismatch %s %s", expectNode, node)
		}
	}
}

var seed int64

func makeRNG(msg string) *rand.Rand {
	s := time.Now().UnixNano()
	if s <= seed {
		s = seed + 1
	}
	seed = s

	fmt.Printf("RNG seed for %s: %d\n", msg, s)
	return rand.New(rand.NewSource(s))
}

func TestConnectivity(t *testing.T) {
	for i := 0; i < 50; i++ {
		rng := makeRNG("TestConnectivity dense")
		makeSim(graph.GenerateDense(rng, 7)).run(t, rng)
		rng = makeRNG("TestConnectivity sparse")
		makeSim(graph.GenerateSparse(rng, 7)).run(t, rng)
	}

}
