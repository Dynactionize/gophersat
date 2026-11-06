package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// PBString wrote the objective as "min: 3 x1+1 ~x4 ;", with no space between
// terms, which ParseOPB (and other OPB readers) cannot parse back.

func TestPBStringObjectiveParsesBack(t *testing.T) {
	r := rand.New(rand.NewSource(33))
	for it := 0; it < 500; it++ {
		n := r.Intn(8) + 2
		constrs, _ := randPBConstrs(r, n, r.Intn(n)+1)
		pb := padVars(ParsePBConstrs(constrs), n)
		nl := r.Intn(n) + 1
		lits := make([]Lit, nl)
		ws := make([]int, nl)
		for j, p := range r.Perm(n)[:nl] {
			lits[j] = IntToLit(int32((p + 1) * (2*r.Intn(2) - 1)))
			ws[j] = r.Intn(9) - 4 // negative weights too
		}
		pb.SetCostFunc(lits, ws)
		out := pb.PBString()
		pb2, err := ParseOPB(strings.NewReader(out))
		if err != nil {
			t.Fatalf("cannot parse back %q: %v", out, err)
		}
		if fmt.Sprint(pb2.minLits, pb2.minWeights) != fmt.Sprint(lits, ws) {
			t.Fatalf("objective %v %v parsed back as %v %v from %q", lits, ws, pb2.minLits, pb2.minWeights, out)
		}
	}
}
