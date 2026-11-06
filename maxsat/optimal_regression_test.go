package maxsat

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/dynactionize/gophersat/solver"
)

// Solver.Optimal stripped the relax vars from the models only when a results
// channel was given; with a nil channel, the returned model also held one
// entry per soft clause.

func TestOptimalModelsHaveOnlyProblemVars(t *testing.T) {
	r := rand.New(rand.NewSource(26))
	for it := 0; it < 500; it++ {
		nbVars := r.Intn(6) + 1
		var sb strings.Builder
		m := r.Intn(3*nbVars) + 1
		fmt.Fprintf(&sb, "p wcnf %d %d 100\n", nbVars, m)
		for k := 0; k < m; k++ {
			w := 100
			if r.Intn(2) == 0 {
				w = r.Intn(5) + 1
			}
			fmt.Fprintf(&sb, "%d %d 0\n", w, (r.Intn(nbVars)+1)*(2*r.Intn(2)-1))
		}
		parse := func() solver.Interface {
			s, err := ParseWCNF(strings.NewReader(sb.String()))
			if err != nil {
				t.Fatal(err)
			}
			return s
		}
		direct := parse().Optimal(nil, nil)
		ch := make(chan solver.Result, 100)
		viaChan := parse().Optimal(ch, nil)
		var last solver.Result
		for res := range ch {
			if res.Status == solver.Sat && len(res.Model) != nbVars {
				t.Fatalf("%q: streamed model has %d vars, want %d", sb.String(), len(res.Model), nbVars)
			}
			last = res
		}
		for name, res := range map[string]solver.Result{"direct": direct, "channel": viaChan, "last streamed": last} {
			if res.Status == solver.Sat && len(res.Model) != nbVars {
				t.Fatalf("%q: %s model has %d vars, want %d", sb.String(), name, len(res.Model), nbVars)
			}
		}
		if direct.Status != viaChan.Status || direct.Weight != viaChan.Weight {
			t.Fatalf("%q: direct %v/%d, via channel %v/%d", sb.String(), direct.Status, direct.Weight, viaChan.Status, viaChan.Weight)
		}
	}
}
