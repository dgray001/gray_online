# Order lifecycle & gather point — status and plan

Working notes from implementing the gather point backend, since it surfaced a real gap in how
orders are validated. Backend only; nothing sets a gather point yet (out of scope, deliberately).

## The three order checkpoints and what each should actually do

- **`validateFrontendOrder`** (risq_order.go): wire-boundary sanitizer. Runs once, at submission,
  against an untrusted client payload. Rejects malformed input (bad target decode, subject you
  don't own, wrong unit type for the order category) as of that instant. Must never be relied on
  for anything time-sensitive — players submit, then wait for everyone else before the batch
  resolves, so real time (and state changes) can pass between submission and resolution.
- **`orderReceivable`** (risq_unit.go / risq_building.go): the "is this still legal, given current
  state" gate, checked once per subject at the moment the order is actually dispatched into that
  subject's queue (`resolveActiveOrders` in risq.go). This is the boundary meant to re-check
  anything validation only confirmed at submission time. Per the `Orderable` interface's own
  doc comment, its contract is "accept/reject by current state."
- **`orderStatus`**: the ongoing, every-tick continuation check for an order already in progress.
  Naturally overlaps with `orderReceivable` (same predicate, checked repeatedly instead of once) —
  Garrison/Attack/Build already have matching pairs in both, correctly.

## Gap found

`orderReceivable` has no real case for `OrderType_UnitGather` or `OrderType_UnitRepair` — both
fall through to `default: return true`. `orderStatus` does have real logic for them (resource
stock, building damage/construction), which is what currently prevents any crash — a stale
Gather/Repair order just gets silently dropped on its first per-tick check before `tickIntent`
ever sees it. But it's misreported: the order ends up marked `executed` (accepted, then instantly
self-completed) instead of `cancelled` (never actually accepted) in `resolveActiveOrders`'
post-loop accounting (risq.go:341-355). This is a pre-existing gap in current behavior, not
something new — closing it benefits normal play too, not just gather points.

**Fix**: add real `OrderType_UnitGather` / `OrderType_UnitRepair` cases to `orderReceivable`,
mirroring `orderStatus`'s existing InProgress conditions.

## Vision principle (new rule, applies to both functions)

`orderReceivable` and `orderStatus` must never use information outside the acting player's own
current vision. Concretely: don't read live `zone.resource` / `risq.buildings[...]` state directly
unless the player currently has at least the required visibility tier for that zone. This is
already violated in spirit for `OrderType_UnitAttackUnit`/`AttackBuilding`'s ongoing pursuit logic
in `tickIntent` (paths toward a moving enemy's live current zone every tick, no vision check at
all) — that's a bigger, separate design problem (what should an in-progress attack order do once
vision is lost — cancel? hold last known position?) and is explicitly **deferred**, not part of
this pass.

Within scope for gather points: Gather and Repair targets are static (a resource in a zone, a
building that doesn't move), so this is tractable now.

### Fog tier uses the cache, not a skip

Checked `zone.toFrontend()` (risq_zone.go) for the established convention: at exactly the Fog tier
(`VisibilityFog`, "remembered but not currently visible") it reads `resource_cache`/
`building_cache` (risq_space.go, populated by `risq_vision_cache.go`). At Poor and above it reads
live state directly — the frontend gets real-time streamed updates for anything currently visible,
so live state *is* what the player already knows. Below Fog (`VisibilityUnexplored`) there is no
cache entry and no information at all.

So the rule per tier:
- `>= VisibilityPoor` (building/resource) or `>= VisibilityGood` (unit): read live state.
- `== VisibilityFog`: read the player's cached snapshot (`RisqResourceCache` / `RisqBuildingCache`
  already carry every field needed: `resources_left`; `player_id`, `cs`, `under_construction`).
- `== VisibilityUnexplored`: no info, must fall back.

Units have no cache anywhere in the codebase (matches `zone.toFrontend()`, which only shows units
at Good+), so `GatherObject_Unit` resolution stays "Good tier or nothing" — no fog-tier answer is
possible for units regardless.

## Plan (not yet implemented)

1. Add two helpers on `*RisqZone`, reusing the existing `cacheRisqResource`/`cacheRisqBuilding`
   constructors so the live and cached paths return the identical shape:
   ```go
   func (z *RisqZone) resourceKnownTo(player_id int) (RisqResourceCache, bool)
   func (z *RisqZone) buildingKnownTo(player_id int) (RisqBuildingCache, bool)
   ```
   (Unexplored → not ok; Fog → cache lookup; Poor+/Good+ → construct on the fly from live state.)
2. Retrofit `orderReceivable`: add `OrderType_UnitGather`/`OrderType_UnitRepair` cases using these
   helpers instead of reading `zone.resource`/`risq.buildings[...]` live and unconditionally.
3. Retrofit `orderStatus`'s existing Gather/Repair cases to use the same helpers (currently reads
   live state with no vision check at all).
4. Update `resolveGatherPointOrder`'s `GatherObject_Resource`/`GatherObject_Building` branches to
   use these helpers instead of the live-only reads currently there.
5. Garrison and the self-building branch stay on live state always — always your own building,
   which is always inherently visible (owns its own vision source), so the cache is irrelevant.
6. Deferred, explicitly out of scope: AttackUnit/AttackBuilding's live tracking of a moving enemy
   through fog in `tickIntent`. Needs its own design pass (what happens to an in-progress attack
   order when vision is lost) — not touched here.

## Already implemented

- `risq_gather_point.go`: `RisqGatherPoint` struct — `location_kind` (Space | Zone, required),
  `location_id` (space or zone coordinate_key), `object_type` (None | Unit | Building | Resource,
  optional), `object_id` (internal id, only meaningful if `object_type != None`).
- `(*RisqGatherPoint) resolveOrder(risq, b, unit) *RisqOrder` — always returns a real order
  (falls back to Move on `location_id`, which is guaranteed valid since spaces/zones are static).
  Currently builds Gather/Repair/Garrison/Attack candidates via hand-written eligibility checks
  that need replacing per the plan above (step 2) so it calls `orderReceivable` instead, per:
  construct the ideal candidate → `orderReceivable` → if rejected, construct the Move fallback →
  `orderReceivable` again (always true in practice) → if somehow still rejected, no order at all.
- `RisqBuilding.gather_point *RisqGatherPoint` field (optional, nil = no rally point).
- Hook in `RisqBuilding.tickExecute`'s `ProducibleKind_UNIT` branch: after a unit is created and
  placed, if `b.gather_point != nil`, calls `unit.receiveOrder(b.gather_point.resolveOrder(...), risq)`
  directly. No `player.active_orders`/turn-report bookkeeping (that's the global right-panel
  order list plumbing, which has no consumer yet since nothing sets a gather point) — the order
  lives only in the unit's own `order_queue`, which is sufficient for `tickIntent` to act on it
  and for that specific unit's own frontend order list to show it.
  Verified the tick loop (`resolveActiveOrders`'s `for {}` in risq.go:320-339) rebuilds `orderables`
  fresh from `r.allOrderables()` every iteration, not once for the whole turn — so the new unit
  (already added to `risq.units`/`player.units` before `receiveOrder` is called) is automatically
  picked up starting the very next tick, no special-casing needed.

## Not yet designed (deliberately)

How `gather_point` actually gets set — no order type, no wire encoding, no frontend UI. Explicitly
deferred until the backend resolution logic above is finished and correct.
