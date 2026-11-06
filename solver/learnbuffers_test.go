package solver

import (
	"math/rand"
	"testing"
)

// learnClause reuses per-solver buffers instead of allocating 2*nbVars bools
// per conflict. These tests check the invariant this relies on (the buffer
// is all false between conflicts) and that answers are unchanged, including
// when AppendClause grows the number of vars between solves.

func checkMetBufClear(t *testing.T, s *Solver) {
	t.Helper()
	for i, b := range s.metBuf {
		if b {
			t.Fatalf("metBuf[%d] still set after conflict analysis", i)
		}
	}
}

func TestLearnBuffersClearedBetweenConflicts(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		s := New(ParseSlice(random3SAT(seed, 120, 4.26)))
		s.Solve()
		if s.Stats.NbConflicts == 0 {
			continue
		}
		checkMetBufClear(t, s)
	}
}

func TestLearnBuffersWithGrowingVars(t *testing.T) {
	r := rand.New(rand.NewSource(29))
	for it := 0; it < 300; it++ {
		n := r.Intn(6) + 4
		k := r.Intn(n-2) + 2
		cnf := randCNF(r, k, 3*k)
		cs := cnfConstrs(cnf)
		s := New(ParseSliceNb(cloneCNF(cnf), k))
		s.Solve()
		for step := 0; step < 3*n; step++ {
			c := randCNF(r, n, 1)[0]
			s.AppendClause(NewClause(intsToLits(c)))
			cs = append(cs, cnfConstrs([][]int{c})...)
			want := bfCount(n, cs, nil) > 0
			st := s.Solve()
			if (st == Sat) != want {
				t.Fatalf("got %v, want sat=%v", st, want)
			}
			checkMetBufClear(t, s)
			if st == Unsat {
				break
			}
		}
	}
}
