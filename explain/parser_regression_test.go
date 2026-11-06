package explain

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// explain.ParseCNF:
//   - trusted the header's clause count: with an over-count, MUS() returned
//     clauses learned during checking; with an under-count, MUS() panicked;
//   - treated each line as a clause, splitting clauses that span lines;
//   - range-checked only unit clauses, so "p cnf 1 2 / 1 2 0 / -1 0" was
//     accepted and Unsat then panicked;
//   - rejected lines over 64KB.

func TestParseCNFHeaderClauseCountIgnored(t *testing.T) {
	clauses := [][]int{{-1, 2}, {-1, -2}, {1}, {2, 3}}
	for _, header := range []string{"p cnf 3 4", "p cnf 3 6", "p cnf 3 2", "p cnf 3 0"} {
		text := header + "\n-1 2 0\n-1 -2 0\n1 0\n2 3 0\n"
		pb := mustParse(t, text)
		if pb.NbClauses != 4 || len(pb.Clauses) != 4 {
			t.Fatalf("%s: NbClauses=%d, %d clauses, want 4", header, pb.NbClauses, len(pb.Clauses))
		}
		for name, f := range map[string]func(*Problem) (*Problem, error){
			"MUS": (*Problem).MUS, "MUSDeletion": (*Problem).MUSDeletion, "MUSInsertion": (*Problem).MUSInsertion,
		} {
			mus, err := f(mustParse(t, text))
			if err != nil {
				t.Fatalf("%s %s: %v", header, name, err)
			}
			checkMUS(t, header+" "+name, 3, clauses, mus.Clauses)
		}
	}
}

func TestParseCNFClauseSpanningLines(t *testing.T) {
	pb := mustParse(t, "p cnf 3 2\n1\n2 3\n0 -1 0\n-2\n0\n-3")
	want := [][]int{{1, 2, 3}, {-1}, {-2}, {-3}}
	if fmt.Sprint(pb.Clauses) != fmt.Sprint(want) {
		t.Fatalf("clauses = %v, want %v", pb.Clauses, want)
	}
	if ok, err := pb.Unsat(strings.NewReader("0\n")); err != nil || !ok {
		t.Errorf("Unsat = %v, %v; want true", ok, err)
	}
}

func TestParseCNFInvalidInput(t *testing.T) {
	inputs := []string{
		"p cnf 1 2\n1 2 0\n-1 0\n", // lit out of range in a non-unit clause
		"p cnf 2 1\n1 -3 0\n",
		"1 2 0\np cnf 2 1\n", // clause before header
		"p cnf 2 1\n1 x 0\n",
		"p cnf 2\n",
		"p cnf -1 0\n",
		"p cnf 2 1\np cnf 2 1\n",
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if e := recover(); e != nil {
					t.Errorf("%q: panic %v", in, e)
				}
			}()
			if _, err := ParseCNF(strings.NewReader(in)); err == nil {
				t.Errorf("%q: parsed without error", in)
			}
		}()
	}
}

func TestParseCNFLongLines(t *testing.T) {
	const n = 30000
	var sb strings.Builder
	fmt.Fprintf(&sb, "p cnf %d 2\n", n)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "%d ", i)
	}
	sb.WriteString("0\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "-%d ", i)
	}
	sb.WriteString("0\n")
	pb := mustParse(t, sb.String())
	if len(pb.Clauses) != 2 || len(pb.Clauses[0]) != n {
		t.Fatalf("got %d clauses", len(pb.Clauses))
	}
}

// Random CNFs written with random line breaks inside clauses, comments, and
// wrong clause counts: the parsed clauses must be exactly the written ones.
func TestParseCNFRandomLayout(t *testing.T) {
	r := rand.New(rand.NewSource(18))
	for it := 0; it < 2000; it++ {
		nbVars := r.Intn(8) + 1
		clauses := randClauses(r, nbVars, r.Intn(10)+1)
		var sb strings.Builder
		fmt.Fprintf(&sb, "c random\np cnf %d %d\n", nbVars, r.Intn(20))
		for i, c := range clauses {
			for _, lit := range c {
				fmt.Fprintf(&sb, "%d", lit)
				sb.WriteString([]string{" ", " ", "\n", "  \t"}[r.Intn(4)])
			}
			if i == len(clauses)-1 && r.Intn(2) == 0 {
				break // no final 0
			}
			sb.WriteString("0")
			sb.WriteString([]string{"\n", " ", "\n\n", " \n"}[r.Intn(4)])
			if r.Intn(5) == 0 && strings.HasSuffix(sb.String(), "\n") {
				sb.WriteString("c comment\n")
			}
		}
		pb := mustParse(t, sb.String())
		if fmt.Sprint(pb.Clauses) != fmt.Sprint(clauses) || pb.NbClauses != len(clauses) {
			t.Fatalf("parsed %v (NbClauses %d), want %v, from %q", pb.Clauses, pb.NbClauses, clauses, sb.String())
		}
	}
}
