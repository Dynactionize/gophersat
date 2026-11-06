package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// A problem found UNSAT while parsing produces a solver without internal
// state. Assume used to reset its status to Indet, after which Solve
// reported SAT (or Assume panicked with a non-empty assumption list).
func TestAssumeOnTriviallyUnsatProblem(t *testing.T) {
	for _, lits := range [][]Lit{nil, {}, IntsToLits(1), IntsToLits(-1, 2)} {
		s := New(ParseSlice([][]int{{1}, {-1}}))
		st := s.Assume(lits)
		if st != Unsat {
			t.Errorf("Assume(%v) = %v, want UNSAT", lits, st)
		}
		if st = s.Solve(); st != Unsat {
			t.Errorf("Solve after Assume(%v) = %v, want UNSAT", lits, st)
		}
	}
	// Units contradicting each other only through propagation.
	s := New(ParseSlice([][]int{{2}, {-3}, {-2, 3}, {1, 2, 3}}))
	if st := s.Assume(nil); st != Unsat {
		t.Errorf("Assume(nil) = %v, want UNSAT", st)
	}
	if st := s.Solve(); st != Unsat {
		t.Errorf("Solve = %v, want UNSAT", st)
	}
}

// Hot reuse: one solver, many rounds of Assume + Solve, each checked against
// brute force (status and model).
func TestAssumeRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	for _, cp := range []bool{false, true} {
		r := rand.New(rand.NewSource(5))
		for it := 0; it < iters; it++ {
			n := r.Intn(9) + 2
			cnf := randCNF(r, n, r.Intn(4*n)+1)
			cs := cnfConstrs(cnf)
			s := New(ParseSliceNb(cloneCNF(cnf), n))
			s.CuttingPlanes = cp
			hist := ""
			for round := 0; round < 8; round++ {
				fixed := map[int]bool{}
				var lits []Lit
				for k := r.Intn(5); k > 0; k-- {
					x := randLitInt(r, n)
					v := max(x, -x)
					if _, ok := fixed[v]; ok {
						continue
					}
					fixed[v] = x > 0
					lits = append(lits, IntToLit(int32(x)))
				}
				hist += fmt.Sprintf(" %v", lits)
				want := bfCount(n, cs, fixed) > 0
				st := s.Assume(lits)
				if st != Unsat {
					st = s.Solve()
				}
				desc := fmt.Sprintf("cp=%v n=%d cnf=%v assumptions:%s", cp, n, cnf, hist)
				if (st == Sat) != want {
					t.Fatalf("got %v, want sat=%v: %s", st, want, desc)
				}
				if st == Sat {
					if msg := checkModel(n, cs, fixed, s.Model()); msg != "" {
						t.Fatalf("%s: %s", msg, desc)
					}
				}
			}
		}
	}
}
