package solver

import (
	"strings"
	"testing"
)

// ParseOPB panicked instead of returning an error on malformed terms
// (weight without variable, x0, negative var ids, bad names in "min:"), and
// on trivially satisfied constraints such as "... >= 0".

func TestParseOPBMalformedInputReturnsError(t *testing.T) {
	inputs := []string{
		"3 >= 1 ;",
		"1 x1 +2 >= 1 ;",
		"min: 1 x1 foo ;",
		"min: 1 x1 2 ;",
		"1 x0 >= 1 ;",
		"1 ~x0 >= 1 ;",
		"1 x1 +1 x-5 >= 1 ;",
		"1 ~x-1 >= 1 ;",
		"1 x >= 1 ;",
		"1 ~ >= 1 ;",
		"1 y1 >= 1 ;",
		"1 x99999999999 >= 1 ;",
		"1 x1 >= ;",
		"1 x1 <= 1 ;",
		"1 x1 >= 1",
		";",
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("%q: panic %v", in, e)
				}
			}()
			if _, err := ParseOPB(strings.NewReader(in + "\n")); err == nil {
				t.Errorf("%q: parsed without error", in)
			}
		}()
	}
}

func TestParseOPBTriviallySatisfiedConstraints(t *testing.T) {
	cases := []struct {
		opb  string
		want Status
	}{
		{"1 x1 +1 x2 >= 0 ;\n", Sat},
		{"1 x1 +1 x2 >= -3 ;\n1 ~x1 >= 1 ;\n", Sat},
		{"2 x1 +1 x2 >= 0 ;\n1 x1 >= 1 ;\n1 ~x1 >= 1 ;\n", Unsat},
		{"x1 x2 >= 0 ;\nx1 ~x2 >= 2 ;\n", Sat}, // weightless lits
		{"min: 1 x1 ;\n1 x1 +1 x2 >= 0 ;\n", Sat},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("%q: panic %v", c.opb, e)
				}
			}()
			if got := opbStatus(t, c.opb); got != c.want {
				t.Errorf("%q: got %v, want %v", c.opb, got, c.want)
			}
		}()
	}
}
