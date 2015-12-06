package propagation

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dpw/monotreme/graph"
	. "github.com/dpw/monotreme/rudiments"
)

type link struct {
	from, to *Connection
}

type sim struct {
	cs      map[NodeID]*Connectivity
	pending []*link
}

func (s *sim) connect(a, b *Connectivity) {
	var ab, ba link
	ab.from = a.Connect(b.id, func() { s.pending = append(s.pending, &ab) })
	ba.from = b.Connect(a.id, func() { s.pending = append(s.pending, &ba) })
	ab.to = ba.from
	ba.to = ab.from
}

func makeSim(u graph.Undirected) *sim {
	sim := &sim{cs: make(map[NodeID]*Connectivity)}

	for _, n := range u.Graph().Nodes {
		sim.cs[n] = NewConnectivity(n)
	}

	for e := range u {
		sim.connect(sim.cs[e.A], sim.cs[e.B])
	}

	return sim
}

func (s *sim) run(rng *rand.Rand) {
	for len(s.pending) > 0 {
		i := rng.Intn(len(s.pending))
		l := s.pending[i]
		s.pending[i] = s.pending[len(s.pending)-1]
		s.pending = s.pending[:len(s.pending)-1]

		us := l.from.Updates()
		if us != nil {
			l.to.Receive(us)
			l.from.Delivered(us)
		}
	}
}

func (s *sim) test(t *testing.T, rng *rand.Rand) {
	s.run(rng)

	var expect map[NodeID][]NodeID
	for _, c := range s.cs {
		if expect == nil {
			expect = c.dump()
		} else {
			require.Equal(t, expect, c.dump())
		}
	}
}

// Dump the contents of a Connectivity to simple representation
func (c *Connectivity) dump() map[NodeID][]NodeID {
	res := make(map[NodeID][]NodeID)
	for n, state := range c.prop.nodes {
		res[n] = state.State.([]NodeID)
	}
	return res
}

func TestConnectivity(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 5; i++ {
		makeSim(graph.GenerateDense(rng, 10)).test(t, rng)
		makeSim(graph.GenerateSparse(rng, 10)).test(t, rng)
	}
}
