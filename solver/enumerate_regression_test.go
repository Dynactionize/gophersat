package solver

import (
	"fmt"
	"math/rand"
	"testing"
)

// The clause blocking an already-found model used to be watched on its two
// lowest-level literals. After a backjump the clause could become unit
// without either watch changing, so the propagation was missed and the same
// model was enumerated again: CountModels/Enumerate over-counted.

func TestEnumerateBlockingClauseCounterexamples(t *testing.T) {
	cases := []struct {
		n    int
		cnf  [][]int
		want int
	}{
		{8, [][]int{{8, -5}, {3, -7, 2, 8}, {-3, 4, 5, 1}, {-6, 4, -8, -3}, {5, 4, 8, -7}, {-5, -1, -4, -3}, {-2, 6, -7}, {1, -4, -6}}, 99},
		{8, [][]int{{2, -4}, {-8, -3, -2, 7}, {2, -1, 7}, {-4, -5, -6, 1}, {4, -5, 6}, {8, 3, -1, -2}, {-4, -3, -1}, {1, 6, 5}, {2, -4}, {-6, 8, 4}, {-8, -6}, {2, -1}, {-3, -6, -7}}, 19},
	}
	for i, c := range cases {
		if got := bfCount(c.n, cnfConstrs(c.cnf), nil); got != c.want {
			t.Fatalf("case %d: oracle says %d, test expects %d", i, got, c.want)
		}
		if got := New(ParseSliceNb(cloneCNF(c.cnf), c.n)).CountModels(); got != c.want {
			t.Errorf("case %d: CountModels = %d, want %d", i, got, c.want)
		}
		checkEnumerate(t, fmt.Sprintf("case %d", i), c.n, c.cnf, c.want)
	}
}

func checkEnumerate(t *testing.T, desc string, n int, cnf [][]int, want int) bool {
	t.Helper()
	cs := cnfConstrs(cnf)
	ch := make(chan []bool, 1<<uint(n)+1)
	got := New(ParseSliceNb(cloneCNF(cnf), n)).Enumerate(ch, nil)
	ok := true
	seen := map[string]bool{}
	for m := range ch {
		k := fmt.Sprint(m)
		if seen[k] {
			t.Errorf("%s: Enumerate sent model %v twice", desc, m)
			ok = false
		}
		seen[k] = true
		if msg := checkModel(n, cs, nil, m); msg != "" {
			t.Errorf("%s: Enumerate sent invalid model: %s", desc, msg)
			ok = false
		}
	}
	if got != want || len(seen) != want {
		t.Errorf("%s: Enumerate returned %d and sent %d distinct models, want %d", desc, got, len(seen), want)
		ok = false
	}
	return ok
}

func TestCountModelsRandomVsBruteForce(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	iters := 5000
	if testing.Short() {
		iters = 500
	}
	for it := 0; it < iters; it++ {
		n := r.Intn(9) + 2
		cnf := randCNF(r, n, r.Intn(3*n)+1)
		want := bfCount(n, cnfConstrs(cnf), nil)
		desc := fmt.Sprintf("n=%d cnf=%v", n, cnf)
		if got := New(ParseSliceNb(cloneCNF(cnf), n)).CountModels(); got != want {
			t.Fatalf("CountModels = %d, want %d: %s", got, want, desc)
		}
		if !checkEnumerate(t, desc, n, cnf, want) {
			t.FailNow()
		}
	}
}
