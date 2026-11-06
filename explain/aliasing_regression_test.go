package explain

import (
	"fmt"
	"math/rand"
	"testing"
)

// When the problem was trivially UNSAT (by unit propagation), UnsatSubset
// returned a shallow copy sharing the Clauses slice, and MUSDeletion then
// wrote its relax literals into the caller's clauses: after MUS() on
// "1 / -1 / 1 2", the problem was [[1 3] [-1 4] [1 2 5]].

func TestMUSDoesNotModifyTriviallyUnsatProblem(t *testing.T) {
	text := "p cnf 2 3\n1 0\n-1 0\n1 2 0\n"
	pb := mustParse(t, text)
	before := fmt.Sprint(pb.Clauses)
	mus, err := pb.MUS()
	if err != nil {
		t.Fatal(err)
	}
	if after := fmt.Sprint(pb.Clauses); after != before {
		t.Fatalf("MUS modified the problem: %s -> %s", before, after)
	}
	checkMUS(t, "trivial", 2, [][]int{{1}, {-1}, {1, 2}}, mus.Clauses)
}

// Random UNSAT problems (many trivially UNSAT): calling the MUS functions
// one after the other on the same Problem must leave it unchanged and give a
// valid MUS each time.
func TestMUSFunctionsLeaveProblemUnchanged(t *testing.T) {
	iters := 600
	if testing.Short() {
		iters = 100
	}
	r := rand.New(rand.NewSource(19))
	done := 0
	for done < iters {
		nbVars := r.Intn(6) + 1
		clauses := randClauses(r, nbVars, r.Intn(4*nbVars)+2)
		if bfSat(nbVars, clauses) {
			continue
		}
		done++
		pb := mustParse(t, cnfText(nbVars, clauses))
		before := fmt.Sprint(pb.Clauses)
		for _, name := range []string{"MUSDeletion", "MUSInsertion", "MUS", "UnsatSubset", "MUSDeletion"} {
			var res *Problem
			var err error
			switch name {
			case "MUSDeletion":
				res, err = pb.MUSDeletion()
			case "MUSInsertion":
				res, err = pb.MUSInsertion()
			case "MUS":
				res, err = pb.MUS()
			case "UnsatSubset":
				res, err = pb.UnsatSubset()
			}
			desc := fmt.Sprintf("%s on %v", name, clauses)
			if err != nil {
				t.Fatalf("%s: %v", desc, err)
			}
			if after := fmt.Sprint(pb.Clauses); after != before {
				t.Fatalf("%s modified the problem: %s -> %s", desc, before, after)
			}
			if name == "UnsatSubset" {
				if bfSat(nbVars, res.Clauses) {
					t.Fatalf("%s: subset %v is satisfiable", desc, res.Clauses)
				}
				continue
			}
			checkMUS(t, desc, nbVars, clauses, res.Clauses)
		}
	}
}
