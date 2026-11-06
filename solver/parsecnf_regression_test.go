package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// ParseCNF used to:
//   - drop the last int of the input when it was directly followed by EOF
//     (readInt returned it together with io.EOF and the caller discarded it),
//     so "... -2" without final "0" or a final "0" alone lost a clause;
//   - reject blank lines / trailing spaces followed by a comment, and the
//     "%" trailer of SATLIB files;
//   - panic on a negative var count in the header.

func parseStatus(t *testing.T, cnf string) (Status, error) {
	t.Helper()
	pb, err := ParseCNF(strings.NewReader(cnf))
	if err != nil {
		return Indet, err
	}
	return New(pb).Solve(), nil
}

func TestParseCNFEdgeCases(t *testing.T) {
	cases := []struct {
		name, cnf string
		want      Status // Indet means "parse error expected"
	}{
		{"last clause without 0 at EOF", "p cnf 2 3\n1 2 0\n-1 0\n-2", Unsat},
		{"last clause without 0 at EOF, multi-digit", "p cnf 12 2\n12 0\n-12", Unsat},
		{"empty clause at EOF", "p cnf 1 1\n0", Unsat},
		{"empty clause at EOF after clause", "p cnf 1 2\n1 0\n0", Unsat},
		{"final 0 at EOF", "p cnf 1 2\n1 0\n-1 0", Unsat},
		{"trailing space then comment", "p cnf 2 2\n1 2 0 \nc comment\n-1 0\n", Sat},
		{"blank lines", "p cnf 2 2\n\n1 2 0\n\n\n-1 0\n\n", Sat},
		{"comment between clauses", "c head\np cnf 2 3\n1 0\nc mid\n-1 2 0\nc end\n-2 0\n", Unsat},
		{"CRLF", "p cnf 2 2\r\n1 2 0\r\n-1 0\r\n", Sat},
		{"tabs", "p cnf 2 2\n1\t2\t0\n\t-1 0\n", Sat},
		{"SATLIB trailer", "p cnf 2 2\n1 2 0\n-1 0\n%\n0\n\n", Sat},
		{"SATLIB trailer, unsat part before", "p cnf 1 2\n1 0\n-1 0\n%\n0\n", Unsat},
		{"clause spanning lines", "p cnf 3 1\n1\n2\n3 0\n", Sat},
		{"negative var count", "p cnf -1 0\n", Indet},
		{"negative clause count", "p cnf 1 -3\n", Indet},
		{"lit out of range", "p cnf 2 1\n1 3 0\n", Indet},
		{"lit out of range at EOF", "p cnf 2 1\n1 -3", Indet},
		{"garbage", "p cnf 2 1\n1 x 0\n", Indet},
		{"lone minus at EOF", "p cnf 2 1\n1 -", Indet},
	}
	for _, c := range cases {
		got, err := parseStatus(t, c.cnf)
		switch {
		case c.want == Indet && err == nil:
			t.Errorf("%s: parsed fine (%v), want an error", c.name, got)
		case c.want != Indet && err != nil:
			t.Errorf("%s: unexpected error %v", c.name, err)
		case c.want != Indet && got != c.want:
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Random CNFs written with random but valid formatting must parse to an
// equisatisfiable problem whose models satisfy the original clauses.
func TestParseCNFRandomFormatting(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(13))
	blank := func() string {
		return []string{" ", "  ", "\t", "\n", " \n", "\r\n", "\n\n"}[r.Intn(7)]
	}
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 2
		cnf := randCNF(r, n, r.Intn(4*n)+1)
		var sb strings.Builder
		if r.Intn(2) == 0 {
			sb.WriteString("c generated\n")
		}
		fmt.Fprintf(&sb, "p cnf %d %d\n", n, len(cnf))
		for i, c := range cnf {
			if r.Intn(5) == 0 {
				sb.WriteString(blank())
			}
			for _, x := range c {
				fmt.Fprintf(&sb, "%d%s", x, blank())
			}
			last := i == len(cnf)-1
			if !last || r.Intn(2) == 0 {
				sb.WriteString("0")
				if !last || r.Intn(2) == 0 {
					sb.WriteString([]string{"\n", " \n", "\r\n", "\n", " "}[r.Intn(5)])
				}
				if r.Intn(6) == 0 && strings.HasSuffix(sb.String(), "\n") { // comments start a line
					sb.WriteString("c note\n")
				}
			} else {
				text := strings.TrimRight(sb.String(), " \t\r\n")
				sb.Reset()
				sb.WriteString(text) // clause ends with its last lit at EOF
			}
		}
		if r.Intn(8) == 0 && strings.HasSuffix(sb.String(), "\n") {
			sb.WriteString("%\n0\n")
		}
		input := sb.String()
		cs := cnfConstrs(cnf)
		want := bfCount(n, cs, nil) > 0
		pb, err := ParseCNF(strings.NewReader(input))
		if err != nil {
			t.Fatalf("parse error %v on %q", err, input)
		}
		s := New(pb)
		st := s.Solve()
		if (st == Sat) != want {
			t.Fatalf("got %v, want sat=%v on %q", st, want, input)
		}
		if st == Sat {
			if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
				t.Fatalf("%s on %q", msg, input)
			}
		}
	}
}
