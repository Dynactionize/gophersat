package solver

// Brute-force oracle and random problem generators used by the differential
// regression tests. Problems are kept small (<= 10 vars) so that every
// assignment can be enumerated.

import (
	"fmt"
	"math/rand"
)

// bfConstr is a constraint sum(w_i * l_i) >= k over CNF-style int literals.
type bfConstr struct {
	lits []int
	ws   []int
	k    int
}

func (c bfConstr) String() string {
	return fmt.Sprintf("%v*%v>=%d", c.ws, c.lits, c.k)
}

// bit returns the value of CNF var v (1-based) in assignment a.
func bit(a uint64, v int) bool { return a>>uint(v-1)&1 == 1 }

func (c bfConstr) holds(a uint64) bool {
	s := 0
	for i, l := range c.lits {
		v := l
		if v < 0 {
			v = -v
		}
		if (l > 0) == bit(a, v) {
			s += c.ws[i]
		}
	}
	return s >= c.k
}

// bfModels calls f on every assignment of n vars satisfying cs and fixed.
func bfModels(n int, cs []bfConstr, fixed map[int]bool, f func(a uint64)) {
	for a := uint64(0); a < 1<<uint(n); a++ {
		ok := true
		for v, b := range fixed {
			if bit(a, v) != b {
				ok = false
				break
			}
		}
		for i := 0; ok && i < len(cs); i++ {
			ok = cs[i].holds(a)
		}
		if ok {
			f(a)
		}
	}
}

func bfCount(n int, cs []bfConstr, fixed map[int]bool) int {
	nb := 0
	bfModels(n, cs, fixed, func(uint64) { nb++ })
	return nb
}

// holdsModel evaluates c on a model given as one bool per var.
func (c bfConstr) holdsModel(m []bool) bool {
	s := 0
	for i, l := range c.lits {
		v := max(l, -l)
		if (l > 0) == m[v-1] {
			s += c.ws[i]
		}
	}
	return s >= c.k
}

// checkModel returns a description of the first violated constraint or
// assumption, or "" if m satisfies everything. It works for any nb of vars.
func checkModel(n int, cs []bfConstr, fixed map[int]bool, m []bool) string {
	if len(m) < n {
		return fmt.Sprintf("model has %d vars, want %d", len(m), n)
	}
	for v, b := range fixed {
		if m[v-1] != b {
			return fmt.Sprintf("model violates assumption x%d=%v", v, b)
		}
	}
	for _, c := range cs {
		if !c.holdsModel(m) {
			return fmt.Sprintf("model violates %v", c)
		}
	}
	return ""
}

func randLitInt(r *rand.Rand, n int) int {
	v := r.Intn(n) + 1
	if r.Intn(2) == 0 {
		return -v
	}
	return v
}

// randCNF returns m random clauses of 1..4 distinct vars over n vars.
func randCNF(r *rand.Rand, n, m int) [][]int {
	res := make([][]int, m)
	for i := range res {
		l := min(r.Intn(4)+1, n)
		for _, p := range r.Perm(n)[:l] {
			x := p + 1
			if r.Intn(2) == 0 {
				x = -x
			}
			res[i] = append(res[i], x)
		}
	}
	return res
}

func cloneCNF(cnf [][]int) [][]int {
	res := make([][]int, len(cnf))
	for i := range cnf {
		res[i] = append([]int(nil), cnf[i]...)
	}
	return res
}

func cnfConstrs(cnf [][]int) []bfConstr {
	res := make([]bfConstr, len(cnf))
	for i, c := range cnf {
		ws := make([]int, len(c))
		for j := range ws {
			ws[j] = 1
		}
		res[i] = bfConstr{lits: append([]int(nil), c...), ws: ws, k: 1}
	}
	return res
}

func intsToLits(xs []int) []Lit {
	res := make([]Lit, len(xs))
	for i, x := range xs {
		res[i] = IntToLit(int32(x))
	}
	return res
}

// randPBConstrs returns m random PB constraints (GtEq, LtEq or AtLeast) over
// n vars, together with their brute-force equivalents.
func randPBConstrs(r *rand.Rand, n, m int) ([]PBConstr, []bfConstr) {
	var constrs []PBConstr
	var cs []bfConstr
	for len(constrs) < m {
		l := r.Intn(n) + 1
		lits := make([]int, l)
		ws := make([]int, l)
		sum := 0
		for j, p := range r.Perm(n)[:l] {
			lits[j] = p + 1
			if r.Intn(2) == 0 {
				lits[j] = -lits[j]
			}
			ws[j] = r.Intn(5) + 1
			sum += ws[j]
		}
		switch r.Intn(3) {
		case 0:
			k := r.Intn(sum) + 1
			constrs = append(constrs, GtEq(append([]int(nil), lits...), append([]int(nil), ws...), k))
			cs = append(cs, bfConstr{lits, ws, k})
		case 1: // sum <= k  <=>  sum(negated lits) >= sum - k
			k := r.Intn(sum)
			neg := make([]int, l)
			for j := range lits {
				neg[j] = -lits[j]
			}
			constrs = append(constrs, LtEq(append([]int(nil), lits...), append([]int(nil), ws...), k))
			cs = append(cs, bfConstr{neg, ws, sum - k})
		default:
			ones := make([]int, l)
			for j := range ones {
				ones[j] = 1
			}
			k := r.Intn(l) + 1
			constrs = append(constrs, AtLeast(append([]int(nil), lits...), k))
			cs = append(cs, bfConstr{lits, ones, k})
		}
	}
	return constrs, cs
}

// padVars makes sure pb declares at least n vars (random constraints may not
// mention every var).
func padVars(pb *Problem, n int) *Problem {
	if pb.Status == Unsat {
		return pb
	}
	for len(pb.Model) < n {
		pb.Model = append(pb.Model, 0)
	}
	if pb.NbVars < n {
		pb.NbVars = n
	}
	return pb
}
