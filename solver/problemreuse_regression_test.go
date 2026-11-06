package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// New() used the Problem's model slice and clause objects directly and
// modified them while solving (bindings, lit order, PB watch flags). A second
// solver built from the same Problem started from the first one's state and
// could panic or give wrong answers.

func TestProblemReusedBySeveralSolvers(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(9))
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 2
		var pb *Problem
		var cs []bfConstr
		if r.Intn(2) == 0 {
			cnf := randCNF(r, n, r.Intn(4*n)+1)
			pb, cs = ParseSliceNb(cnf, n), cnfConstrs(cnf)
		} else {
			var constrs []PBConstr
			constrs, cs = randPBConstrs(r, n, r.Intn(2*n)+1)
			pb = padVars(ParsePBConstrs(constrs), n)
		}
		want := bfCount(n, cs, nil) > 0
		desc := fmt.Sprintf("n=%d %v", n, cs)
		snapshot := pb.PBString()
		for k := 0; k < 3; k++ {
			s := New(pb)
			s.CuttingPlanes = k == 2
			st := s.Solve()
			if (st == Sat) != want {
				t.Fatalf("solver #%d: got %v, want sat=%v: %s", k, st, want, desc)
			}
			if st == Sat {
				if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
					t.Fatalf("solver #%d: %s: %s", k, msg, desc)
				}
			}
			if got := pb.PBString(); got != snapshot {
				t.Fatalf("solver #%d modified the problem:\nbefore %s\nafter  %s", k, snapshot, got)
			}
		}
	}
}
