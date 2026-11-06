package explain

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// Brute-force helpers for small CNFs (<= 12 vars).

func bfSat(nbVars int, clauses [][]int) bool {
	for a := 0; a < 1<<nbVars; a++ {
		ok := true
		for _, c := range clauses {
			sat := false
			for _, lit := range c {
				v := max(lit, -lit)
				if (a>>(v-1)&1 == 1) == (lit > 0) {
					sat = true
					break
				}
			}
			if !sat {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func randClauses(r *rand.Rand, nbVars, nbClauses int) [][]int {
	res := make([][]int, nbClauses)
	for i := range res {
		l := min(r.Intn(3)+1, nbVars)
		for _, p := range r.Perm(nbVars)[:l] {
			lit := p + 1
			if r.Intn(2) == 0 {
				lit = -lit
			}
			res[i] = append(res[i], lit)
		}
	}
	return res
}

func cnfText(nbVars int, clauses [][]int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "p cnf %d %d\n", nbVars, len(clauses))
	for _, c := range clauses {
		for _, lit := range c {
			fmt.Fprintf(&sb, "%d ", lit)
		}
		sb.WriteString("0\n")
	}
	return sb.String()
}

func clauseKey(c []int) string { return fmt.Sprint(c) }

// checkMUS verifies that mus is a minimal unsatisfiable subset of orig.
func checkMUS(t *testing.T, desc string, nbVars int, orig, mus [][]int) {
	t.Helper()
	avail := map[string]int{}
	for _, c := range orig {
		avail[clauseKey(c)]++
	}
	for _, c := range mus {
		if avail[clauseKey(c)] == 0 {
			t.Fatalf("%s: MUS clause %v is not an original clause (orig=%v, mus=%v)", desc, c, orig, mus)
		}
		avail[clauseKey(c)]--
	}
	if bfSat(nbVars, mus) {
		t.Fatalf("%s: MUS %v is satisfiable (orig=%v)", desc, mus, orig)
	}
	for i := range mus {
		rest := append(append([][]int{}, mus[:i]...), mus[i+1:]...)
		if !bfSat(nbVars, rest) {
			t.Fatalf("%s: MUS %v is not minimal: still UNSAT without %v (orig=%v)", desc, mus, mus[i], orig)
		}
	}
}

func mustParse(t *testing.T, text string) *Problem {
	t.Helper()
	pb, err := ParseCNF(strings.NewReader(text))
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	return pb
}
