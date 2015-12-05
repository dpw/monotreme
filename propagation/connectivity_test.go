package propagation

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "github.com/dpw/monotreme/rudiments"
)

type link struct {
	from, to *Connection
}

type sim struct {
	cs      []*Connectivity
	pending []*link
}

func (s *sim) connect(a, b *Connectivity) {
	var ab, ba link
	ab.from = a.Connect(b.id, func() { s.pending = append(s.pending, &ab) })
	ba.from = b.Connect(a.id, func() { s.pending = append(s.pending, &ba) })
	ab.to = ba.from
	ba.to = ab.from
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

func fullyConnected() *sim {
	sim := &sim{}

	for i := 0; i < 10; i++ {
		n := len(sim.cs)
		sim.cs = append(sim.cs, NewConnectivity(NodeID(fmt.Sprint(n))))

		for i := 0; i < n; i++ {
			sim.connect(sim.cs[i], sim.cs[n])
		}
	}

	return sim
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

	for i := 0; i < 10; i++ {
		sim := fullyConnected()
		sim.run(rng)

		expect := sim.cs[0].dump()
		for _, c := range sim.cs[1:] {
			require.Equal(t, expect, c.dump())
		}
	}
}
