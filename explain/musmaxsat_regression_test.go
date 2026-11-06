package explain

import (
	"fmt"
	"math/rand"
	"testing"
)

// MUSMaxSat returned the union of the clauses found in successive minimal
// correction sets, which is unsatisfiable but not minimal: for
// "1 / -1 / 2 / -2" it returned all 4 clauses.

func TestMUSMaxSatIsMinimal(t *testing.T) {
	clauses := [][]int{{1}, {-1}, {2}, {-2}}
	mus, err := mustParse(t, cnfText(2, clauses)).MUSMaxSat()
	if err != nil {
		t.Fatal(err)
	}
	checkMUS(t, "two independent conflicts", 2, clauses, mus.Clauses)
}

func TestMUSMaxSatRandomVsBruteForce(t *testing.T) {
	iters := 400
	if testing.Short() {
		iters = 60
	}
	r := rand.New(rand.NewSource(23))
	done := 0
	for done < iters {
		nbVars := r.Intn(6) + 1
		clauses := randClauses(r, nbVars, r.Intn(4*nbVars)+2)
		if bfSat(nbVars, clauses) {
			if _, err := mustParse(t, cnfText(nbVars, clauses)).MUSMaxSat(); err == nil {
				t.Fatalf("%v: MUSMaxSat of a SAT problem returned no error", clauses)
			}
			continue
		}
		done++
		mus, err := mustParse(t, cnfText(nbVars, clauses)).MUSMaxSat()
		if err != nil {
			t.Fatalf("%v: %v", clauses, err)
		}
		checkMUS(t, fmt.Sprintf("MUSMaxSat on %v", clauses), nbVars, clauses, mus.Clauses)
	}
}
