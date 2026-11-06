package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// PB constraints whose weights are all equal are built as cardinality
// constraints (or clauses), which propagate much more cheaply.

func TestEqualWeightsBuildCardinalityClauses(t *testing.T) {
	cases := []struct {
		weights  []int
		atLeast  int
		pb       bool // still a general PB constraint?
		wantCard int
	}{
		{[]int{1, 1, 1, 1}, 2, false, 2},
		{[]int{3, 3, 3, 3}, 7, false, 3}, // ceil(7/3)
		{[]int{2, 2, 2}, 1, false, 1},    // saturated weights: a clause
		{[]int{1, 2, 1}, 2, true, 2},
	}
	for _, c := range cases {
		pb := ParsePBConstrs([]PBConstr{GtEq([]int{1, 2, 3, 4}[:len(c.weights)], c.weights, c.atLeast), PropClause(-1, -2)})
		var cl *Clause
		for _, x := range pb.Clauses {
			if x.Len() > 2 {
				cl = x
			}
		}
		if cl == nil {
			t.Fatalf("%v >= %d: constraint not found in %s", c.weights, c.atLeast, pb.PBString())
		}
		if cl.PseudoBoolean() != c.pb || cl.Cardinality() != c.wantCard {
			t.Errorf("%v >= %d: got %s (PB=%v), want PB=%v card %d", c.weights, c.atLeast, cl.PBString(), cl.PseudoBoolean(), c.pb, c.wantCard)
		}
	}
}

// Random problems made of equal-weight constraints, through both parsers
// and both learning modes, checked against brute force.
func TestEqualWeightsRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(31))
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 2
		var constrs []PBConstr
		var cs []bfConstr
		var opb strings.Builder
		for k := r.Intn(2*n) + 1; k > 0; k-- {
			l := r.Intn(n) + 1
			w := r.Intn(4) + 1
			lits := make([]int, l)
			ws := make([]int, l)
			for j, p := range r.Perm(n)[:l] {
				lits[j] = (p + 1) * (2*r.Intn(2) - 1)
				ws[j] = w
				writeOPBTerm(&opb, w, lits[j])
			}
			atLeast := r.Intn(w*l) + 1
			fmt.Fprintf(&opb, ">= %d ;\n", atLeast)
			constrs = append(constrs, GtEq(append([]int(nil), lits...), append([]int(nil), ws...), atLeast))
			cs = append(cs, bfConstr{lits, ws, atLeast})
		}
		want := bfCount(n, cs, nil) > 0
		desc := fmt.Sprintf("n=%d %v", n, cs)
		for _, cp := range []bool{false, true} {
			for _, viaOPB := range []bool{false, true} {
				var pb *Problem
				if viaOPB {
					var err error
					if pb, err = ParseOPB(strings.NewReader(opb.String())); err != nil {
						t.Fatal(err)
					}
				} else {
					pb = ParsePBConstrs(clonePB(constrs))
				}
				s := New(padVars(pb, n))
				s.CuttingPlanes = cp
				st := s.Solve()
				if (st == Sat) != want {
					t.Fatalf("cp=%v opb=%v: got %v, want sat=%v: %s", cp, viaOPB, st, want, desc)
				}
				if st == Sat {
					if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
						t.Fatalf("cp=%v opb=%v: %s: %s", cp, viaOPB, msg, desc)
					}
				}
			}
		}
	}
}
