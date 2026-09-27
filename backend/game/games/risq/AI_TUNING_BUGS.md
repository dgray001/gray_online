# Bugs found during risq AI v4 tuning session

1. **`garrison` order crashes the sim.** Nil-pointer panic in `RisqUnit.recordDeath`
   (risq_unit.go:143), reached via `resolveHealthDelta` → `resolveActiveOrders`. A
   garrisoned unit dying appears to hit a field (zone/space?) that's unset while
   garrisoned.

2. **`target_priority` (via `set_unit_behavior`) can hang the engine.** With a large
   army (40-90 units), the process stops making progress — CPU time frozen across
   repeated checks minutes apart, unlike a legitimately slow game (which climbs
   steadily and finishes in 5-6 min). Killed manually. Likely an infinite loop in
   target-selection when resolving priority order against some army-size case.

3. **Same seed + same config gives different outcomes across separate process runs.**
   Identical scenario file run 3x: 69/100, 56/100, 63/100 wins — far beyond binomial
   noise for n=100. Violates the codebase's own stated invariant that tick-processing
   order must never affect outcome; likely Go's per-process map-iteration
   randomization leaking into the RNG draw sequence. Means single 100-game runs aren't
   trustworthy for comparing configs.

4. **`buildNextInQ`/`researchNextInQ` can get permanently stuck on a dead entry.**
   `selectFromQueue` picks the top-priority "affordable" queue item without checking
   it's actually produceable (building busy, tech already done). A dead pick is a
   silent no-op but keeps winning every turn, starving everything behind it. Worked
   around by using per-item `research`/`build` actions instead of the shared queue.
