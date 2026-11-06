package solver

import (
	"fmt"
	"strings"
	"testing"
)

// ParseOPB rejected valid files: whitespace after the final ";" ("does not
// end with semicolon"), indented lines, and any line longer than
// bufio.Scanner's default 64KB ("token too long"), e.g. a big objective.

func TestParseOPBWhitespaceAroundLines(t *testing.T) {
	cases := []struct {
		opb  string
		want Status
	}{
		{"1 x1 +1 x2 >= 1 ; \n1 ~x1 >= 1 ;\t\n", Sat},
		{"1 x1 >= 1 ;   \r\n1 ~x1 >= 1 ;\r\n", Unsat},
		{"  1 x1 >= 1 ;\n\t* indented comment\n   \n1 ~x1 +1 x2 >= 1 ;\n", Sat},
		{"min: 1 x1 ; \n1 x1 +1 x2 >= 1 ;\n", Sat},
	}
	for _, c := range cases {
		pb, err := ParseOPB(strings.NewReader(c.opb))
		if err != nil {
			t.Errorf("%q: %v", c.opb, err)
			continue
		}
		if got := New(pb).Solve(); got != c.want {
			t.Errorf("%q: got %v, want %v", c.opb, got, c.want)
		}
	}
}

func TestParseOPBLongLines(t *testing.T) {
	const n = 20000 // ~10 bytes per term: lines of ~200KB
	var obj, constr strings.Builder
	obj.WriteString("min:")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&obj, " +%d x%d", i%7+1, i)
		fmt.Fprintf(&constr, "+1 x%d ", i)
	}
	opb := obj.String() + " ;\n" + constr.String() + ">= 3 ;\n"
	pb, err := ParseOPB(strings.NewReader(opb))
	if err != nil {
		t.Fatalf("long lines: %v", err)
	}
	if pb.NbVars != n || len(pb.minLits) != n {
		t.Fatalf("parsed %d vars and %d objective lits, want %d", pb.NbVars, len(pb.minLits), n)
	}
	sumW := 0
	for _, w := range pb.minWeights {
		sumW += w
	}
	if want := n * 4; sumW < want-7 || sumW > want+7 { // weights cycle through 1..7
		t.Errorf("objective weights sum to %d, want about %d", sumW, want)
	}
	s := New(pb)
	if st := s.Solve(); st != Sat {
		t.Fatalf("Solve = %v, want SAT", st)
	}
	nbTrue := 0
	for _, b := range s.Model() {
		if b {
			nbTrue++
		}
	}
	if nbTrue < 3 {
		t.Errorf("model has %d true vars, constraint needs 3", nbTrue)
	}
}
