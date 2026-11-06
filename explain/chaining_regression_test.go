package explain

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// The Problems returned by MUSDeletion, MUSInsertion and UnsatSubset had no
// unit table, so calling a method on them panicked: pb.MUS() followed by
// .UnsatSubset() gave "index out of range". Their clauses also aliased the
// original problem's clauses.

func TestChainedExplainCalls(t *testing.T) {
	iters := 300
	if testing.Short() {
		iters = 50
	}
	r := rand.New(rand.NewSource(20))
	type step struct {
		name string
		f    func(*Problem) (*Problem, error)
	}
	steps := []step{
		{"MUS", (*Problem).MUS},
		{"MUSDeletion", (*Problem).MUSDeletion},
		{"MUSInsertion", (*Problem).MUSInsertion},
		{"UnsatSubset", (*Problem).UnsatSubset},
	}
	done := 0
	for done < iters {
		nbVars := r.Intn(6) + 1
		clauses := randClauses(r, nbVars, r.Intn(4*nbVars)+2)
		if bfSat(nbVars, clauses) {
			continue
		}
		done++
		pb := mustParse(t, cnfText(nbVars, clauses))
		cur := pb
		desc := fmt.Sprintf("%v:", clauses)
		for k := 0; k < 4; k++ {
			st := steps[r.Intn(len(steps))]
			desc += " " + st.name
			func() {
				defer func() {
					if e := recover(); e != nil {
						t.Fatalf("%s: panic %v", desc, e)
					}
				}()
				res, err := st.f(cur)
				if err != nil {
					t.Fatalf("%s: %v", desc, err)
				}
				if bfSat(nbVars, res.Clauses) {
					t.Fatalf("%s: result %v is satisfiable", desc, res.Clauses)
				}
				if st.name != "UnsatSubset" {
					checkMUS(t, desc, nbVars, clauses, res.Clauses)
				}
				// Certificate checking on the result must not panic.
				if _, err := res.Unsat(strings.NewReader("c no clause\n")); err != nil {
					t.Fatalf("%s: Unsat: %v", desc, err)
				}
				cur = res
			}()
		}
		// Mutating the results must not affect the original problem.
		before := fmt.Sprint(pb.Clauses)
		for _, c := range cur.Clauses {
			for i := range c {
				c[i] = -c[i]
			}
		}
		if after := fmt.Sprint(pb.Clauses); after != before {
			t.Fatalf("%s: result clauses alias the original problem", desc)
		}
	}
}
