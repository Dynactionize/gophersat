package bf

import "testing"

// and.nnf simplified a conjunction with no remaining subformula to False and
// or.nnf simplified an empty disjunction to True: the identities were
// swapped. Solve(And(True)) and Solve(Not(Or(False))) were UNSAT, and
// Solve(Or(False)) was SAT.

func TestEmptyConnectivesAfterSimplification(t *testing.T) {
	a, b := Var("a"), Var("b")
	cases := []struct {
		f    Formula
		want bool // satisfiable?
	}{
		{And(), true},
		{Or(), false},
		{And(True), true},
		{And(True, True), true},
		{Or(False), false},
		{Or(False, False), false},
		{Not(Or(False)), true},
		{Not(And(True)), false},
		{And(a, And(True)), true},
		{And(a, Or(False)), false},
		{Or(a, Or(False)), true},
		{Or(Not(a), And()), true},
		{And(a, Not(a), Or(b, And(True))), false},
		{Implies(Or(False), a), true},
		{Implies(And(True), And(a, Not(a))), false},
		{Eq(And(), Or()), false},
		{Xor(And(), Or()), true},
	}
	for _, c := range cases {
		if got := Solve(c.f) != nil; got != c.want {
			t.Errorf("Solve(%v): sat=%v, want %v", c.f, got, c.want)
		}
		checkSolve(t, c.f)
	}
}
