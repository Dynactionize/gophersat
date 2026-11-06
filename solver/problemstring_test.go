package solver

import (
	"strings"
	"testing"
)

// Problem.CNF, Problem.PBString and Clause.CNF used to build their output
// with string concatenation in a loop (quadratic). The output must stay
// byte-for-byte the same.

func TestProblemStringsGolden(t *testing.T) {
	pb := ParsePBConstrs([]PBConstr{GtEq([]int{1, -2, 3}, []int{2, 3, 1}, 3), AtLeast([]int{1, 2, 4}, 2), PropClause(-3, 4), PropClause(5)})
	if got, want := pb.CNF(), "p cnf 5 4\n5 0\n-2 1 3 0\n1 2 4 0\n-3 4 0\n"; got != want {
		t.Errorf("CNF() = %q, want %q", got, want)
	}
	if got, want := pb.PBString(), "1 x5 = 1 ;\n3 ~x2 +2 x1 +1 x3 >= 3 ;\n1 x1 +1 x2 +1 x4 >= 2 ;\n1 ~x3 +1 x4 >= 1 ;\n"; got != want {
		t.Errorf("PBString() = %q, want %q", got, want)
	}
	if got, want := NewClause(IntsToLits(1, -2, 30)).CNF(), "1 -2 30 0"; got != want {
		t.Errorf("Clause.CNF() = %q, want %q", got, want)
	}
}

// The CNF output of a CNF problem parses back to the same problem.
func TestProblemCNFRoundTrip(t *testing.T) {
	cnf := random3SAT(3, 300, 3.0)
	pb := ParseSlice(cloneCNF(cnf))
	out := pb.CNF()
	pb2, err := ParseCNF(strings.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if pb2.CNF() != out {
		t.Fatalf("round trip changed the problem")
	}
	if st1, st2 := New(pb).Solve(), New(pb2).Solve(); st1 != st2 {
		t.Fatalf("statuses differ: %v vs %v", st1, st2)
	}
}

func BenchmarkProblemCNF(b *testing.B) {
	pb := ParseSlice(random3SAT(1, 50000, 4.0))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.CNF()
	}
}

func BenchmarkProblemPBString(b *testing.B) {
	pb := ParseSlice(random3SAT(1, 50000, 4.0))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.PBString()
	}
}
