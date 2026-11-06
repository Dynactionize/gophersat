package explain

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

// UnsatSubset read the solver goroutine's status without synchronization (a
// data race reported by -race), checked it before the error, and left the
// solver goroutine blocked forever on CertChan whenever UnsatChan returned
// early.

func TestUnsatSubsetWaitsForSolver(t *testing.T) {
	r := rand.New(rand.NewSource(22))
	base := runtime.NumGoroutine()
	nbSat, nbUnsat := 0, 0
	for nbSat < 100 || nbUnsat < 200 {
		// 3-SAT near the phase transition: rarely refuted by unit
		// propagation alone, so the solver goroutine is really used.
		nbVars := r.Intn(8) + 5
		clauses := make([][]int, int(4.5*float64(nbVars)))
		for i := range clauses {
			for _, p := range r.Perm(nbVars)[:3] {
				lit := p + 1
				if r.Intn(2) == 0 {
					lit = -lit
				}
				clauses[i] = append(clauses[i], lit)
			}
		}
		sat := bfSat(nbVars, clauses)
		if sat {
			nbSat++
		} else {
			nbUnsat++
		}
		sub, err := mustParse(t, cnfText(nbVars, clauses)).UnsatSubset()
		if sat {
			if err != ErrNotUnsat {
				t.Fatalf("%v: UnsatSubset of a SAT problem: %v, %v", clauses, sub, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%v: %v", clauses, err)
		}
		if bfSat(nbVars, sub.Clauses) {
			t.Fatalf("%v: subset %v is satisfiable", clauses, sub.Clauses)
		}
	}
	// Every solver goroutine must have finished.
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > base && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > base {
		t.Errorf("%d goroutines still running after UnsatSubset calls (baseline %d)", n, base)
	}
}
