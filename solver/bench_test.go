package solver

import (
	"math/rand"
	"testing"
)

// Benchmarks on generated instances, used to measure performance changes.
// random3SAT is defined in concurrency_regression_test.go.

func benchSolve(b *testing.B, cnfs [][][]int, cp bool) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cnf := range cnfs {
			s := New(ParseSlice(cloneCNF(cnf)))
			s.CuttingPlanes = cp
			s.Solve()
		}
	}
}

// Hard, small: many conflicts on few vars.
func BenchmarkSolve3SATPhaseTransition(b *testing.B) {
	var cnfs [][][]int
	for seed := int64(1); seed <= 4; seed++ {
		cnfs = append(cnfs, random3SAT(seed, 180, 4.26))
	}
	benchSolve(b, cnfs, false)
}

// Medium: thousands of conflicts on thousands of vars.
func BenchmarkSolve3SATMedium(b *testing.B) {
	benchSolve(b, [][][]int{random3SAT(1, 2000, 3.4)}, false)
}

// Large and easy: mostly propagation.
func BenchmarkSolve3SATLarge(b *testing.B) {
	benchSolve(b, [][][]int{random3SAT(1, 50000, 3.0)}, false)
}

// Pigeonhole principle (n+1 pigeons, n holes) with cardinality
// constraints, solved with cutting planes.
func BenchmarkSolvePigeonholeCuttingPlanes(b *testing.B) {
	const holes = 30
	var constrs []PBConstr
	v := func(p, h int) int { return p*holes + h + 1 }
	for p := 0; p <= holes; p++ {
		lits := make([]int, holes)
		for h := range lits {
			lits[h] = v(p, h)
		}
		constrs = append(constrs, AtLeast(lits, 1))
	}
	for h := 0; h < holes; h++ {
		lits := make([]int, holes+1)
		for p := range lits {
			lits[p] = v(p, h)
		}
		constrs = append(constrs, AtMost(lits, 1))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(ParsePBConstrs(clonePB(constrs)))
		s.CuttingPlanes = true
		if s.Solve() != Unsat {
			b.Fatal("pigeonhole must be UNSAT")
		}
	}
}

// Random PB problems with cutting planes.
func BenchmarkSolveRandomPBCuttingPlanes(b *testing.B) {
	r := rand.New(rand.NewSource(1))
	var all [][]PBConstr
	for len(all) < 200 {
		constrs, _ := randPBConstrs(r, 30, 60)
		all = append(all, constrs)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, constrs := range all {
			s := New(padVars(ParsePBConstrs(clonePB(constrs)), 30))
			s.CuttingPlanes = true
			s.Solve()
		}
	}
}
