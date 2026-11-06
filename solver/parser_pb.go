package solver

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// ParseCardConstrs parses the given cardinality constraints.
// Will panic if a zero value appears in the literals.
func ParseCardConstrs(constrs []CardConstr) *Problem {
	var pb Problem
	// Count every var, including those of skipped (trivially satisfied)
	// constraints, so that the model covers all the caller's vars.
	for _, constr := range constrs {
		for _, val := range constr.Lits {
			if v := max(val, -val); v > pb.NbVars {
				pb.NbVars = v
			}
		}
	}
	for _, constr := range constrs {
		card := constr.AtLeast
		if card <= 0 { // Clause is trivially SAT, ignore
			continue
		}
		if len(constr.Lits) < card { // Clause cannot be satsfied
			pb.Status = Unsat
			return &pb
		}
		if len(constr.Lits) == card { // All lits must be true
			for i := range constr.Lits {
				if constr.Lits[i] == 0 {
					panic("literal 0 found in clause")
				}
				lit := IntToLit(int32(constr.Lits[i]))
				v := lit.Var()
				if int(v) >= pb.NbVars {
					pb.NbVars = int(v) + 1
				}
				pb.Units = append(pb.Units, lit)
			}
		} else {
			lits := make([]Lit, len(constr.Lits))
			for j, val := range constr.Lits {
				if val == 0 {
					panic("literal 0 found in clause")
				}
				lits[j] = IntToLit(int32(val))
				if v := int(lits[j].Var()); v >= pb.NbVars {
					pb.NbVars = v + 1
				}
			}
			pb.Clauses = append(pb.Clauses, NewCardClause(lits, card))
		}
	}
	pb.Model = make([]decLevel, pb.NbVars)
	for _, unit := range pb.Units {
		v := unit.Var()
		if pb.Model[v] == 0 {
			if unit.IsPositive() {
				pb.Model[v] = 1
			} else {
				pb.Model[v] = -1
			}
		} else if pb.Model[v] > 0 != unit.IsPositive() {
			pb.Status = Unsat
			return &pb
		}
	}
	pb.simplifyCard()
	return &pb
}

func (pb *Problem) appendClause(constr PBConstr) {
	lits := make([]Lit, len(constr.Lits))
	for j, val := range constr.Lits {
		lits[j] = IntToLit(int32(val))
	}
	pb.Clauses = append(pb.Clauses, NewPBClause(lits, constr.Weights, constr.AtLeast))
}

// coeffsFit returns true iff 2*sum(|weights|) + |rhs| fits in an int. This
// bounds every intermediate sum computed while normalizing and solving the
// constraint (normalized cardinality, slack, watched weight), which would
// otherwise silently overflow and give wrong results.
func coeffsFit(weights []int, rhs int) bool {
	const limit = math.MaxInt
	abs := func(x int) (uint64, bool) {
		if x == math.MinInt {
			return 0, false
		}
		if x < 0 {
			x = -x
		}
		return uint64(x), true
	}
	total, ok := abs(rhs)
	if !ok {
		return false
	}
	for _, w := range weights {
		aw, ok := abs(w)
		if !ok {
			return false
		}
		for range 2 {
			if total > limit-aw {
				return false
			}
			total += aw
		}
	}
	return true
}

// saturate lowers every weight above card to card. This does not change the
// constraint's meaning, and bounds all sums of weights.
func saturate(weights []int, card int) {
	for i, w := range weights {
		if w > card {
			weights[i] = card
		}
	}
}

// ParsePBConstrs parses and returns a PB problem from PBConstr values.
// It panics if the coefficients of a constraint are so large that their sum
// overflows an int, or if a normalized cardinality exceeds MaxCardinality.
func ParsePBConstrs(constrs []PBConstr) *Problem {
	var pb Problem
	for _, constr := range constrs {
		if !coeffsFit(constr.Weights, constr.AtLeast) {
			panic(fmt.Sprintf("coefficients too large in %v: their sum overflows", constr))
		}
		for i := range constr.Lits {
			lit := IntToLit(int32(constr.Lits[i]))
			v := lit.Var()
			if int(v) >= pb.NbVars {
				pb.NbVars = int(v) + 1
			}
		}
		card := constr.AtLeast
		if card <= 0 { // Clause is trivially SAT, ignore
			continue
		}
		if card > MaxCardinality {
			panic(fmt.Sprintf("cardinality %d of %v exceeds %d", card, constr, MaxCardinality))
		}
		if constr.Weights != nil {
			constr.Weights = append([]int(nil), constr.Weights...) // don't modify the caller's slice
			saturate(constr.Weights, card)
		}
		sumW := constr.WeightSum()
		if sumW < card { // Clause cannot be satsfied
			pb.Status = Unsat
			return &pb
		}
		if sumW == card { // All lits must be true
			for i := range constr.Lits {
				lit := IntToLit(int32(constr.Lits[i]))
				found := false
				for _, u := range pb.Units {
					if u == lit {
						found = true
						break
					}
				}
				if !found {
					pb.Units = append(pb.Units, lit)
				}
			}
		} else {
			pb.appendClause(constr)
		}
	}
	pb.Model = make([]decLevel, pb.NbVars)
	for _, unit := range pb.Units {
		v := unit.Var()
		if pb.Model[v] == 0 {
			if unit.IsPositive() {
				pb.Model[v] = 1
			} else {
				pb.Model[v] = -1
			}
		} else if pb.Model[v] > 0 != unit.IsPositive() {
			pb.Status = Unsat
			return &pb
		}
	}
	pb.simplifyPB()
	return &pb
}

