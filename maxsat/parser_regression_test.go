package maxsat

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/dynactionize/gophersat/solver"
)

// ParseWCNF:
//   - did not check var ids against the header, so they could collide with
//     the relax vars it creates ("p wcnf 1 2 10 / 10 2 0 / 1 -1 0" reported
//     cost 1 with model -x1, the optimum being 0);
//   - never checked the final 0 and overwrote the last field ("10 1 2"
//     silently dropped literal 2);
//   - panicked on whitespace-only and one-field lines, on a header
//     announcing more vars than used, and without a header;
//   - rejected lines over 64KB.

func TestParseWCNFInvalidInput(t *testing.T) {
	inputs := []string{
		"p wcnf 1 2 10\n10 2 0\n1 -1 0\n", // var 2 > nbvars: would collide with relax var
		"p wcnf 2 1 10\n10 1 2\n",         // no final 0
		"p wcnf 2 1 10\n10 1 0 2\n",
		"p wcnf 2 1 10\n5\n", // one field
		"p wcnf 2 1 10\n0\n",
		"p wcnf 2 1 10\n-3 1 0\n", // negative weight
		"p wcnf 2 1 10\n3 1 x 0\n",
		"1 2 0\n", // no header
		"10 1 0\n",
		"p wcnf -1 1\n",
		"p wcnf 2 1 0\n",
		"p wcnf 2\n",
		"p wcnf 2 1 10\np wcnf 2 1 10\n",
		"",
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("%q: panic %v", in, e)
				}
			}()
			if _, err := ParseWCNF(strings.NewReader(in)); err == nil {
				t.Errorf("%q: parsed without error", in)
			}
		}()
	}
}

func TestParseWCNFValidEdgeCases(t *testing.T) {
	cases := []struct {
		wcnf string
		want int // optimal cost, -1 for UNSAT
	}{
		{"p wcnf 5 1 10\n10 1 0\n", 0},                  // header announces more vars than used
		{"p wcnf 3 2\n1 1 0\n1 -1 0\n", 1},              // no top weight: all soft
		{"p wcnf 2 2 10\n   \n10 1 0\n\t3 -1 0  \n", 3}, // blank and indented lines
		{"c x\np wcnf 2 3 10\n10 1 0\n10 -1 0\n1 2 0\n", -1},
		{"p wcnf 2 1 10\n4 0\n", 4}, // empty soft clause: always violated
	}
	for _, c := range cases {
		s, err := ParseWCNF(strings.NewReader(c.wcnf))
		if err != nil {
			t.Errorf("%q: %v", c.wcnf, err)
			continue
		}
		res := s.Optimal(nil, nil)
		got := res.Weight
		if res.Status == solver.Unsat {
			got = -1
		}
		if got != c.want {
			t.Errorf("%q: cost %d (%v), want %d", c.wcnf, got, res.Status, c.want)
		}
	}
}

func TestParseWCNFLongLines(t *testing.T) {
	const n = 30000
	var sb strings.Builder
	fmt.Fprintf(&sb, "p wcnf %d 2 100\n100 ", n)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "%d ", i)
	}
	sb.WriteString("0\n1 -1 0\n")
	s, err := ParseWCNF(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatal(err)
	}
	if res := s.Optimal(nil, nil); res.Status != solver.Sat || res.Weight != 0 {
		t.Errorf("got %v cost %d, want SAT cost 0", res.Status, res.Weight)
	}
}

// Random WCNF files checked against the brute-force optimum.
func TestParseWCNFRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(25))
	for it := 0; it < iters; it++ {
		nbVars := r.Intn(6) + 1
		declared := nbVars + r.Intn(3) // header may announce unused vars
		top := 100
		var sb strings.Builder
		type cl struct {
			lits   []int
			weight int
		}
		var cls []cl
		m := r.Intn(3*nbVars) + 1
		fmt.Fprintf(&sb, "p wcnf %d %d %d\n", declared, m, top)
		for k := 0; k < m; k++ {
			c := cl{weight: top}
			if r.Intn(3) > 0 {
				c.weight = r.Intn(9) + 1
			}
			fmt.Fprintf(&sb, "%d ", c.weight)
			for _, p := range r.Perm(nbVars)[:r.Intn(min(nbVars, 3))+1] {
				lit := p + 1
				if r.Intn(2) == 0 {
					lit = -lit
				}
				c.lits = append(c.lits, lit)
				fmt.Fprintf(&sb, "%d ", lit)
			}
			sb.WriteString("0\n")
			cls = append(cls, c)
		}
		best := -1
		for a := 0; a < 1<<nbVars; a++ {
			cost, ok := 0, true
			for _, c := range cls {
				sat := false
				for _, lit := range c.lits {
					if (a>>(max(lit, -lit)-1)&1 == 1) == (lit > 0) {
						sat = true
					}
				}
				if !sat {
					if c.weight >= top {
						ok = false
						break
					}
					cost += c.weight
				}
			}
			if ok && (best == -1 || cost < best) {
				best = cost
			}
		}
		s, err := ParseWCNF(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("%q: %v", sb.String(), err)
		}
		res := s.Optimal(nil, nil)
		got := res.Weight
		if res.Status == solver.Unsat {
			got = -1
		}
		if got != best {
			t.Fatalf("%q: cost %d, want %d", sb.String(), got, best)
		}
	}
}
