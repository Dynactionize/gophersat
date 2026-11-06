package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// watchClause tested card == Len()+1 (an unsatisfiable constraint) instead
// of card == Len()-1 to select the at-most-one-false fast path, which was
// therefore never used. These tests exercise that path.

func TestAtMostOneFalseUsesFastPath(t *testing.T) {
	// x1+x2+x3+x4 >= 3: at most one lit false.
	s := New(ParseCardConstrs([]CardConstr{{Lits: []int{1, 2, 3, 4}, AtLeast: 3}, {Lits: []int{1, 5, 6}, AtLeast: 2}}))
	nb := 0
	for _, lst := range s.wl.wlistCardAMO {
		nb += len(lst)
	}
	if nb != 4+3 {
		t.Fatalf("%d AMO watches, want 7 (one per lit)", nb)
	}
	// Falsifying x1 must propagate x2, x3, x4 and x5, x6.
	if st := s.Assume(IntsToLits(-1)); st == Unsat {
		t.Fatal("UNSAT after assuming ¬x1")
	}
	for v := 2; v <= 6; v++ {
		if s.litStatus(IntToLit(int32(v))) != Sat {
			t.Errorf("x%d not propagated to true", v)
		}
	}
	if st := s.Assume(IntsToLits(-2, -3)); st != Unsat {
		t.Errorf("assuming ¬x2, ¬x3: got %v, want UNSAT", st)
	}
}

// Random problems made mostly of "at most one false" constraints, through
// Solve (both modes), Assume sessions and CountModels.
func TestAtMostOneFalseRandomVsBruteForce(t *testing.T) {
	iters := 2000
	if testing.Short() {
		iters = 200
	}
	r := rand.New(rand.NewSource(32))
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 3
		var constrs []CardConstr
		var cs []bfConstr
		for k := r.Intn(n) + 1; k > 0; k-- {
			l := r.Intn(n-2) + 3
			lits := make([]int, l)
			ones := make([]int, l)
			for j, p := range r.Perm(n)[:l] {
				lits[j] = (p + 1) * (2*r.Intn(2) - 1)
				ones[j] = 1
			}
			card := l - 1
			if r.Intn(4) == 0 {
				card = r.Intn(l) + 1 // some other cardinalities too
			}
			constrs = append(constrs, CardConstr{Lits: append([]int(nil), lits...), AtLeast: card})
			cs = append(cs, bfConstr{lits, ones, card})
		}
		for _, c := range randCNF(r, n, r.Intn(n)) {
			constrs = append(constrs, AtLeast1(c...))
			cs = append(cs, cnfConstrs([][]int{c})...)
		}
		desc := fmt.Sprintf("n=%d %v", n, cs)
		want := bfCount(n, cs, nil)
		for _, cp := range []bool{false, true} {
			s := New(padVars(ParseCardConstrs(constrs), n))
			s.CuttingPlanes = cp
			st := s.Solve()
			if (st == Sat) != (want > 0) {
				t.Fatalf("cp=%v: got %v, want %d models: %s", cp, st, want, desc)
			}
			if st == Sat {
				if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
					t.Fatalf("cp=%v: %s: %s", cp, msg, desc)
				}
			}
		}
		if got := New(padVars(ParseCardConstrs(constrs), n)).CountModels(); got != want {
			t.Fatalf("CountModels = %d, want %d: %s", got, want, desc)
		}
		s := New(padVars(ParseCardConstrs(constrs), n))
		for round := 0; round < 4; round++ {
			x := randLitInt(r, n)
			fixed := map[int]bool{max(x, -x): x > 0}
			st := s.Assume(IntsToLits(int32(x)))
			if st != Unsat {
				st = s.Solve()
			}
			if (st == Sat) != (bfCount(n, cs, fixed) > 0) {
				t.Fatalf("Assume(%d): got %v: %s", x, st, desc)
			}
		}
	}
}
