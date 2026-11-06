package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// DetectAtMostOne (used by the CLI with -cp) dropped clauses:
//   - binary clauses were scheduled for removal even when no cardinality
//     constraint was emitted for them;
//   - removeBinaries stopped copying clauses after the last removed index and
//     assumed its input sorted and without duplicates.
// The result was a weaker problem, e.g. SAT for an UNSAT formula.

func TestDetectAtMostOneKeepsUnrelatedClauses(t *testing.T) {
	cnf := "p cnf 5 6\n-1 -2 0\n-1 -3 0\n4 5 0\n4 -5 0\n-4 5 0\n-4 -5 0\n"
	pb, err := ParseCNF(strings.NewReader(cnf))
	if err != nil {
		t.Fatal(err)
	}
	pb.DetectAtMostOne()
	for _, cp := range []bool{false, true} {
		s := New(pb)
		s.CuttingPlanes = cp
		if st := s.Solve(); st != Unsat {
			t.Errorf("cp=%v: got %v, want UNSAT", cp, st)
		}
	}
}

func TestDetectAtMostOneNoCliqueRemovesNothing(t *testing.T) {
	pb := ParseSlice([][]int{{2, 3}, {-1, -2}, {-1, -3}, {1, 4}})
	before := len(pb.Clauses)
	pb.DetectAtMostOne()
	if len(pb.Clauses) != before {
		t.Errorf("clauses: %d -> %d, want unchanged: %s", before, len(pb.Clauses), pb.PBString())
	}
}

func TestDetectAtMostOneFindsClique(t *testing.T) {
	// Pairwise encoding of AtMostOne(x1..x4), plus an unrelated clause.
	var cnf [][]int
	for i := 1; i <= 4; i++ {
		for j := i + 1; j <= 4; j++ {
			cnf = append(cnf, []int{-i, -j})
		}
	}
	cnf = append(cnf, []int{1, 2, 5})
	pb := ParseSlice(cnf)
	pb.DetectAtMostOne()
	card := 0
	for _, c := range pb.Clauses {
		if c.Cardinality() > 1 {
			card++
		}
	}
	if card != 1 {
		t.Fatalf("found %d cardinality constraints, want 1: %s", card, pb.PBString())
	}
	want := bfCount(5, cnfConstrs(cnf), nil)
	if got := New(pb).CountModels(); got != want {
		t.Errorf("CountModels = %d, want %d", got, want)
	}
}

// Random problems rich in negative binary clauses (cliques), mixed with other
// clauses: detection must preserve the set of models.
func TestDetectAtMostOneRandomVsBruteForce(t *testing.T) {
	iters := 3000
	if testing.Short() {
		iters = 300
	}
	r := rand.New(rand.NewSource(12))
	for it := 0; it < iters; it++ {
		n := r.Intn(8) + 3
		var cnf [][]int
		// a few random cliques, pairwise encoded, sometimes with a pair missing
		for k := r.Intn(3); k >= 0; k-- {
			vars := r.Perm(n)[:r.Intn(n-1)+2]
			for i := range vars {
				for j := i + 1; j < len(vars); j++ {
					if r.Intn(10) == 0 {
						continue
					}
					cnf = append(cnf, []int{-(vars[i] + 1), -(vars[j] + 1)})
				}
			}
		}
		cnf = append(cnf, randCNF(r, n, r.Intn(2*n)+1)...)
		cs := cnfConstrs(cnf)
		want := bfCount(n, cs, nil)
		desc := fmt.Sprintf("n=%d cnf=%v", n, cnf)
		for _, cp := range []bool{false, true} {
			pb := ParseSliceNb(cloneCNF(cnf), n)
			pb.DetectAtMostOne()
			s := New(pb)
			s.CuttingPlanes = cp
			st := s.Solve()
			if (st == Sat) != (want > 0) {
				t.Fatalf("cp=%v: got %v, want sat=%v: %s", cp, st, want > 0, desc)
			}
			if st == Sat {
				if msg := checkModel(n, cs, nil, s.Model()); msg != "" {
					t.Fatalf("cp=%v: %s: %s", cp, msg, desc)
				}
			}
		}
		pb := ParseSliceNb(cloneCNF(cnf), n)
		pb.DetectAtMostOne()
		if got := New(pb).CountModels(); got != want {
			t.Fatalf("CountModels after detection = %d, want %d: %s", got, want, desc)
		}
	}
}
