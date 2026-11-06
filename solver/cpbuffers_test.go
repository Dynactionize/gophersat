package solver

import (
	"math/rand"
	"testing"
)

// cuttingPlanes now reuses a per-solver "seen" buffer and builds learned
// constraints without intermediate copies. Check the buffer invariant and
// that the public SimplifyPB still returns an independent clause.

func TestCuttingPlanesSeenBufferCleared(t *testing.T) {
	r := rand.New(rand.NewSource(30))
	for it := 0; it < 300; it++ {
		n := r.Intn(8) + 3
		constrs, cs := randPBConstrs(r, n, 2*n)
		s := New(padVars(ParsePBConstrs(clonePB(constrs)), n))
		s.CuttingPlanes = true
		st := s.Solve()
		if (st == Sat) != (bfCount(n, cs, nil) > 0) {
			t.Fatalf("wrong answer %v on %v", st, cs)
		}
		for i, b := range s.cpSeen {
			if b {
				t.Fatalf("cpSeen[%d] still set after solving %v", i, cs)
			}
		}
	}
}

func TestSimplifyPBReturnsIndependentClause(t *testing.T) {
	c := NewPBClause(IntsToLits(1, 2, 3, 4), []int{2, 2, 1, 1}, 3)
	_, c2, ok := c.SimplifyPB()
	if !ok || c2 == nil {
		t.Fatalf("SimplifyPB: ok=%v c2=%v", ok, c2)
	}
	before := c.PBString()
	for i := range c2.lits {
		c2.lits[i] = c2.lits[i].Negation()
		c2.pbData.weights[i] = 7
	}
	if c.PBString() != before {
		t.Fatalf("modifying the simplified clause changed the original: %s -> %s", before, c.PBString())
	}
}
