package bf

import (
	"fmt"
	"math/rand"
	"testing"
)

// For more than 4 vars, Unique was encoded with dummy line/column vars
// defined by equivalences inside the formula. That encoding is only valid
// as a positive top-level conjunct: under a negation or in a disjunction,
// the solver could violate the definitions. Not(Unique(a..e)) was SAT with
// exactly a true, and Or(x, Unique(a..e)) was SAT with ¬x, a and b true.

func TestUniqueUnderNegationAndDisjunction(t *testing.T) {
	vs := []string{"a", "b", "c", "d", "e"}
	a, b, c, d, e, x := Var("a"), Var("b"), Var("c"), Var("d"), Var("e"), Var("x")
	cases := []struct {
		f    Formula
		want bool
	}{
		{And(Not(Unique(vs...)), a, Not(b), Not(c), Not(d), Not(e)), false},
		{And(Not(Unique(vs...)), a, b), true},
		{And(Not(Unique(vs...)), Not(a), Not(b), Not(c), Not(d), Not(e)), true},
		{And(Or(x, Unique(vs...)), Not(x), a, b), false},
		{And(Or(x, Unique(vs...)), Not(x), a), true},
		{And(Implies(x, Unique(vs...)), x, Not(a), Not(b), Not(c), Not(d), Not(e)), false},
		{Eq(Unique(vs...), Unique("a", "b", "c", "d", "e", "x")), true},
		{And(Eq(Unique(vs...), Unique("a", "b", "c", "d", "e", "x")), x, a), true},
		{And(Eq(Unique(vs...), Unique("a", "b", "c", "d", "e", "x")), x, a, Not(b), Not(c), Not(d), Not(e)), false},
		{Not(Not(Unique(vs...))), true},
		{Unique(), false},
		{Not(Unique()), true},
	}
	for _, c := range cases {
		checkSolve(t, c.f)
		if got := Solve(c.f) != nil; got != c.want {
			t.Errorf("Solve(%v): sat=%v, want %v", c.f, got, c.want)
		}
	}
}

func TestUniqueEval(t *testing.T) {
	u := Unique("a", "b", "c", "d", "e", "f")
	for a := 0; a < 64; a++ {
		m := map[string]bool{}
		nb := 0
		for i, n := range []string{"a", "b", "c", "d", "e", "f"} {
			m[n] = a>>i&1 == 1
			if m[n] {
				nb++
			}
		}
		if u.Eval(m) != (nb == 1) {
			t.Fatalf("Eval(%v) = %v with %d vars true", m, u.Eval(m), nb)
		}
	}
}

// Random formulas mixing Unique (of up to 9 vars) with every connective, at
// every position, checked against brute force.
func TestUniqueInRandomFormulas(t *testing.T) {
	iters := 5000
	if testing.Short() {
		iters = 500
	}
	r := rand.New(rand.NewSource(34))
	var gen func(depth int) Formula
	gen = func(depth int) Formula {
		if depth == 0 || r.Intn(4) == 0 {
			if r.Intn(3) == 0 {
				var vs []string
				for _, p := range r.Perm(len(names))[:r.Intn(len(names))+1] {
					vs = append(vs, names[p])
				}
				return Unique(vs...)
			}
			return Var(names[r.Intn(len(names))])
		}
		switch r.Intn(5) {
		case 0:
			return Not(gen(depth - 1))
		case 1:
			return Or(gen(depth-1), gen(depth-1))
		case 2:
			return Implies(gen(depth-1), gen(depth-1))
		case 3:
			return Eq(gen(depth-1), gen(depth-1))
		default:
			return And(gen(depth-1), gen(depth-1))
		}
	}
	for it := 0; it < iters; it++ {
		f := gen(r.Intn(4) + 1)
		t.Run(fmt.Sprint(it), func(t *testing.T) { checkSolve(t, f) })
		if t.Failed() {
			return
		}
	}
}
