package maxsat

import (
	"fmt"
	"math/rand"
	"testing"
)

// The blocking literal added to a soft constraint had coefficient 1 when
// Coeffs was nil (cardinality constraints) and ignored the bound shift
// caused by negative coeffs, so relaxing the constraint was impossible and
// optimal solutions were missed or the problem reported UNSAT:
//   hard ¬a, ¬b + SoftPBConstr([a,b], nil, 2)  -> UNSAT, expected cost 1
//   hard a + WeightedPBConstr([a], [-1], 0, 5) -> UNSAT, expected cost 5

func TestSoftConstraintsCanBeRelaxed(t *testing.T) {
	a, b := Var("a"), Var("b")
	cases := []struct {
		constrs []Constr
		want    int
	}{
		{[]Constr{HardClause(Not("a")), HardClause(Not("b")), SoftPBConstr([]Lit{a, b}, nil, 2)}, 1},
		{[]Constr{HardClause(a), WeightedPBConstr([]Lit{a}, []int{-1}, 0, 5)}, 5},
		{[]Constr{HardClause(a), HardClause(b), WeightedPBConstr([]Lit{a, b}, []int{-2, -3}, -4, 7)}, 7},
		{[]Constr{HardClause(Not("a")), WeightedPBConstr([]Lit{a, b}, []int{3, 1}, 3, 2), SoftClause(Not("b"))}, 2},
	}
	for i, c := range cases {
		m, cost := New(c.constrs...).Solve()
		if m == nil || cost != c.want {
			t.Errorf("case %d: got model %v cost %d, want cost %d", i, m, cost, c.want)
		}
	}
}

type bfConstr struct {
	lits    []int // 1-based var indices, negative if negated
	coeffs  []int
	atLeast int
	weight  int
}

func (c bfConstr) holds(a int) bool {
	sum := 0
	for i, l := range c.lits {
		v := max(l, -l)
		if (a>>(v-1)&1 == 1) == (l > 0) {
			sum += c.coeffs[i]
		}
	}
	return sum >= c.atLeast
}

func TestMaxSatRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(24))
	for it := 0; it < iters; it++ {
		nbVars := r.Intn(6) + 1
		var bcs []bfConstr
		var constrs []Constr
		for k := r.Intn(3*nbVars) + 1; k > 0; k-- {
			l := r.Intn(nbVars) + 1
			c := bfConstr{}
			var lits []Lit
			var coeffs []int
			kind := r.Intn(3) // 0: clause, 1: cardinality (nil coeffs), 2: PB with signed coeffs
			sumPos, sumNeg := 0, 0
			for _, p := range r.Perm(nbVars)[:l] {
				v := p + 1
				lit := Var(fmt.Sprint("x", v))
				if r.Intn(2) == 0 {
					v, lit = -v, lit.Negation()
				}
				w := 1
				if kind == 2 {
					w = r.Intn(7) - 3
				}
				if w > 0 {
					sumPos += w
				} else {
					sumNeg += w
				}
				c.lits = append(c.lits, v)
				c.coeffs = append(c.coeffs, w)
				lits = append(lits, lit)
				coeffs = append(coeffs, w)
			}
			switch kind {
			case 0:
				c.atLeast = 1
			case 1:
				c.atLeast = r.Intn(l) + 1
			default:
				c.atLeast = sumNeg + r.Intn(sumPos-sumNeg+2)
			}
			if r.Intn(3) > 0 {
				c.weight = r.Intn(5) + 1
			}
			if kind != 2 {
				coeffs = nil
			}
			bcs = append(bcs, c)
			constrs = append(constrs, Constr{Lits: lits, Coeffs: coeffs, AtLeast: c.atLeast, Weight: c.weight})
		}
		best := -1
		for a := 0; a < 1<<nbVars; a++ {
			cost := 0
			ok := true
			for _, c := range bcs {
				if !c.holds(a) {
					if c.weight == 0 {
						ok = false
						break
					}
					cost += c.weight
				}
			}
			if ok && (best == -1 || cost < best) {
				best = cost
			}
		}
		desc := fmt.Sprintf("%+v", bcs)
		m, cost := New(constrs...).Solve()
		if cost != best {
			t.Fatalf("cost %d, want %d: %s", cost, best, desc)
		}
		if best == -1 {
			continue
		}
		a := 0
		for v := 1; v <= nbVars; v++ {
			if m[fmt.Sprint("x", v)] {
				a |= 1 << (v - 1)
			}
		}
		got := 0
		for _, c := range bcs {
			if !c.holds(a) {
				if c.weight == 0 {
					t.Fatalf("model %v violates hard constraint %+v: %s", m, c, desc)
				}
				got += c.weight
			}
		}
		if got != cost {
			t.Fatalf("model %v costs %d, Solve returned %d: %s", m, got, cost, desc)
		}
	}
}
