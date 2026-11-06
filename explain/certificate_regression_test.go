package explain

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/dynactionize/gophersat/solver"
)

// Unsat and UnsatChan accepted any certificate whose clauses were RUP, even
// when it never derived the empty clause: an empty certificate "proved" that
// a satisfiable problem was UNSAT.

// solverCertificate solves the problem with certification on and returns
// the status and the certificate lines.
func solverCertificate(nbVars int, clauses [][]int) (solver.Status, []string) {
	s := solver.New(solver.ParseSliceNb(clauses, nbVars))
	s.Certified = true
	s.CertChan = make(chan string)
	var st solver.Status
	done := make(chan struct{})
	go func() {
		st = s.Solve()
		close(s.CertChan)
	}()
	var lines []string
	go func() {
		for l := range s.CertChan {
			lines = append(lines, l)
		}
		close(done)
	}()
	<-done
	return st, lines
}

func TestCertificateForSatProblemIsInvalid(t *testing.T) {
	pb := mustParse(t, "p cnf 2 2\n1 2 0\n-1 2 0\n")
	for _, cert := range []string{"", "c nothing\n", "2 0\n", "2 0\n2 0\n"} {
		if ok, err := pb.Unsat(strings.NewReader(cert)); ok || err != nil {
			t.Errorf("Unsat(%q) = %v, %v; want false", cert, ok, err)
		}
		ch := make(chan string, 10)
		for _, l := range strings.Split(strings.TrimSpace(cert), "\n") {
			ch <- l
		}
		close(ch)
		if ok, err := pb.UnsatChan(ch); ok || err != nil {
			t.Errorf("UnsatChan(%q) = %v, %v; want false", cert, ok, err)
		}
	}
}

func TestCertificatesRandomVsBruteForce(t *testing.T) {
	iters := 2000
	if testing.Short() {
		iters = 200
	}
	r := rand.New(rand.NewSource(21))
	for it := 0; it < iters; it++ {
		nbVars := r.Intn(7) + 1
		clauses := randClauses(r, nbVars, r.Intn(5*nbVars)+1)
		sat := bfSat(nbVars, clauses)
		text := cnfText(nbVars, clauses)
		st, lines := solverCertificate(nbVars, clauses)
		if (st == solver.Sat) != sat {
			t.Fatalf("%v: solver says %v, brute force sat=%v", clauses, st, sat)
		}
		cert := strings.Join(lines, "\n") + "\n"
		desc := fmt.Sprintf("%v with certificate %q", clauses, cert)
		// Solver certificates are valid exactly for UNSAT problems.
		ok, err := mustParse(t, text).Unsat(strings.NewReader(cert))
		if err != nil || ok == sat {
			t.Fatalf("Unsat = %v, %v for sat=%v: %s", ok, err, sat, desc)
		}
		ch := make(chan string, len(lines))
		for _, l := range lines {
			ch <- l
		}
		close(ch)
		if ok, err := mustParse(t, text).UnsatChan(ch); err != nil || ok == sat {
			t.Fatalf("UnsatChan = %v, %v for sat=%v: %s", ok, err, sat, desc)
		}
		// No certificate at all can prove a satisfiable problem UNSAT.
		if sat {
			for _, c := range []string{"", "0\n"} {
				if ok, _ := mustParse(t, text).Unsat(strings.NewReader(c)); ok {
					t.Fatalf("Unsat(%q) accepted for satisfiable %v", c, clauses)
				}
			}
		}
	}
}
