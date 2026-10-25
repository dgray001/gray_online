# AI engine: scans and repeated work

Source audit: 2026-10-25. These are remaining scans, allocations, and repeated work verified in the current implementation, not measured performance rankings. Costs depend on selector shape, source size, and cache state; some paths serve legacy actions rather than every AI rule.

Entries give the source, remaining cost, and a possible improvement. Selector paths live in `ai/`; view paths live in `internal/aibridge/` unless stated otherwise. Suggested changes must preserve snapshot semantics, iteration order, and RNG consumption.

Existing optimizations: keyed iteration for units, zones and unconditioned spaces; a decision-wide world index; cached own/enemy unit lists; order-independent joins for attackables, resources and gatherables; memoized lets; and a 100,000-item selector budget reset per rule. `limit: 1` with zero skip now uses `bestRanked`, preserving tie draws without sorting. Join eligibility checks selector-local impurity rather than model-wide randomness. Assignment carry reuses initialized actors and slots; the v6 converter currently uses it for the ungrouped queued enemy-unit-to-space action.

## Selector resolution

### 1. `actorScope` builds a map of all scoped units per resolve
- Where: `select_filter.go`, `actorScope`, called from `newFilter` for every `actors` selector.
- What: it loops `view.ScopedUnits()` and builds `map[uint64]bool` each time the selector resolves.
- Cost: O(scoped units) per unit-actors resolution, even when `in_set` narrows the result to one unit.
- Triggered by: resolving unit actors; building actors use an ownership predicate, and assignment carry reuses initialized actors.
- Fix: test scope membership without a map (scope is all own units except inside a bucket, where the bucket's member map already exists), or build it once per revision.

### 2. Membership maps are rebuilt for every filter
- Where: `select_filter.go`, `setKeys` (for `in_set`, `not_in_set`, `region_in_set`) and the `bucket: "none"` branch of `newFilter`.
- What: a stored set is a slice of keys, so each filter turns it into a new map. `bucket: "none"` builds the union of every bucket's members.
- Cost: O(set size) per filter construction, O(all bucket members) for `none`. A nested query resolved separately for many targets repeats this construction; memoized hits do not resolve the query again.
- Fix: store the lookup map in `entitySet` when the set is stored, and keep a "bucketed" set up to date instead of rebuilding the union.

### 3. Plain selects eagerly rebuild idle maps
- Where: `select_items.go`, `newCensus`, called at the start of `selectAction.ToOrders`.
- What: plain selects call `IdleUnits()` (a loop over own units) and `IdleBuildings()` (which rebuilds own building views, see item 16) and build two maps before knowing whether `idle` is read. Assignment carry reuses its initialized census.
- Cost: own-unit scanning plus own-building conversion/sorting before matching, per plain select execution. A loop with k plain selects pays k times; building conversion includes producible and queue processing.
- Fix: build the idle sets lazily, only when a selector reads `idle`, and cache them with per-subject updates after orders.

### 4. Ordinary `near` clauses still scan anchors per candidate
- Where: `select_filter.go`, `nearAll`; `select_resolve.go`, `anchorSets` and `resolveNearLocations`.
- What:
  1. For each candidate, `nearAll` loops every anchor location and measures `SpaceDistance`.
  2. Bounds independent of the candidate role are already evaluated once per resolution; only dependent bounds are evaluated per item. Space anchors are already deduplicated.
  3. Anchor selectors are resolved once per enclosing resolution, but repeated resolutions rebuild the same independent anchor list. Clauses with `through` already compute reach with multi-source BFS.
- Cost: worst-case O(candidates x distinct anchors) distance lookups for ordinary clauses, with early exit on a matching anchor; separate anchor-resolution work repeats across resolutions.
- Fix: cache independent anchor results with appropriate binding/revision keys; consider multi-source reach for space-distance clauses. Preserve zone-distance semantics and random evaluation order rather than replacing every clause with BFS.

### 5. Sources with no keyed lookup still scan for `in_set`
- Where: `loop_sources.go` and the other source files; only units, zones and unconditioned spaces implement `itemsOf`.
- What: buildings, resources, gatherables, foundations, attackables, players, regions, mercenaries, and conditioned spaces have no keyed membership path. They materialize their source before membership filtering unless an eligible join supplies candidates. Orders and assignments cannot be stored as entity sets and are not examples of this issue.
- Cost: without a join, source construction plus one selector step per materialized item, regardless of set size. Source-level filters may already narrow that collection.
- Fix: add keyed lookup for sources that support entity membership, preserving scan order. Building lookup also needs cached views (item 16), otherwise it still rebuilds the collection. Do not imply that orders or assignment records can be stored in entity sets.

### 6. Join coverage is narrow and mutable indexes rebuild by revision
- Where: `join.go`.
- What:
  1. Indexes over units, buildings, orders and foundations rebuild on their next use after a view revision change, charging one step per source item. They are cached per selector, not shared between equivalent selectors.
  2. `static()` exists only for units, buildings, attackables, resources, gatherables, foundations and orders. Spaces, zones, players, regions and mercenaries never get an index.
- Cost: eligible unit/building/order/foundation indexes rebuild on the next use after a view revision change. Space/zone equality filters cannot use this join index, although entity membership may use keyed iteration. Independent selectors still qualify when randomness exists elsewhere in the model.
- Fix: track relevant mutations or update affected subjects in place, and add static-source support where safe. Preserve candidate order and local impurity checks. Foundations currently invalidate conservatively even for unrelated orders.

## View and helpers

### 7. `BuildSites` repeats owned-space scans for eligible zones
- Where: `spaces.go`, `BuildSites` calls `buildable`, which calls `ownSpaceCount()`.
- What: empty eligible zones in known unowned spaces reach `buildable`, which calls `ownSpaceCount`. Owned/unknown/enemy spaces and occupied zones avoid this scan; the count itself is fixed within a view.
- Cost: O(eligible unowned zones x spaces), rather than an unconditional scan for every zone.
- Fix: compute it once per view (lazy field).

### 8. `spaceUnitCount` can scan snapshot units per space
- Where: `spaces.go`, `spaceUnitCount`.
- What: it returns an explicit snapshot count immediately, and rejects unknown counts below good visibility. Only good-visibility spaces without an explicit count scan `v.units`, excluding garrisoned units.
- Cost: O(queried good-visibility spaces without explicit counts x snapshot units). This is a normal path: `space.go` serializes unit details instead of `unit_count` at good visibility.
- Fix: tally units per space once per view and look up.

### 9. `KnownResources` loops all zones per category
- Where: `view.go`, `KnownResources`.
- What: each call scans all indexed zones and filters live snapshot resources by category. The world index requests four categories once per decision; unrestricted resource/gatherable source resolution requests all categories again, while category-constrained sources request only one.
- Fix: index resource zones by category once per view.

### 10. `NearestResource` scans every zone per call
- Where: `view.go`, `NearestResource`.
- What: it scans all zones with `gatherableAt`, then checks cached gather counts only for matching nodes. The legacy `gather` action calls it per unit; selector-based gathering does not directly call this helper.
- Cost: O(zones) per unit, O(units x zones) for a gather assignment.
- Fix: iterate a category index of resource and own gatherable-building nodes. `gathererCount` already builds a lazy count map; `recordOrder` invalidates it, so interleaved gather orders can still trigger repeated own-unit scans. Update affected node counts incrementally if profiling justifies it.

### 11. `EnemiesFound` loops all zones on every read
- Where: `view.go`, `EnemiesFound`, used by the `enemies_found` counter.
- Fix: compute once per view.

### 12. `OwnBuildingsIn` rebuilds the foundation map per call
- Where: `spaces.go`, `OwnBuildingsIn`, used by `target_own_buildings` per candidate.
- What: it loops own snapshot buildings and calls `foundationsByLocation()` (item 15) for every requested space.
- Fix: tally own buildings and planned foundations per space once per view.

### 13. `RegionProgress` loops regions and their spaces per call
- Where: `spaces.go`, `RegionProgress`, used by `target_region_progress` per candidate.
- What: already-owned targets return immediately. Otherwise a lazy space-key lookup avoids repeated coordinate indexing, but each query still scans region membership and recounts known owned members until it finds the target's region.
- Fix: cache space-to-region membership and known owned counts/progress per region for the snapshot, preserving explored-membership semantics.

### 14. Attack-load queries still scan order queues
- Where: `internal/aibridge/attackers.go`, `AssignedTo`; `ai/orders.go`, `join.go`, and v6 converter attack-load aggregates.
- What: legacy spread queries scan eligible own-unit queues and deduplicate attacking subjects per target. Converted outside-load counts use the orders selector, which can index eligible equality queries but rebuilds its index after revisions. Rule 181 now carries assignment loads and pairs directly; other conversion paths retain queued-load reconstruction.
- Cost: legacy queries scan eligible queue entries per target. Selector counts pay source/index construction per eligible selector and revision, then matching-bucket/filter work; do not treat every indexed lookup as a full scan.
- Fix: index target-to-distinct-attackers with mutation-aware updates. Extend assignment reuse only where it preserves the relevant legacy semantics; keep outside attackers separate from current-chain allocation.

### 15. Foundation collections are rebuilt on every call
- Where: `view.go`, `foundationsByLocation` and `Foundations`.
- What: `foundationsByLocation` rebuilds a map from planned/ordered foundations and unfinished own buildings, then applies cancellations. Only `Foundations()` additionally scans own units for builders and sorts the returned slice.
- Fix: cache the location map separately from builder counts/sorted views. Invalidate location data on foundation creation/cancellation, and builder data when a unit's effective build order changes.

### 16. `Buildings()` rebuilds own building views on every call
- Where: `view.go`, `Buildings`, `IdleBuildings`, `ownBuildingView`, `producibles`.
- What: own building views are not cached. Each call rebuilds and sorts them, including producible lists, tech checks, and active-order data. `homeLocation` uses this path; legacy `nearFilter.contains` resolves its anchor per candidate, so repeated home-distance checks can rebuild buildings repeatedly. Selector `near` anchor handling is separate (item 4).
- Cost: O(buildings x producibles) per call; O(spaces x buildings x producibles) for one such counter.
- Fix: cache own building views with invalidation for production, queues, claims, and planned research (which can change other buildings' producibles). Cache home for the decision snapshot, and resolve independent legacy filter anchors once per query.

### 17. Collection slices are rebuilt per call
- Where: `spaces.go` (`MatchingSpaces`, `AllSpaces`, `AllZones`), `view.go` (`AllUnits`, `AllBuildings`), `order_queue.go` (`OrderQueue`).
- What: `MatchingSpaces` scans snapshot slots and the space source sorts matches each time; `AllSpaces`/`AllZones` rebuild info slices. `AllUnits` concatenates cached own/enemy slices; `AllBuildings` also rebuilds own views. `OrderQueue` rebuilds entries and sorts subject keys. The world index already reuses spaces/zones within a decision, so repeated selectors do not necessarily repeat the underlying view call.
- Fix: cache the sorted unconditioned list; build the others lazily once and invalidate only on the changes that affect them.

### 18. Bucket membership and graph-neighbor work
- `isBucketed` (`rules.go`) loops every bucket for every unit; legacy fill and army loops pay O(buckets x units). Fix: keep a unit-to-bucket map.
- `SpaceNeighbors` (`distance.go`) allocates, sorts, and deduplicates on each call. `SpaceDistance` already caches a complete BFS row per source, but constructing another source row repeats neighbor work. Cache sorted neighbor lists per view. Dense distance storage is a profiling-dependent alternative to the current coordinate-keyed rows, not evidence of a separate defect.
