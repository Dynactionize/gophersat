package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// Two cutting-planes bugs in learn_pb.go:
//
//  1. When resolution derived a constraint that no assignment can satisfy
//     (e.g. "x2 >= 2"), the analysis kept looking for a falsified literal on
//     the trail, walked past its start and panicked with index -1.
//  2. When the walk back reached the decision literal of a lower level, the
//     analysis "resolved" it although it has no reason, decremented the
//     level and could end in a bogus top-level conflict: wrong UNSAT.

func solveCP(t *testing.T, constrs []PBConstr, n int, cp bool) (st Status, model []bool) {
	t.Helper()
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("cp=%v: panic %v", cp, e)
		}
	}()
	s := New(padVars(ParsePBConstrs(constrs), n))
	s.CuttingPlanes = cp
	st = s.Solve()
	if st == Sat {
		model = s.Model()
	}
	return st, model
}

func TestCuttingPlanesCounterexamples(t *testing.T) {
	cases := []struct {
		name string
		n    int
		cs   []bfConstr
	}{
		{"unsat-derived-constraint-panicked", 6, []bfConstr{
			{[]int{-6, -3, -5}, []int{1, 1, 1}, 2},
			{[]int{6, 2, 5, 3}, []int{1, 1, 1, 1}, 3},
		}},
		{"unsat-derived-constraint-panicked-2", 7, []bfConstr{
			{[]int{-1, 5, 4, -7, -2}, []int{1, 1, 1, 1, 1}, 3},
			{[]int{-4, -5, -1, 7, 2, 3, 6}, []int{1, 1, 1, 1, 1, 1, 1}, 6},
		}},
		{"resolved-decision-wrong-unsat", 7, []bfConstr{
			{[]int{3, 2, 5, -4}, []int{1, 1, 1, 1}, 3},
			{[]int{4, -7, -5, -3, 1, 2}, []int{1, 1, 1, 1, 1, 1}, 4},
		}},
		{"resolved-decision-wrong-unsat-2", 8, []bfConstr{
			{[]int{-5, -7, 2, 8, -6, -4, 1, -3}, []int{1, 1, 1, 1, 1, 1, 1, 1}, 5},
			{[]int{6, 5, -8, 3}, []int{1, 1, 1, 1}, 2},
			{[]int{7}, []int{3}, 1},
		}},
	}
	for _, c := range cases {
		for _, cp := range []bool{false, true} {
			constrs := make([]PBConstr, len(c.cs))
			for i, bc := range c.cs {
				constrs[i] = GtEq(append([]int(nil), bc.lits...), append([]int(nil), bc.ws...), bc.k)
			}
			want := bfCount(c.n, c.cs, nil) > 0
			st, m := solveCP(t, constrs, c.n, cp)
			if (st == Sat) != want {
				t.Errorf("%s cp=%v: got %v, want sat=%v", c.name, cp, st, want)
			}
			if st == Sat {
				if msg := checkModel(c.n, c.cs, nil, m); msg != "" {
					t.Errorf("%s cp=%v: %s", c.name, cp, msg)
				}
			}
		}
	}
}

func clonePB(constrs []PBConstr) []PBConstr {
	res := make([]PBConstr, len(constrs))
	for i, c := range constrs {
		res[i] = PBConstr{Lits: append([]int(nil), c.Lits...), AtLeast: c.AtLeast}
		if c.Weights != nil {
			res[i].Weights = append([]int(nil), c.Weights...)
		}
	}
	return res
}

func TestPBSolveRandomVsBruteForce(t *testing.T) {
	iters := 10000
	if testing.Short() {
		iters = 1000
	}
	for _, cp := range []bool{false, true} {
		r := rand.New(rand.NewSource(3))
		for it := 0; it < iters; it++ {
			n := r.Intn(8) + 2
			constrs, cs := randPBConstrs(r, n, r.Intn(2*n)+1)
			want := bfCount(n, cs, nil) > 0
			st, m := solveCP(t, clonePB(constrs), n, cp)
			desc := fmt.Sprintf("cp=%v n=%d %v", cp, n, cs)
			if (st == Sat) != want {
				t.Fatalf("got %v, want sat=%v: %s", st, want, desc)
			}
			if st == Sat {
				if msg := checkModel(n, cs, nil, m); msg != "" {
					t.Fatalf("%s: %s", msg, desc)
				}
			}
		}
	}
}

func TestMinimizeRandomVsBruteForce(t *testing.T) {
	iters := 4000
	if testing.Short() {
		iters = 400
	}
	for _, cp := range []bool{false, true} {
		r := rand.New(rand.NewSource(4))
		for it := 0; it < iters; it++ {
			n := r.Intn(8) + 2
			constrs, cs := randPBConstrs(r, n, r.Intn(2*n)+1)
			nl := r.Intn(n) + 1
			olits := make([]Lit, nl)
			ow := make([]int, nl)
			for j, p := range r.Perm(n)[:nl] {
				x := p + 1
				if r.Intn(2) == 0 {
					x = -x
				}
				olits[j] = IntToLit(int32(x))
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
			desc := fmt.Sprintf("cp=%v n=%d %v obj=%v w=%v", cp, n, cs, olits, ow)
			func() {
				defer func() {
					if e := recover(); e != nil {
						t.Fatalf("panic %v: %s", e, desc)
					}
				}()
				pb := padVars(ParsePBConstrs(clonePB(constrs)), n)
				pb.SetCostFunc(append([]Lit(nil), olits...), append([]int(nil), ow...))
				s := New(pb)
				s.CuttingPlanes = cp
				got := s.Minimize()
				if got != best {
					t.Fatalf("Minimize = %d, want %d: %s", got, best, desc)
				}
				if got >= 0 {
					m := s.Model()
					if msg := checkModel(n, cs, nil, m); msg != "" {
						t.Fatalf("%s: %s", msg, desc)
					}
					if c := cost(m); c != got {
						t.Fatalf("model cost %d, Minimize returned %d: %s", c, got, desc)
					}
				}
			}()
		}
	}
}
