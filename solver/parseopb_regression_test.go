package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// ParseOPB collected unit constraints without checking them against each
// other, and replicateUnits then overwrote earlier bindings: contradictory
// units such as "1 x1 >= 1 ; 1 ~x1 >= 1 ;" gave SAT.

func opbStatus(t *testing.T, opb string) Status {
	t.Helper()
	pb, err := ParseOPB(strings.NewReader(opb))
	if err != nil {
		t.Fatalf("parse error %v on %q", err, opb)
	}
	return New(pb).Solve()
}

func TestParseOPBContradictoryUnits(t *testing.T) {
	cases := []struct {
		opb  string
		want Status
	}{
		{"1 x1 >= 1 ;\n1 ~x1 >= 1 ;\n", Unsat},
		{"1 x1 = 1 ;\n1 x1 = 0 ;\n", Unsat},
		{"1 x1 +1 x2 >= 2 ;\n1 ~x2 >= 1 ;\n", Unsat},
		{"1 x1 +1 x2 = 2 ;\n1 x3 +1 ~x1 >= 2 ;\n", Unsat},
		{"1 x1 >= 1 ;\n1 x1 >= 1 ;\n1 x2 +1 x3 >= 1 ;\n", Sat},
		{"1 x1 = 1 ;\n1 ~x1 = 0 ;\n", Sat},
	}
	for _, c := range cases {
		if got := opbStatus(t, c.opb); got != c.want {
			t.Errorf("%q: got %v, want %v", c.opb, got, c.want)
		}
	}
}

// writeOPBTerm renders w*lit in OPB syntax.
func writeOPBTerm(sb *strings.Builder, w, lit int) {
	if lit < 0 {
		fmt.Fprintf(sb, "%d ~x%d ", w, -lit)
	} else {
		fmt.Fprintf(sb, "%d x%d ", w, lit)
	}
}

// Random OPB files mixing ">=" and "=" constraints, with many short
// constraints that turn into (possibly contradictory) units.
func TestParseOPBRandomVsBruteForce(t *testing.T) {
	iters := 4000
	if testing.Short() {
		iters = 400
	}
	r := rand.New(rand.NewSource(14))
	for it := 0; it < iters; it++ {
		n := r.Intn(7) + 2
		var sb strings.Builder
		var cs []bfConstr
		for m := r.Intn(2*n) + 1; m > 0; m-- {
			l := r.Intn(min(n, 3)) + 1
			if r.Intn(2) == 0 {
				l = r.Intn(n) + 1
			}
			lits := make([]int, l)
			ws := make([]int, l)
			sum := 0
			for j, p := range r.Perm(n)[:l] {
				lits[j] = p + 1
				if r.Intn(2) == 0 {
					lits[j] = -lits[j]
				}
				ws[j] = r.Intn(4) + 1
				sum += ws[j]
				writeOPBTerm(&sb, ws[j], lits[j])
			}
			k := r.Intn(sum + 1)
			if r.Intn(3) == 0 { // equality: >= k and <= k
				fmt.Fprintf(&sb, "= %d ;\n", k)
				neg := make([]int, l)
				for j := range lits {
					neg[j] = -lits[j]
				}
				cs = append(cs, bfConstr{lits, ws, k}, bfConstr{neg, ws, sum - k})
			} else {
				if r.Intn(10) == 0 {
					k = -r.Intn(3) // trivially satisfied
				}
				fmt.Fprintf(&sb, ">= %d ;\n", k)
				cs = append(cs, bfConstr{lits, ws, k})
			}
		}
		opb := sb.String()
		pb, err := ParseOPB(strings.NewReader(opb))
		if err != nil {
			t.Fatalf("parse error %v on %q", err, opb)
		}
		padVars(pb, n)
		want := bfCount(n, cs, nil) > 0
		s := New(pb)
		st := s.Solve()
		desc := fmt.Sprintf("%q", opb)
		if (st == Sat) != want {
			t.Fatalf("got %v, want sat=%v on %s", st, want, desc)
		}
		if st == Sat {
			if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
				t.Fatalf("%s on %s", msg, desc)
			}
		}
	}
}
