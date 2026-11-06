package maxsat

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dynactionize/gophersat/solver"
)

// A Solver is a [partial][weighted] MAXSAT solver.
// It implements solver.Interface.
type Solver struct {
	solver     *solver.Solver
	firstRelax int // Identifier of first relax variable: those must not be provided as part of the actual result
}

// Optimal looks for the optimal solution to the underlying problem.
// If results is not nil, it writes a suboptimal solution every time it finds a new, better one.
// In any case, it returns the optimal solution to the problem, or UNSAT if the problem cannot be found.
func (s *Solver) Optimal(results chan solver.Result, stop chan struct{}) solver.Result {
	if results == nil {
		res := s.solver.Optimal(nil, stop)
		return res
	}
	localRes := make(chan solver.Result)
	defer close(results)
	go s.solver.Optimal(localRes, stop)
	var res solver.Result
	for res = range localRes {
		if res.Status == solver.Sat {
			res.Model = res.Model[:s.firstRelax] // Remove relax vars from the model
		}
		results <- res
	}
	return res // Last result is returned
}

// Enumerate does not make sense for a MAXSAT problem, so it will panic when called.
// This might change in later versions.
func (s *Solver) Enumerate(models chan []bool, stop chan struct{}) int {
	panic("trying to call Enumerate on a MAXSAT problem")
}

// ParseWCNF parses a WCNF file and returns the corresponding solver.Interface.
// The file must start with a "p wcnf nbvars nbclauses [top]" header; each
// clause is on its own line: a weight, literals over vars 1..nbvars, and a
// final 0.
func ParseWCNF(f io.Reader) (solver.Interface, error) {
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), solver.MaxLineLen)
	var (
		headerSeen bool
		nbVars     int
		topWeight  int // weight of hard clauses
		clauses    [][]int
		weights    []int
		relaxLit   int // index of current relax lit
	)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == 'c' {
			continue
		}
		if line[0] == 'p' {
			if headerSeen {
				return nil, fmt.Errorf("duplicate header %q in WCNF file", line)
			}
			fields := strings.Fields(line)
			if len(fields) < 4 || len(fields) > 5 || fields[1] != "wcnf" {
				return nil, fmt.Errorf("invalid syntax %q in WCNF file", line)
			}
			var err error
			nbVars, err = strconv.Atoi(fields[2])
			if err != nil || nbVars < 0 {
				return nil, fmt.Errorf("nbvars not a non-negative int: %q", fields[2])
			}
			nbClauses, err := strconv.Atoi(fields[3])
			if err != nil || nbClauses < 0 {
				return nil, fmt.Errorf("nbClauses not a non-negative int: %q", fields[3])
			}
			relaxLit = nbVars + 1
			clauses = make([][]int, 0, nbClauses)
			weights = make([]int, 0, nbClauses)
			if len(fields) == 5 {
				topWeight, err = strconv.Atoi(fields[4])
				if err != nil || topWeight <= 0 {
					return nil, fmt.Errorf("top weight not a positive int: %q", fields[4])
				}
			}
			headerSeen = true
			continue
		}
		if !headerSeen {
			return nil, fmt.Errorf("clause %q found before the \"p wcnf\" header", line)
		}
		clause, weight, err := parseWCNFClause(line, nbVars)
		if err != nil {
			return nil, err
		}
		if topWeight == 0 || weight < topWeight { // Soft clause: add a relax lit
			clause = append(clause, relaxLit)
			relaxLit++
			weights = append(weights, weight)
		}
		clauses = append(clauses, clause)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not parse WCNF: %v", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing \"p wcnf\" header")
	}
	relaxLits := make([]solver.Lit, relaxLit-nbVars-1)
	for i := range relaxLits {
		relaxLits[i] = solver.IntToLit(int32(nbVars + i + 1))
	}
	prob := solver.ParseSliceNb(clauses, relaxLit-1)
	prob.SetCostFunc(relaxLits, weights)
	s := solver.New(prob)
	return &Solver{solver: s, firstRelax: nbVars}, nil
}

// parseWCNFClause parses a WCNF line containing a clause (weight, lits, 0)
// and returns its lits and its weight.
func parseWCNFClause(line string, nbVars int) (lits []int, weight int, err error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, 0, fmt.Errorf("invalid WCNF clause %q: expected a weight, literals and a final 0", line)
	}
	vals := make([]int, len(fields))
	for i, field := range fields {
		if vals[i], err = strconv.Atoi(field); err != nil {
			return nil, 0, fmt.Errorf("invalid integer %q in WCNF clause %q", field, line)
		}
	}
	weight = vals[0]
	if weight < 0 {
		return nil, 0, fmt.Errorf("negative weight in WCNF clause %q", line)
	}
	if vals[len(vals)-1] != 0 {
		return nil, 0, fmt.Errorf("WCNF clause %q does not end with 0", line)
	}
	lits = vals[1 : len(vals)-1]
	for _, lit := range lits {
		if lit == 0 || lit > nbVars || -lit > nbVars {
			return nil, 0, fmt.Errorf("invalid literal %d in WCNF clause %q: vars must be in 1..%d", lit, line, nbVars)
		}
	}
	return lits, weight, nil
}
