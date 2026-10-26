package solver

import (
	"math/rand"
	"testing"
)

// Regression tests for the dynactionize fork fixes (see README-FORK.md).

// Fix #1: Assume() must not erase the problem's unit clauses.
// {x1} ∧ {¬x1∨x2} entails x2, so assuming ¬x2 is UNSAT. Upstream erased the
// unit {x1} on the first Assume and reported SAT.
func TestForkAssumeKeepsUnits(t *testing.T) {
	s := New(ParseSlice([][]int{{1}, {-1, 2}}))
	if st := s.Solve(); st != Sat {
		t.Fatalf("Solve: got %v, want SAT", st)
	}
	st := s.Assume([]Lit{IntToLit(-2)})
	if st == Indet {
		st = s.Solve()
	}
	if st != Unsat {
		t.Fatalf("Assume(-2): got %v, want UNSAT", st)
	}
}

// Fix #3: queue.insert is idempotent.
func TestForkQueueInsertIdempotent(t *testing.T) {
	q := newQueue([]float64{0.5, 0.1, 0.9, 0.3})
	for _, n := range []int{0, 1, 2, 3, 2, 0, 2} {
		q.insert(n)
	}
	if len(q.content) != 4 {
		t.Fatalf("queue holds %d entries, want 4: %v", len(q.content), q.content)
	}
	seen := map[int]bool{}
	for !q.empty() {
		n := q.removeMin()
		if seen[n] {
			t.Fatalf("var %d popped twice", n)
		}
		seen[n] = true
	}
}

// Fix #2 (and #1 under load): repeated Assume on one hot solver must agree
// with a fresh solver on status, and every SAT model must satisfy all clauses
// and assumptions. Upstream returned models with unbound (defaulted-false)
// vars that falsified clauses after a few Assume calls.
func TestForkHotAssumeMatchesFresh(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for round := 0; round < 200; round++ {
		nbVars := 8 + rng.Intn(12)
		var clauses [][]int
		for i := 0; i < nbVars*3; i++ {
			size := 1 + rng.Intn(3)
			if rng.Intn(10) > 0 && size == 1 {
				size = 2 // keep a few units, but not too many
			}
			c := make([]int, size)
			for j := range c {
				c[j] = 1 + rng.Intn(nbVars)
				if rng.Intn(2) == 0 {
					c[j] = -c[j]
				}
			}
			clauses = append(clauses, c)
		}
		hot := New(ParseSlice(clauses))
		if hot.Solve() == Unsat {
			continue
		}
		for q := 0; q < 20; q++ {
			var assumptions []int
			for k := 0; k < 1+rng.Intn(3); k++ {
				a := 1 + rng.Intn(nbVars)
				if rng.Intn(2) == 0 {
					a = -a
				}
				assumptions = append(assumptions, a)
			}
			lits := make([]Lit, len(assumptions))
			for i, a := range assumptions {
				lits[i] = IntToLit(int32(a))
			}
			got := hot.Assume(lits)
			if got == Indet {
				got = hot.Solve()
			}

			freshClauses := append([][]int{}, clauses...)
			for _, a := range assumptions {
				freshClauses = append(freshClauses, []int{a})
			}
			want := New(ParseSlice(freshClauses)).Solve()
			if got != want {
				t.Fatalf("round %d query %d: hot solver %v, fresh solver %v (clauses %v, assumptions %v)", round, q, got, want, clauses, assumptions)
			}
			if got == Sat {
				model := hot.Model()
				for _, c := range freshClauses {
					if !satisfied(c, model) {
						t.Fatalf("round %d query %d: model %v falsifies clause %v (assumptions %v)", round, q, model, c, assumptions)
					}
				}
			}
		}
	}
}

func satisfied(clause []int, model []bool) bool {
	for _, l := range clause {
		v := l
		if v < 0 {
			v = -v
		}
		if model[v-1] == (l > 0) {
			return true
		}
	}
	return false
}
