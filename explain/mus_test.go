package explain

import (
	"os"
	"testing"

	"github.com/dynactionize/gophersat/solver"
)

func TestTrivialMUS(t *testing.T) {
	cnf, err := os.Open("testcnf/trivial.cnf")
	if err != nil {
		t.Errorf("could not read CNF file: %v", err)
		return
	}
	defer cnf.Close()
	pb, err := ParseCNF(cnf)
	if err != nil {
		t.Fatalf("could not parse cnf: %v", err)
	}
	mus, err := pb.MUSDeletion()
	if err != nil {
		t.Fatalf("could not compute MUS: %v", err)
	}
	s := solver.New(solver.ParseSlice(mus.Clauses))
	if s.Solve() != solver.Unsat {
		t.Errorf("MUS was satisfiable")
	}
}

func TestMUSOnSatisfiableFormula(t *testing.T) {
	cnf, err := os.Open("testcnf/impossible.cnf")
	if err != nil {
		t.Errorf("could not read CNF file: %v", err)
		return
	}
	defer cnf.Close()

	pb, err := ParseCNF(cnf)
	if err != nil {
		t.Fatalf("could not parse cnf: %v", err)
	}
	mus, err := pb.MUSDeletion()
	if err == nil {
		t.Fatal("This function should return an error on satisfiable formula")
		s := solver.New(solver.ParseSlice(mus.Clauses))
		if s.Solve() != solver.Unsat {
			t.Errorf("MUS was satisfiable")
		}
	}
}
