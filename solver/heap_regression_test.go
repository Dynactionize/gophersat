package solver

import (
	"math/rand"
	"testing"
)

// rebuildOrderHeap started from make([]int, nbVars) and then appended the
// unbound vars, so the decision heap held nbVars extra copies of var 0 after
// every restart: twice the size it should have, a wrong index for var 0, and
// var 0 ordered incorrectly.

// checkHeap verifies the heap invariants of q: each var appears at most once,
// indices are consistent, and the max-activity heap property holds.
func checkHeap(t *testing.T, q *queue) {
	t.Helper()
	seen := map[int]bool{}
	for i, v := range q.content {
		if seen[v] {
			t.Fatalf("var %d appears twice in the heap (size %d)", v, len(q.content))
		}
		seen[v] = true
		if q.indices[v] != i {
			t.Fatalf("indices[%d] = %d, but var is at position %d", v, q.indices[v], i)
		}
		if i > 0 && q.lt(v, q.content[parent(i)]) {
			t.Fatalf("heap property violated at position %d", i)
		}
	}
	for v, idx := range q.indices {
		if idx >= 0 && !seen[v] {
			t.Fatalf("indices[%d] = %d but var is not in the heap", v, idx)
		}
	}
}

func TestRebuildOrderHeapHoldsEachUnboundVarOnce(t *testing.T) {
	s := New(ParseSlice([][]int{{1, 2, 3}, {-1, 4}, {-2, -3, 5}, {4, 5, 6}}))
	s.activity[0] = 10 // var 0 must come out first, exactly once
	s.model[2] = 2     // a bound var must not be in the heap
	s.rebuildOrderHeap()
	checkHeap(t, &s.varQueue)
	if got, want := len(s.varQueue.content), s.nbVars-1; got != want {
		t.Fatalf("heap holds %d entries, want %d (unbound vars)", got, want)
	}
	if s.varQueue.contains(2) {
		t.Fatalf("bound var 2 is in the heap")
	}
	if v := s.varQueue.removeMin(); v != 0 {
		t.Fatalf("first var out is %d, want 0", v)
	}
	if s.varQueue.contains(0) {
		t.Fatalf("var 0 still in the heap after removing it")
	}
}

// After solving random problems (with restarts and learned units, which
// rebuild the heap), the heap must still be a valid heap without duplicates.
func TestOrderHeapStaysValidWhileSolving(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		s := New(ParseSlice(random3SAT(seed, 80, 4.26)))
		s.Solve()
		checkHeap(t, &s.varQueue)
		s.rebuildOrderHeap()
		checkHeap(t, &s.varQueue)
	}
	r := rand.New(rand.NewSource(1))
	for it := 0; it < 500; it++ {
		n := r.Intn(8) + 2
		s := New(ParseSliceNb(randCNF(r, n, r.Intn(4*n)+1), n))
		s.CountModels()
		checkHeap(t, &s.varQueue)
	}
}
