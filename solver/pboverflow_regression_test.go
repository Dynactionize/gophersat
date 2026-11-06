package solver

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// Huge PB coefficients made WeightSum overflow and wrap negative, so a
// satisfiable constraint was declared UNSAT. Independently, a Clause stores
// its cardinality on 30 bits, so any right-hand side >= 2^30 was silently
// truncated (e.g. wrong SAT with cutting planes). Such input is now rejected
// (error from ParseOPB, panic from ParsePBConstrs / NewPBClause) instead of
// being solved wrongly; weights are saturated at the cardinality, and values
// up to MaxCardinality are solved correctly.

func TestParseOPBRejectsOverflowingCoefficients(t *testing.T) {
	m := math.MaxInt
	inputs := []string{
		fmt.Sprintf("%d x1 +%d x2 >= %d ;", m, m, m),
		fmt.Sprintf("%d x1 +%d x2 +%d x3 >= 1 ;", 1<<62, 1<<62, 1<<62),
		fmt.Sprintf("%d x1 +%d x2 = %d ;", m/2, m/2, 1),
		fmt.Sprintf("-%d x1 +%d x2 >= -%d ;", m, 1, m),
		fmt.Sprintf("1 x1 >= %d ;", math.MinInt),
		fmt.Sprintf("%d x1 +%d x2 >= %d ;", MaxCardinality, MaxCardinality, MaxCardinality+1),
		fmt.Sprintf("%d x1 +%d x2 = %d ;", MaxCardinality+1, 1, 1), // <= side normalizes above the limit
	}
	for _, in := range inputs {
		if pb, err := ParseOPB(strings.NewReader(in + "\n")); err == nil {
			t.Errorf("%q: accepted (status after solving: %v), want an error", in, New(pb).Solve())
		}
	}
}

func TestParsePBConstrsPanicsOnOverflowingCoefficients(t *testing.T) {
	for _, c := range []PBConstr{
		{Lits: []int{1, 2}, Weights: []int{math.MaxInt, math.MaxInt}, AtLeast: math.MaxInt},
		{Lits: []int{1, 2}, Weights: []int{MaxCardinality, 1}, AtLeast: MaxCardinality + 1},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%v: no panic", c)
				}
			}()
			ParsePBConstrs([]PBConstr{c})
		}()
	}
	for _, f := range []func(){
		func() { NewPBClause(IntsToLits(1, 2), []int{1, 1}, MaxCardinality+1) },
		func() { NewCardClause(IntsToLits(1, 2), MaxCardinality+1) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("clause constructor accepted a cardinality above MaxCardinality")
				}
			}()
			f()
		}()
	}
}

// Cardinality must round-trip exactly up to MaxCardinality.
func TestClauseCardinalityRoundTrip(t *testing.T) {
	for _, card := range []int{1, 2, 1 << 20, MaxCardinality - 1, MaxCardinality} {
		c := NewPBClause(IntsToLits(1, 2), []int{card, card}, card)
		if got := c.Cardinality(); got != card {
			t.Errorf("Cardinality() = %d, want %d", got, card)
		}
		if c.Learned() || c.isLocked() {
			t.Errorf("card %d: flags corrupted", card)
		}
	}
}

// Large but non-overflowing coefficients keep working.
func TestLargeCoefficientsStillSolved(t *testing.T) {
	big := MaxCardinality / 2
	cases := []struct {
		opb  string
		want Status
	}{
		{fmt.Sprintf("%d x1 +%d x2 >= %d ;\n1 ~x1 >= 1 ;\n", big, big, big), Sat},
		{fmt.Sprintf("%d x1 +%d x2 >= %d ;\n1 ~x1 >= 1 ;\n", big, big, big+1), Unsat},
		{fmt.Sprintf("%d x1 +%d x2 +%d x3 >= %d ;\n1 ~x1 >= 1 ;\n", big, big, 1, big+1), Sat},
		{fmt.Sprintf("%d x1 +%d x2 = %d ;\n", big, big-1, big-1), Sat},
		{fmt.Sprintf("%d x1 +%d x2 >= %d ;\n1 ~x2 >= 1 ;\n", math.MaxInt/4, 3, MaxCardinality), Sat}, // saturated weight
		{fmt.Sprintf("%d x1 +%d x2 +%d x3 >= %d ;\n1 ~x1 >= 1 ;\n", MaxCardinality, MaxCardinality-1, 1, MaxCardinality), Sat},
		{fmt.Sprintf("%d x1 +%d x2 +%d x3 >= %d ;\n1 ~x1 >= 1 ;\n1 ~x3 >= 1 ;\n", MaxCardinality, MaxCardinality-1, 1, MaxCardinality), Unsat},
	}
	for _, c := range cases {
		if got := opbStatus(t, c.opb); got != c.want {
			t.Errorf("%q: got %v, want %v", c.opb, got, c.want)
		}
		pb, _ := ParseOPB(strings.NewReader(c.opb))
		s := New(pb)
		s.CuttingPlanes = true
		if got := s.Solve(); got != c.want {
			t.Errorf("%q with cutting planes: got %v, want %v", c.opb, got, c.want)
		}
	}
}
