package bf

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// litValue registered Unique's internal line/column vars as problem vars,
// so Solve returned them in the model (e.g. "line-0-a-b-c-d-e": true).

func TestUniqueModelHasOnlyProblemVars(t *testing.T) {
	for n := 1; n <= 30; n++ {
		vars := make([]string, n)
		for i := range vars {
			vars[i] = fmt.Sprintf("v%d", i)
		}
		m := Solve(Unique(vars...))
		if m == nil {
			t.Fatalf("Unique(%d vars) UNSAT", n)
		}
		if len(m) != n {
			t.Errorf("Unique(%d vars): model has %d entries: %v", n, len(m), m)
		}
		nbTrue := 0
		for k, b := range m {
			if !strings.HasPrefix(k, "v") {
				t.Errorf("Unique(%d vars): unexpected var %q in model", n, k)
			}
			if b {
				nbTrue++
			}
		}
		if nbTrue != 1 {
			t.Errorf("Unique(%d vars): %d vars true, want 1", n, nbTrue)
		}
	}
}

// Unique combined with random constraints on the same vars: exactly one var
// true, and the other constraints hold.
func TestUniqueWithConstraintsVsBruteForce(t *testing.T) {
	r := rand.New(rand.NewSource(16))
	for it := 0; it < 2000; it++ {
		n := r.Intn(9) + 1
		vars := make([]string, n)
		for i := range vars {
			vars[i] = fmt.Sprintf("v%d", i)
		}
		// forbid a random subset of vars: SAT iff at least one is allowed
		var extra []Formula
		allowed := 0
		for _, v := range vars {
			if r.Intn(3) == 0 {
				extra = append(extra, Not(Var(v)))
			} else {
				allowed++
			}
		}
		f := And(append(extra, Unique(vars...))...)
		m := Solve(f)
		if (m != nil) != (allowed > 0) {
			t.Fatalf("%v: sat=%v, want %v", f, m != nil, allowed > 0)
		}
		if m == nil {
			continue
		}
		nbTrue := 0
		for _, v := range vars {
			if m[v] {
				nbTrue++
			}
		}
		if nbTrue != 1 || len(m) != n {
			t.Fatalf("%v: model %v", f, m)
		}
		for _, e := range extra {
			if !e.Eval(m) {
				t.Fatalf("%v: model %v violates %v", f, m, e)
			}
		}
	}
}
