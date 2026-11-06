package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// simplifyCard (run by ParseCardConstrs) counted a true literal again and
// again without moving to the next one, so a single true literal made any
// cardinality constraint look satisfied and it was dropped: the solver then
// returned models violating it.

func TestSimplifyCardWithTrueLits(t *testing.T) {
	// x1 is true; x1 + x2 + x3 >= 2 still needs one of x2, x3.
	pb := ParseCardConstrs([]CardConstr{AtLeast1(1), {Lits: []int{1, 2, 3}, AtLeast: 2}, AtLeast1(-2), AtLeast1(-3)})
	if st := New(pb).Solve(); st != Unsat {
		t.Errorf("got %v, want UNSAT", st)
	}
	pb = ParseCardConstrs([]CardConstr{AtLeast1(1), {Lits: []int{1, 2, 3, 4}, AtLeast: 3}, AtLeast1(-2)})
	s := New(pb)
	if st := s.Solve(); st != Sat {
		t.Fatalf("got %v, want SAT", st)
	}
	if m := s.Model(); !m[0] || m[1] || !m[2] || !m[3] {
		t.Errorf("model %v, want [true false true true]", m)
	}
}

func TestParseCardConstrsRandomVsBruteForce(t *testing.T) {
	iters := 4000
	if testing.Short() {
		iters = 400
	}
	r := rand.New(rand.NewSource(27))
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
			for len(m) < n {
				m = append(m, false)
			}
			if msg := checkModel(n, cs, nil, m); msg != "" {
				t.Fatalf("cp=%v: %s: %s", cp, msg, desc)
			}
		}
	}
}
