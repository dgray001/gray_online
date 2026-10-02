# Risq backend audit

Date: 2026-10-01. Source reviewed in the current working tree; no Git commands used.

## Scope and original audit verification

Reviewed the core `backend/game/games/risq` engine, including orders, ticks, construction, production, combat, gathering, repair, renewal, mercenaries, ownership, vision, serialization, reports, metrics, and state hashing; also `internal/defs`, `internal/mapgen`, and gameplay/map JSON configuration.

Excluded `ai`, `sim`, `ttk`, `internal/aibridge`, AI configuration, and the root `ttk.go` utility. Shared game-base behavior was inspected only to trace game termination. Compilation necessarily includes imported dependencies; that does not constitute an audit of excluded packages.

- `go test ./game/games/risq ./game/games/risq/internal/defs ./game/games/risq/internal/mapgen`: passed compilation; all three reported `[no test files]`. **Zero tests ran.**
- `go vet` for those same three packages: passed.
- Parsed all 17 gameplay/map JSON files successfully.
- Findings below are static code traces, not gameplay reproductions. No test harnesses or application-code edits were made.

Priority: P1 = gameplay determinism; P2 = functional correctness or information exposure; P3 = latent/configuration limitations. Suggested fixes are proposals only.

## Current defects

Remaining: 4 findings. Clear fixes: #1 (paused) and #3 (aggregate policy agreed). Findings #5 and #6 require a behavior/API decision. Resolved findings, unsupported spectators, intended cluster-size behavior, and the final-state broadcast finding (tracked in PLANS.md) have been removed.

### 1. P2: An unaffordable replacement cancels the previous valid order

Evidence: [tick.go:125](/home/yodan/software/websites/gray_online/backend/game/games/risq/tick.go:125), [unit_orders.go:10](/home/yodan/software/websites/gray_online/backend/game/games/risq/unit_orders.go:10), [building.go:209](/home/yodan/software/websites/gray_online/backend/game/games/risq/building.go:209).

Delivery clears previous orders after `orderReceivable`, but before `receiveOrder`. Affordability is checked inside `receiveOrder`. For example, a villager with a valid move/gather order receives a build replacement that passes structural/state checks but cannot be afforded at receipt. Its previous order is cancelled, then the replacement is rejected, leaving it idle. The failure has already changed subject state.

Proposal: receive the replacement successfully before clearing previous orders, excluding the newly accepted order from clearing. Production/research orders are already clear-immune, so this clearing path does not need their refunds to fund receipt.

### 3. P2: Group-order outcome depends on which subject resolves last

Evidence: [order.go:148](/home/yodan/software/websites/gray_online/backend/game/games/risq/order.go:148), [player.go:157](/home/yodan/software/websites/gray_online/backend/game/games/risq/player.go:157), [tick.go:83](/home/yodan/software/websites/gray_online/backend/game/games/risq/tick.go:83).

`nextOrder` removes a resolving subject while more than one remains, recording no aggregate outcome. The last subject sets either `executed` or `cancelled`. For a group move where one unit is already at its destination and another cannot reach it when both resolve their order status, unit-map iteration determines whether the shared order is reported as executed or cancelled. The same pattern exists in explicit cancellation/death subject removal.

Agreed policy: once every subject has resolved, mark the shared order executed if any subject completed, otherwise cancelled. Retain whether a subject completed while removing resolved subjects; keep the order active while any subject is still working.

## Latent defects and configuration limitations

### 5. P3: Mirror accepts steps whose mutations it cannot replay

Evidence: [mapgen/mirror.go:20](/home/yodan/software/websites/gray_online/backend/game/games/risq/internal/mapgen/mirror.go:20), [mapgen/mirror.go:56](/home/yodan/software/websites/gray_online/backend/game/games/risq/internal/mapgen/mirror.go:56).

The wrapper accepts any registered step, but records only changed space terrain and resources newly added to previously empty resource slots. Resource removals/replacements, buildings, units, zone overrides, and region changes are not replayed. For example, wrapping resource-min-spacing can remove resources in the original run without symmetric removal elsewhere. This concerns the wrapper's accepted API, not a demonstrated shipped-map failure.

Proposal: restrict supported wrapped steps/mutations with an explicit error, or extend the replay representation for the required mutation types.

### 6. P3: Terrain-border width and shape semantics need clarification

Evidence: [mapgen/terrain.go:264](/home/yodan/software/websites/gray_online/backend/game/games/risq/internal/mapgen/terrain.go:264).

The border uses axial distance from the origin with threshold `board_size-width`, rather than distance from the carved board boundary. On a full hexagon, width zero still paints the outermost ring and width one paints two rings. On rectangles, triangles, and rings, it does not generally follow the shape's boundary or inner hole. If width means boundary layers, this is an off-by-one plus a shape limitation; otherwise document the radial-band convention.

## Policy questions and checks without a finding

- Enemy `researched_techs` and population limits are always serialized ([player.go:227](/home/yodan/software/websites/gray_online/backend/game/games/risq/player.go:227)). These reveal global enemy progress. Their public/private policy is not stated clearly enough to classify this as a confirmed defect.
- `r.players` is a slice, so metrics indexing by player ID and player iteration in the state hash are valid. The initial suspected metrics-indexing defect was rejected after checking the declaration.
- Frontend batches enter through format/subject validation, then the common delivery path. Current stance, building auto-attack, gather-point, and auto-gather producers use `addSyntheticOrder`; building auto-attack requests prepend through that path.
- Produced units receive half their configured turn stamina and first participate in the next tick's fresh actor snapshot. Auto-gather is issued after completion using that tick's construction intents, with deterministic builder-ID sorting; extra workers contend through gather-slot allocation.
- Gather-slot allocation, population-completion winners, new unit IDs, foundation winners/IDs, and garrison admission have explicit deterministic selection. These checks do not establish determinism for the separate issues above.
- Player-start resource IDs are drawn into the shared planned slots before applying them to individual starts; equivalent slots keep the same selected resource type. Center/edge selectors are applied in planned and fallback placement.
- Board carving/nil handling, ownership computation, range/pathfinding, deferred health/renewal/tech application, costs/refunds, and map expression/step dispatch were reviewed. No additional definite finding is claimed for those paths.

Suggested order of work: fix gather-point timing first; then combat/order accounting and visibility/update delivery. Add focused reproductions only when implementing the chosen fixes. Existing compile/vet success is not gameplay validation.
