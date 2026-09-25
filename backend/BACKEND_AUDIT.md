# Backend risq audit

Source: 6 parallel audits of `backend/game/games/risq/` (logic errors, panics/bugs, verbose
comments, readability/DRY, subpackage reorg proposal) plus the `.claude/risq.md` doc-update pass.
Straightforward, contained fixes were applied directly and confirmed against current source.
Everything below needs a design decision, touches too much code for a blind fix, or is a proposal
to weigh in on — for discussion.

## 1. Open items

1. **Retaliation cancels a unit's whole order queue**, even mid-retreat, unless it's already
   attacking and `interrupt_current` is off. `risq_unit.go:534-548`. Not a bug — behavior isn't
   specified anywhere. Is this intended?
2. **Stand Ground units only scan their own zone for targets**, not everything within attack range.
   `risq_unit.go:591-593`. Not a bug. Ranged Stand Ground units may be under-reacting by design or by
   omission.
3. **Regions render correctly now, but always carry a $0 gold bonus.** Re-checked against current
   code: the "invisible" half of this finding (from the `.claude/risq.md` doc-update pass) is stale —
   since that doc was written, the frontend gained real region rendering (`risq_region.ts`: perimeter
   borders owner-colored per space, a dedicated REGION map view mode in `risq.ts`, region name +
   gold-bonus labels) and the turn report dialog renders held/claimed/lost regions with their gold
   bonus (`turn_report_dialog.ts:renderRegions`). What's still true, and now more visible for it:
   `stepRegionsSeven` (`risq_map_steps.go`) calls `addRegion` with a literal `0` for `gold_bonus` at
   both its call sites, and nothing else ever creates a region — so every region a player sees now
   visibly displays "+0 gold/turn" in both the map view and the turn report, which reads as a bug
   rather than an unfinished feature. Needs a call: give regions a real gold formula (by size? by
   terrain composition?), or intentionally make them a non-gold ("territory control") bonus and drop
   the "+0 gold/turn" display so it doesn't look broken.
4. **`CreateGame` defaults `map_name` to `"ring"`, but the map-script loader's own fallback is
   `"default.json"`.** Re-checked, still true and unchanged: `risq_create_game.go` falls back to
   `"ring"` when no `map` setting is given; `loadMapScript` (`risq_map_script.go`) separately falls
   back to the embedded `default.json` when the *requested* file can't be read. A client that omits
   `map` gets ring; a client that requests a typo'd/nonexistent map name gets the hexagon default.
   Two different "defaults" for two different failure modes that probably should be the same one.
5. **`ring.json`, `rectangle.json`, `triangle.json` are still untracked in git**, and `default.json`
   is staged (`AM`) — re-checked via `git status`, unchanged since the original finding. Worth
   confirming these are meant to be committed.

## 2. Refactoring

Of the 8 readability/DRY root causes the two audits converged on, only issue 1 below remains open;
the rest were done and verified (embedded `orderableBase`, the `Attackable`-returning target
functions, the `moveOrAct`/`reachableStatus` helpers, the `risq_hex.go`/`risq_board.go` split,
`CombatBonus`/`costJSON.toCost()`, and `allSpaces()` — all confirmed via the full regression sweep
plus hundreds of simulated AI games with zero errors) and have been removed from this list.

1. **Order validation/status/intent logic is split across 5 parallel switch statements per order
   type** (`validateFrontendOrder`, `receiveOrder`, `orderReceivable`, `orderStatus`, `tickIntent`),
   each re-deriving the same checks. Proposed fix: a `map[OrderType]unitOrderHandler` strategy table
   so each order type's full lifecycle lives in one place. **Not done, not attempted** — the riskiest
   remaining item, touches order-resolution directly with the largest surface area (5 functions × ~15
   order types); needs a dedicated go/no-go rather than folding into a batch pass.
2. **Subpackage reorg proposal — recommendation: don't do a full split.** The root package is one
   tangled object graph (units/buildings/orders/space/zone/player/vision) and three of its core
   interfaces (`Orderable`, `Attackable`, `IntentKind`) use **unexported methods** — Go forbids
   implementing an interface with unexported methods from another package, so splitting any of these
   into a subpackage means exporting everything, which is pure churn with no real encapsulation
   gained. Only one clean extraction candidate exists: `risq/mapgen` — turn the map-script DSL into a
   function producing a plain-data `Blueprint` (spaces/terrain/placements/regions), then materialize
   it back into `GameRisq` in root. Real risk to manage: RNG call order must stay byte-identical or
   seeded maps stop matching the sim determinism baselines — map generation is now seed-reproducible
   (fixed and confirmed), so this just needs to be preserved through the extraction, not fixed as
   part of it. Recommendation: extract `mapgen`, skip every other package boundary. The other pure
   file-move win (splitting `risq_map_steps.go`'s geometry/board-mutation code out) is already done
   (`risq_hex.go`/`risq_board.go`). Still open: `risq.go` itself still mixes the action dispatcher,
   behavior setters, gather points, turn resolution, cleanup, vision, and serialization — splitting
   it into `risq_actions.go` (the `execute*` handlers) + `risq_tick.go` (`resolveActiveOrders`,
   `cleanupDeleted`, `recalculateVision`) would be the same zero-risk, zero-rename kind of move.
3. **There are no `_test.go` files anywhere in risq.** The only safety net today is
   `go build`/`go vet` plus the sim determinism harness (`sim/inputs/determinism_check.json`). Worth
   discussing independent of the reorg question.
