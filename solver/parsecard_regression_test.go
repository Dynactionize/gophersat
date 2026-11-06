package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// ParseCardConstrs did not count the vars of skipped (trivially satisfied)
// constraints, so the solver's model was shorter than the caller's var range
// and indexing it by those vars panicked.

func TestParseCardConstrsCountsAllVars(t *testing.T) {
	pb := ParseCardConstrs([]CardConstr{AtLeast1(1, 2), {Lits: []int{3, -5}, AtLeast: 0}})
	if pb.NbVars != 5 {
		t.Fatalf("NbVars = %d, want 5", pb.NbVars)
	}
	s := New(pb)
	if st := s.Solve(); st != Sat {
		t.Fatalf("Solve = %v", st)
	}
	if m := s.Model(); len(m) != 5 {
		t.Fatalf("model has %d vars, want 5", len(m))
	}
}

func TestParseCardConstrsModelCoversAllVars(t *testing.T) {
	iters := 4000
	if testing.Short() {
		iters = 400
	}
	r := rand.New(rand.NewSource(28))
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 2
		var constrs []CardConstr
		var cs []bfConstr
		for k := r.Intn(2*n) + 1; k > 0; k-- {
			l := r.Intn(n) + 1
			var lits []int
			for _, p := range r.Perm(n)[:l] {
				lits = append(lits, (p+1)*(2*r.Intn(2)-1))
			}
			card := r.Intn(l+2) - 1 // from -1 (trivial) to l+1 (impossible)
			ones := make([]int, l)
			for j := range ones {
				ones[j] = 1
			}
			constrs = append(constrs, CardConstr{Lits: append([]int(nil), lits...), AtLeast: card})
			cs = append(cs, bfConstr{lits, ones, card})
		}
		maxVar := 0
		for _, c := range cs {
			for _, l := range c.lits {
				maxVar = max(maxVar, l, -l)
			}
		}
		desc := fmt.Sprintf("n=%d %v", n, cs)
		want := bfCount(n, cs, nil) > 0
		for _, cp := range []bool{false, true} {
			s := New(ParseCardConstrs(constrs))
			s.CuttingPlanes = cp
			st := s.Solve()
			if (st == Sat) != want {
				t.Fatalf("cp=%v: got %v, want sat=%v: %s", cp, st, want, desc)
			}
			if st != Sat {
				continue
			}
			m := s.Model()
			if len(m) != maxVar {
				t.Fatalf("cp=%v: model has %d vars, want %d: %s", cp, len(m), maxVar, desc)
			}
			for len(m) < n {
				m = append(m, false)
			}
			if msg := checkModel(n, cs, nil, m); msg != "" {
				t.Fatalf("cp=%v: %s: %s", cp, msg, desc)
			}
		}
	}
}
