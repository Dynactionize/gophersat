package solver

import (
	"math/rand"
	"sync"
	"testing"
)

// learnClause used a package-level buffer, so solvers running in different
// goroutines overwrote each other's learned clauses (a data race that
// `go test -race` reports immediately, and that can produce unsound learned
// clauses). Each solver now owns its buffer.

func random3SAT(seed int64, n int, ratio float64) [][]int {
	r := rand.New(rand.NewSource(seed))
	cnf := make([][]int, int(float64(n)*ratio))
	for i := range cnf {
		for _, p := range r.Perm(n)[:3] {
			x := p + 1
			if r.Intn(2) == 0 {
				x = -x
			}
			cnf[i] = append(cnf[i], x)
		}
	}
	return cnf
}

func TestConcurrentSolversAgreeWithSequential(t *testing.T) {
	const nbProblems = 16
	n := 120
	if testing.Short() {
		n = 60
	}
	cnfs := make([][][]int, nbProblems)
	want := make([]Status, nbProblems)
	for i := range cnfs {
		cnfs[i] = random3SAT(int64(i), n, 4.26) // phase transition: many conflicts
		want[i] = New(ParseSlice(cloneCNF(cnfs[i]))).Solve()
	}
	for round := 0; round < 3; round++ {
		var wg sync.WaitGroup
		got := make([]Status, nbProblems)
		models := make([][]bool, nbProblems)
		for i := range cnfs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				s := New(ParseSlice(cloneCNF(cnfs[i])))
				got[i] = s.Solve()
				if got[i] == Sat {
					models[i] = s.Model()
				}
			}(i)
		}
		wg.Wait()
		for i := range cnfs {
			if got[i] != want[i] {
				t.Fatalf("round %d problem %d: concurrent %v, sequential %v", round, i, got[i], want[i])
			}
			if got[i] == Sat {
				if msg := checkModel(n, cnfConstrs(cnfs[i]), nil, models[i]); msg != "" {
					t.Fatalf("round %d problem %d: %s", round, i, msg)
				}
			}
		}
	}
}