// parsePBOptim parses the "min:" instruction.
func (pb *Problem) parsePBOptim(fields []string, line string) error {
	weights, lits, err := pb.parseTerms(fields[1:], line)
	if err != nil {
		return err
	}
	pb.minLits = make([]Lit, len(lits))
	for i, lit := range lits {
		pb.minLits[i] = IntToLit(int32(lit))
	}
	pb.minWeights = weights
	return nil
}

func (pb *Problem) parsePBLine(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if line[len(line)-1] != ';' {
		return fmt.Errorf("line %q does not end with semicolon", line)
	}
	fields := strings.Fields(line[:len(line)-1])
	if len(fields) == 0 {
		return fmt.Errorf("empty line in file")
	}
	if fields[0] == "min:" { // Optimization constraint
		return pb.parsePBOptim(fields, line)
	}
	return pb.parsePBConstrLine(fields, line)
}

func (pb *Problem) parsePBConstrLine(fields []string, line string) error {
	if len(fields) < 3 {
		return fmt.Errorf("invalid syntax %q", line)
	}
	operator := fields[len(fields)-2]
	if operator != ">=" && operator != "=" {
		return fmt.Errorf("invalid operator %q in %q: expected \">=\" or \"=\"", operator, line)
	}
	rhs, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		return fmt.Errorf("invalid value %q in %q: %v", fields[len(fields)-1], line, err)
	}
	weights, lits, err := pb.parseTerms(fields[:len(fields)-2], line)
	if err != nil {
		return err
	}
	if !coeffsFit(weights, rhs) {
		return fmt.Errorf("coefficients too large in %q: their sum overflows", line)
	}
	var constrs []PBConstr
	if operator == ">=" {
		constrs = []PBConstr{GtEq(lits, weights, rhs)}
	} else {
		constrs = Eq(lits, weights, rhs)
	}
	for _, constr := range constrs {
		card := constr.AtLeast
		if card <= 0 { // Constraint is trivially satisfied, ignore
			continue
		}
		if card > MaxCardinality {
			return fmt.Errorf("right-hand side too large in %q: normalized value %d exceeds %d", line, card, MaxCardinality)
		}
		saturate(constr.Weights, card)
		sumW := constr.WeightSum()
		if sumW < card { // Clause cannot be satsfied
			pb.Status = Unsat
			return nil
		}
		if sumW == card { // All lits must be true
			for i := range constr.Lits {
				lit := IntToLit(int32(constr.Lits[i]))
				pb.Units = append(pb.Units, lit)
			}
		} else {
			lits := make([]Lit, len(constr.Lits))
			for j, val := range constr.Lits {
				lits[j] = IntToLit(int32(val))
			}
			pb.Clauses = append(pb.Clauses, NewPBClause(lits, constr.Weights, card))
		}
	}
	return nil
}

func (pb *Problem) parseTerms(terms []string, line string) (weights []int, lits []int, err error) {
	weights = make([]int, 0, len(terms)/2)
	lits = make([]int, 0, len(terms)/2)
	i := 0
	for i < len(terms) {
		var l string
		w, err := strconv.Atoi(terms[i])
		if err != nil {
			// This should be a weightless lit, i.e a lit with weight 1.
			l = terms[i]
			weights = append(weights, 1)
		} else {
			weights = append(weights, w)
			i++
			if i == len(terms) {
				return nil, nil, fmt.Errorf("weight %d without variable in %q", w, line)
			}
			l = terms[i]
		}
		name := strings.TrimPrefix(l, "~")
		if !strings.HasPrefix(name, "x") {
			return nil, nil, fmt.Errorf("invalid variable name %q in %q", l, line)
		}
		v, err := strconv.Atoi(name[1:])
		if err != nil || v < 1 || v > math.MaxInt32/2 {
			return nil, nil, fmt.Errorf("invalid variable %q in %q", l, line)
		}
		if l[0] == '~' {
			lits = append(lits, -v)
		} else {
			lits = append(lits, v)
		}
		if v > pb.NbVars {
			pb.NbVars = v
		}
		i++
	}
	return weights, lits, nil
}

// MaxLineLen is the length of the longest line accepted by the line-based
// parsers (OPB here, WCNF and CNF in the maxsat and explain packages).
const MaxLineLen = 1 << 30

// ParseOPB parses a file corresponding to the OPB syntax.
// See http://www.cril.univ-artois.fr/PB16/format.pdf for more details.
func ParseOPB(f io.Reader) (*Problem, error) {
	scanner := bufio.NewScanner(f)
	// Constraints and objectives with many terms make long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), MaxLineLen)
	var pb Problem
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '*' {
			continue
		}
		if err := pb.parsePBLine(line); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not parse OPB: %v", err)
	}
	pb.Model = make([]decLevel, pb.NbVars)
	pb.simplifyPB()
	return &pb, nil
}
