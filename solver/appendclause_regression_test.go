package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// AppendClause may introduce vars unknown to the solver. newVar did not grow
// the assumptions and cutting-planes buffers, so the next conflict involving
// a new var panicked (learn.go / learn_pb.go index out of range).

func TestAppendClauseNewVarsThenConflicts(t *testing.T) {
	for _, cp := range []bool{false, true} {
		s := New(ParseSliceNb([][]int{{1, 2}, {-1, 2}}, 2))
		s.CuttingPlanes = cp
		if st := s.Solve(); st != Sat {
			t.Fatalf("cp=%v: initial Solve = %v, want SAT", cp, st)
		}
		cnf := [][]int{{1, 2}, {-1, 2}}
		extra := [][]int{{3, 4}, {3, -4}, {-3, 5}, {-3, -5, 6}, {-6, 4}, {-3, -4, -6}, {5, 6}}
		for _, c := range extra {
			s.AppendClause(NewClause(intsToLits(c)))
			cnf = append(cnf, c)
		}
		want := bfCount(6, cnfConstrs(cnf), nil) > 0
		if st := s.Solve(); (st == Sat) != want {
			t.Errorf("cp=%v: Solve = %v, want sat=%v", cp, st, want)
		}
	}
}

// Random incremental sessions: the solver starts with clauses over the first
// k vars only; clauses and PB constraints over all n vars are appended one by
// one, solving after each, and every answer is checked against brute force.
func TestAppendClauseRandomVsBruteForce(t *testing.T) {
	iters := 2000
	if testing.Short() {
		iters = 200
	}
	for _, cp := range []bool{false, true} {
		r := rand.New(rand.NewSource(6))
		for it := 0; it < iters; it++ {
			n := r.Intn(8) + 3
			k := r.Intn(n-1) + 1
			cnf := randCNF(r, k, r.Intn(2*k)+1)
			cs := cnfConstrs(cnf)
			s := New(ParseSliceNb(cloneCNF(cnf), k))
			s.CuttingPlanes = cp
			desc := fmt.Sprintf("cp=%v n=%d k=%d cnf=%v", cp, n, k, cnf)
			s.Solve()
			for step := 0; step < 2*n; step++ {
				if r.Intn(3) == 0 {
					pbs, bfs := randPBConstrs(r, n, 1)
					pb := pbs[0]
					if pb.AtLeast <= 0 || pb.AtLeast > pb.WeightSum() {
						continue
					}
					desc += fmt.Sprintf(" +%v", bfs[0])
					s.AppendClause(pb.Clause())
					cs = append(cs, bfs[0])
				} else {
					c := randCNF(r, n, 1)[0]
					desc += fmt.Sprintf(" +%v", c)
					s.AppendClause(NewClause(intsToLits(c)))
					cs = append(cs, cnfConstrs([][]int{c})...)
				}
				want := bfCount(n, cs, nil) > 0
				st := s.Solve()
				if (st == Sat) != want {
					t.Fatalf("Solve = %v, want sat=%v: %s", st, want, desc)
				}
				if st == Sat {
					m := s.Model()
					for len(m) < n { // vars never mentioned are free
						m = append(m, false)
					}
					if msg := checkModel(n, cs, nil, m); msg != "" {
						t.Fatalf("%s: %s", msg, desc)
					}
				}
				if st == Unsat {
					break
				}
			}
		}
	}
}
