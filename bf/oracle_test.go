package bf

import (
	"sort"
	"testing"
)

// Brute-force oracle for formulas over the given var names.

// formulaVars returns the sorted names of the problem vars of f.
func formulaVars(f Formula) []string {
	set := map[string]bool{}
	var rec func(Formula)
	rec = func(f Formula) {
		switch f := f.(type) {
		case variable:
			set[f.name] = true
		case lit:
			set[f.v.name] = true
		case not:
			rec(f[0])
		case and:
			for _, s := range f {
				rec(s)
			}
		case or:
			for _, s := range f {
				rec(s)
			}
		}
	}
	rec(f)
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// bfSatisfiable reports whether some assignment of f's vars makes f true.
func bfSatisfiable(f Formula) bool {
	names := formulaVars(f)
	for a := 0; a < 1<<len(names); a++ {
		m := map[string]bool{}
		for i, n := range names {
			m[n] = a>>i&1 == 1
		}
		if f.Eval(m) {
			return true
		}
	}
	return false
}

// checkSolve solves f and checks the answer against brute force, and the
// returned model against f.
func checkSolve(t *testing.T, f Formula) {
	t.Helper()
	want := bfSatisfiable(f)
	m := Solve(f)
	if (m != nil) != want {
		t.Fatalf("Solve(%v): sat=%v, want %v", f, m != nil, want)
	}
	if m == nil {
		return
	}
	full := map[string]bool{}
	for _, n := range formulaVars(f) {
		full[n] = false // vars simplified away may take any value
	}
	for n, b := range m {
		full[n] = b
	}
	if !f.Eval(full) {
		t.Fatalf("Solve(%v) returned model %v that falsifies the formula", f, m)
	}
}
