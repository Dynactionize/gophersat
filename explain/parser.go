package explain

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dynactionize/gophersat/solver"
)

// parseClause parses a line representing a clause in the DIMACS CNF syntax.
func parseClause(fields []string) ([]int, error) {
	clause := make([]int, 0, len(fields)-1)
	for _, rawLit := range fields {
		lit, err := strconv.Atoi(rawLit)
		if err != nil {
			return nil, fmt.Errorf("could not parse clause %v: %v", fields, err)
		}
		if lit != 0 {
			clause = append(clause, lit)
		}
	}
	return clause, nil
}

// ParseCNF parses a CNF and returns the associated problem.
// Clauses may span several lines and the last one may lack its final 0. The
// number of clauses announced in the header is not trusted: NbClauses is the
// number of clauses actually read.
func ParseCNF(r io.Reader) (*Problem, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), solver.MaxLineLen)
	var pb Problem
	headerSeen := false
	var clause []int // clause being read; may span several lines
	inClause := false
	addClause := func() {
		pb.Clauses = append(pb.Clauses, clause)
		if len(clause) == 1 {
			lit := clause[0]
			if lit > 0 {
				pb.units[lit-1] = 1
			} else {
				pb.units[-lit-1] = -1
			}
		}
		clause = nil
		inClause = false
	}
lines:
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(fields[0], "c"): // comment
			continue
		case fields[0] == "%": // SATLIB trailer
			break lines
		case fields[0] == "p":
			if headerSeen {
				return nil, fmt.Errorf("duplicate header %q", line)
			}
			if err := pb.parseHeader(fields); err != nil {
				return nil, fmt.Errorf("could not parse header %q: %v", line, err)
			}
			headerSeen = true
			continue
		}
		if !headerSeen {
			return nil, fmt.Errorf("clause %q found before the header", line)
		}
		for _, field := range fields {
			lit, err := strconv.Atoi(field)
			if err != nil {
				return nil, fmt.Errorf("could not parse clause %q: %v", line, err)
			}
			if lit == 0 {
				addClause()
				continue
			}
			if lit > pb.NbVars || -lit > pb.NbVars {
				return nil, fmt.Errorf("found lit %d but problem was supposed to hold only %d vars", lit, pb.NbVars)
			}
			clause = append(clause, lit)
			inClause = true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("could not parse problem: %v", err)
	}
	if inClause { // last clause without its final 0
		addClause()
	}
	pb.NbClauses = len(pb.Clauses)
	return &pb, nil
}

func (pb *Problem) parseHeader(fields []string) error {
	if len(fields) != 4 {
		return fmt.Errorf("expected 4 fields, got %d", len(fields))
	}
	strVars := fields[2]
	strClauses := fields[3]
	var err error
	pb.NbVars, err = strconv.Atoi(fields[2])
	if err != nil {
		return fmt.Errorf("invalid number of vars %q: %v", strVars, err)
	}
	if pb.NbVars < 0 {
		return fmt.Errorf("negative number of vars %d", pb.NbVars)
	}
	pb.units = make([]int, pb.NbVars)
	nbClauses, err := strconv.Atoi(fields[3])
	if err != nil {
		return fmt.Errorf("invalid number of clauses %s: %v", strClauses, err)
	}
	if nbClauses < 0 {
		return fmt.Errorf("negative number of clauses %d", nbClauses)
	}
	pb.Clauses = make([][]int, 0, nbClauses)
	return nil
}
