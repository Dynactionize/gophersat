# gophersat — dynactionize fork

Hard fork of [`github.com/crillab/gophersat`](https://github.com/crillab/gophersat),
maintained by dynactionize because upstream no longer merges outside contributions
(last release v1.4.0, 2023). Based on upstream `master` at `14b4f8c` (July 2025,
cosmetic modernisation on top of v1.4.0).

Module path: `github.com/dynactionize/gophersat`. Used by the dynizer constraint
engine (`implementations/constraint/constraintsat`) for incremental SAT with
assumptions, unsat cores and backbones.

## Fixes (all in `solver/`)

1. **`Assume()` erased the problem's unit clauses** (`solver.go`). `New()` binds
   `Problem.Units` at level 1 on the trail; units are never stored as clauses.
   `Assume()` starts with `cleanupBindings(0)`, which unbinds every level ≥ 1 trail
   var — so the first `Assume` call irrecoverably deleted the formula's unit facts.
   Later calls could return SAT for UNSAT queries and models violating the units.
   Minimal repro: `{x1} ∧ {¬x1∨x2}` + `Assume(¬x2)` returned SAT.
   Fix: `Solver.rootUnits` captured in `New()`, rebound at the top of `Assume()`,
   with a contradiction check for assumptions against root units.

2. **`cleanupBindings()` double-inserted vars into the VSIDS queue** (`solver.go`).
   A half-finished refactor inserted each unbound var once in-loop and again via a
   `toInsert` replay. `queue.insert` was not idempotent, so duplicates corrupted the
   heap; a `contains()` false-positive then skipped reinsertion of an unbound var,
   `chooseLit()` drained the queue, and `search()` declared **Sat with unbound
   variables** — models silently falsifying clauses (`Model()` defaults unbound to
   false). Fix: insert exactly once.

3. **`queue.insert` made idempotent** (`queue.go`) — defence in depth so no other
   call path can corrupt the heap invariant.

Regression tests: `solver/fork_regression_test.go` (all three fail on upstream).

## Impact

Upstream v1.4.0 under hot `Assume` reuse: 26% (corpus) to 67% (VOD) of SAT answers
carried invalid models; 68 wrong-SAT statuses in one benchmark; one genuinely forced
backbone literal lost. Fork: 0 invalid models, 0 wrong statuses, exact backbones;
full-VOD verified greying drops from ~5.9ms to ~0.83ms/op. Measurements and the
differential harness live in the dynizer repo under `testclients/gophersat-spike/`.
