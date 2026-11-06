package bf

import (
	"fmt"
	"math/rand"
	"testing"
)

// In cnfRec, for an "or" containing an "and", the dummy var d (d -> each
// conjunct) was appended to the first clause returned for the conjunct.
// When the conjunct was itself an "or" with a nested "and", that first clause
// was a definitional clause of the nested and, not the conjunct's main
// disjunction, which then became unconditional: satisfiable formulas were
// reported UNSAT.

func TestNestedOrAndOrAnd(t *testing.T) {
	a, b, c, d, e := Var("a"), Var("b"), Var("c"), Var("d"), Var("e")
	f := And(a, Not(c), Not(d), Not(e), Or(a, And(b, Or(c, And(d, e)))))
	checkSolve(t, f)
	if Solve(f) == nil {
		t.Fatalf("Solve(%v) = UNSAT, but a=1 and every other var 0 satisfies it", f)
	}
}

var names = []string{"a", "b", "c", "d", "e", "f"}

// randFormula builds a random formula of the given depth over names, using
// every connective of the package.
func randFormula(r *rand.Rand, depth int) Formula {
	if depth == 0 || r.Intn(5) == 0 {
		switch r.Intn(12) {
		case 0:
			return True
		case 1:
			return False
		default:
			return Var(names[r.Intn(len(names))])
		}
	}
	subs := func() []Formula {
		res := make([]Formula, r.Intn(4))
		for i := range res {
			res[i] = randFormula(r, depth-1)
		}
		return res
	}
	switch r.Intn(8) {
	case 0, 1:
		return And(subs()...)
	case 2, 3:
		return Or(subs()...)
	case 4:
		return Not(randFormula(r, depth-1))
	case 5:
		return Implies(randFormula(r, depth-1), randFormula(r, depth-1))
	case 6:
		return Eq(randFormula(r, depth-1), randFormula(r, depth-1))
	default:
		return Xor(randFormula(r, depth-1), randFormula(r, depth-1))
	}
}

func TestSolveRandomFormulasVsBruteForce(t *testing.T) {
	iters := 20000
	if testing.Short() {
		iters = 2000
	}
	r := rand.New(rand.NewSource(15))
	for it := 0; it < iters; it++ {
		f := randFormula(r, r.Intn(5)+1)
		t.Run(fmt.Sprint(it), func(t *testing.T) { checkSolve(t, f) })
		if t.Failed() {
			return
		}
	}
}
