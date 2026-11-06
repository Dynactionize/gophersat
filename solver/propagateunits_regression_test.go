package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// propagateUnits (used by AppendClause, hence by Minimize and Optimal) bound
// each unit at top level without checking whether propagating an earlier
// unit had already falsified it. The conflict was hidden and the solver
// returned models violating the problem, e.g. Minimize returned cost 0 with
// an invalid model where the optimum is 1.

func TestMinimizeHiddenTopLevelConflict(t *testing.T) {
	// 5 x2 + ~x1 + ~x3 >= 1, minimize 4 ~x1 + 2 x2 + ~x3. Optimum: x1, ~x2, ~x3, cost 1.
	pb := ParsePBConstrs([]PBConstr{GtEq([]int{-1, -3, 2}, []int{1, 1, 5}, 1)})
	pb.SetCostFunc(IntsToLits(-1, 2, -3), []int{4, 2, 1})
	s := New(pb)
	if got := s.Minimize(); got != 1 {
		t.Errorf("Minimize = %d, want 1", got)
	}
	if m := s.Model(); !m[0] || m[1] || m[2] {
		t.Errorf("model = %v, want [true false false]", m)
	}
}

// AppendClause of a constraint whose lits must all be true, where the first
// lits propagate the negation of a later one.
func TestAppendClauseAllUnitsConflicting(t *testing.T) {
	// x1 -> ~x3 (via x1 -> x2 -> ~x3); then require x1 + x3 >= 2.
	s := New(ParseSlice([][]int{{-1, 2}, {-2, -3}, {1, 3, 4}}))
	if st := s.Solve(); st != Sat {
		t.Fatalf("Solve = %v, want SAT", st)
	}
	s.AppendClause(NewCardClause(IntsToLits(1, 3), 2))
	if st := s.Solve(); st != Unsat {
		t.Errorf("Solve after AppendClause = %v, want UNSAT", st)
	}
}

// Optimal goes through the same path: check the full stream of results.
func TestOptimalRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(11))
	for it := 0; it < iters; it++ {
		n := r.Intn(7) + 2
		constrs, cs := randPBConstrs(r, n, r.Intn(2*n)+1)
		nl := r.Intn(n) + 1
		olits := make([]Lit, nl)
		ow := make([]int, nl)
		for j, p := range r.Perm(n)[:nl] {
			olits[j] = IntToLit(int32(randSign(r) * (p + 1)))
			ow[j] = r.Intn(4) + 1
		}
		cost := func(m []bool) int {
			c := 0
			for j, l := range olits {
				if m[l.Var()] == l.IsPositive() {
					c += ow[j]
				}
			}
			return c
		}
		best := -1
		bfModels(n, cs, nil, func(a uint64) {
			m := make([]bool, n)
			for v := range m {
				m[v] = bit(a, v+1)
			}
			if c := cost(m); best == -1 || c < best {
				best = c
			}
		})
		desc := fmt.Sprintf("n=%d %v obj=%v w=%v", n, cs, olits, ow)
		pb := padVars(ParsePBConstrs(constrs), n)
		pb.SetCostFunc(olits, ow)
		results := make(chan Result, 1000)
		res := New(pb).Optimal(results, nil)
		prev := -1
		for r := range results {
			if r.Status != Sat {
				continue
			}
			if msg := checkModel(n, cs, nil, r.Model); msg != "" {
				t.Fatalf("intermediate result invalid: %s: %s", msg, desc)
			}
			if c := cost(r.Model); c != r.Weight {
				t.Fatalf("result weight %d but model costs %d: %s", r.Weight, c, desc)
			}
			if prev != -1 && r.Weight >= prev {
				t.Fatalf("costs not strictly decreasing (%d then %d): %s", prev, r.Weight, desc)
			}
			prev = r.Weight
		}
		if best == -1 {
			if res.Status != Unsat {
				t.Fatalf("Optimal status %v, want UNSAT: %s", res.Status, desc)
			}
			continue
		}
		if res.Status != Sat || res.Weight != best {
			t.Fatalf("Optimal = %v/%d, want SAT/%d: %s", res.Status, res.Weight, best, desc)
		}
	}
}

func randSign(r *rand.Rand) int {
	if r.Intn(2) == 0 {
		return -1
	}
	return 1
}
