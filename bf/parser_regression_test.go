package bf

import (
	"math/rand"
	"strings"
	"testing"
)

// The parser accepted any token as a variable name: "a & }", "a | -",
// ", & a" and "{1, 2}" parsed into formulas over variables named "}", "-",
// "," or "1" instead of being rejected.

func TestParseRejectsNonIdentifiers(t *testing.T) {
	for _, in := range []string{"a & }", "a | -", ", & a", "{1, 2}", "{a, 2}", "1", "a & 3", "{a, }", "{}", "a & (", "a -> ;"} {
		if f, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%q: parsed as %v, want an error", in, f)
		}
	}
	for _, in := range []string{"a", "a1 & _b", "{a, b, c}", "x_1 -> ^y2", "a = b; c | ^d"} {
		if _, err := Parse(strings.NewReader(in)); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
}

// toSyntax prints f in the parser's input syntax (fully parenthesized).
func toSyntax(f Formula) string {
	switch f := f.(type) {
	case variable:
		return f.name
	case not:
		return "^(" + toSyntax(f[0]) + ")"
	case and, or:
		var subs []Formula
		op := " & "
		if o, ok := f.(or); ok {
			subs, op = o, " | "
		} else {
			subs = f.(and)
		}
		parts := make([]string, len(subs))
		for i, s := range subs {
			parts[i] = "(" + toSyntax(s) + ")"
		}
		return strings.Join(parts, op)
	}
	panic("unsupported")
}

// randSyntaxFormula is like randFormula but without constants and empty
// connectives, which have no syntax.
func randSyntaxFormula(r *rand.Rand, depth int) Formula {
	if depth == 0 || r.Intn(4) == 0 {
		return Var(names[r.Intn(len(names))])
	}
	switch r.Intn(3) {
	case 0:
		return Not(randSyntaxFormula(r, depth-1))
	case 1:
		return And(randSyntaxFormula(r, depth-1), randSyntaxFormula(r, depth-1))
	default:
		return Or(randSyntaxFormula(r, depth-1), randSyntaxFormula(r, depth-1))
	}
}

// Printing random formulas and parsing them back gives equivalent formulas.
func TestParseRoundTripRandom(t *testing.T) {
	r := rand.New(rand.NewSource(17))
	for it := 0; it < 3000; it++ {
		f := randSyntaxFormula(r, r.Intn(5)+1)
		src := toSyntax(f)
		g, err := Parse(strings.NewReader(src))
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		for a := 0; a < 1<<len(names); a++ {
			m := map[string]bool{}
			for i, n := range names {
				m[n] = a>>i&1 == 1
			}
			if f.Eval(m) != g.Eval(m) {
				t.Fatalf("%q parsed as %v, which differs from %v on %v", src, g, f, m)
			}
		}
	}
}
