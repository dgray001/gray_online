# Risq tick actions and replay

## Objective and scope
Record what each unit intended and what actually happened, with an explicit tick number on every TickAction. First use: show the specific target chosen by Attack Zone/Space. Subsequent use: replay the resolved turn tick by tick, including blocked attempts, incoming damage, movement, and death.
This is a proposed design, not an implementation. It uses an event journal with a turn-start baseline and authoritative tick-end state changes; the frontend does not rerun game rules.
Retain the latest resolved turn initially. Persistent multi-turn replay, cross-player omniscient playback, and continuous animation between ticks are separate features.

## Existing behavior and integration points
- `tick.go:resolveActiveOrders` delivers orders, computes intents, resolves melee meetings, allocates contested work, executes actors, settles deferred effects, cleans deleted actors, and advances the turn.
- `unit_tick.go:tickIntent` chooses area-attack targets without replacing the original order. `intent.go:resolveCost` can erase a proposed intent for insufficient stamina.
- `melee_meeting.go` can replace a move intent with an attack plus half-move cost. `allocateRenewStamina` can adjust or discard a renew intent.
- `unit_tick.go:tickExecute` has early returns for gather capacity, construction races, invalid repair/renew targets, and garrison contention. These are outcomes to record explicitly.
- `repair.go:resolveRepairs`, combat health settlement, building renewal, research completion, and gather-point assignment happen outside individual execution methods.
- `building.go` executes production and garrison attacks; a garrisoned unit can attack without its own intent. Record each building-initiated unit attack and its actual stamina expenditure in that unit's history, including skipped contributions. Unit replay also includes incoming building attacks.
- `unit.go:cleanupDeleted` removes units before the next snapshot. `risq.go:startNextTurn` regenerates stamina before broadcasting; the live snapshot is not the resolved tick-end state.
- Frontend snapshots enter through `transport/snapshot_types.ts`, `transport/snapshot_conversion.ts`, and `application/session.ts`. Map orders are drawn by `rendering/board/order_overlays.ts`.

## Record ownership
Keep a per-unit `tick_actions` array, backed by a turn journal that also retains deleted units. Live-unit serialization may expose the array, but the journal is the authoritative replay payload so deleted units and completed orders remain inspectable.

## Proposed value schema
All records contain immutable values, never pointers to live intents, orders, resources, or entities. Go uses explicit typed structs and existing hand-written `ToFrontend` serialization; TypeScript uses discriminated unions.
- `TickAction`: `tick`, per-unit `sequence`, `order` (optional), `intent`, `execute`. Its containing journal supplies `turn_number`; actor identity is supplied by the containing unit history.
- `order`: original order type, original target ID, optional internal ID, and source (`player`, `stance`, `gather_point`, or `building_garrison`). Synthetic orders can lack an internal ID.
- `intent`: discriminated action kind and action-specific payload, actor location at tick start, available stamina, min/max requested stamina, and proposed target. Include movement destination/path and chased target where applicable.
- `resolution`: within `intent`, retain changes to the proposal, final executable kind/payload, allocated stamina, half-move sunk cost, and a reason when it was changed or blocked.
- `execute`: discriminated outcome (`executed`, `partial`, `blocked`, `skipped`), stable reason code when needed, actual stamina spent, actual action target, and action-specific effects.
- `EntityRef`: entity kind plus internal ID and player ID where applicable. Unit/building ID counters overlap, so an ID alone is insufficient. Resource references use zone key and resource config ID.
- `Location`: space and zone coordinates, optional garrison building reference. Capture target coordinates at intent time and execution time rather than looking up current coordinates during replay.
- Effects: movement/garrison changes; gathered category/amount; construction progress and created building; repair contribution; renew contribution; damage by swing; created/deleted entities; production/research progress. Store authoritative tick-end state separately from attempted contributions.
- Example reason codes: `insufficient_stamina`, `gather_capacity`, `resource_depleted`, `foundation_lost`, `garrison_full`, `target_invalid`, `no_reachable_target`, `population_cap`, `own_action_preempted_garrison_attack`.
Do not store backend functions or executable logic in the payload. Store the inputs, decisions, and results of that logic.

## Tick numbering and attempts
Tick 0 is the baseline after order receipt and before the first intent pass. Execution ticks start at 1, matching existing combat records. Reserve `current_tick + 1` before intent generation so discarded attempts have a tick too.
Record attempts with meaningful work or a concrete blocked decision, not empty idle entries for every unit. Within a tick, `sequence` preserves multiple decisions or garrison swings for one actor; it does not imply priority across actors.
The loop currently performs a final intent pass with no executable intents and does not increment `current_tick`. Retain meaningful terminal attempts as an explicitly marked terminal frame, with its own tick number and no world effects; do not silently discard them or change simulation tick progression.
Order completion/cancellation discovered by `nextOrder` needs a decision record when relevant. Receipt-time rejection is a separate tick-0 event, not an attempted tick execution. Audit order-status exits as well as intent and execution exits.

## Capture lifecycle
1. At entry to `resolveActiveOrders`, replace the previous journal and clear existing actor histories once. Capture pre-receipt state if showing receipt effects; capture the replay baseline after delivery. Newly created actors start with empty histories and a spawn event.
2. Open an attempt when an order/action is selected. Snapshot its proposed intent before `resolveCost` can erase it. Record explicit blocked decisions where no low-level intent is formed.
3. Record melee replacement and allocation decisions without changing their rules. Preserve proposed and final intents, including renew allocation and construction/garrison winners.
4. In execution branches, record the actual result and reason for each early return. Derive actual stamina spent from authoritative changes, not allocated cost: existing early returns do not all spend stamina.
5. Record individual combat swings and incoming hits, including building/garrison swings. An idle unit can still have state changes and incoming effects without an outgoing action.
6. After repairs, net health/death, renewals, research, and automatic assignments settle, finalize frame effects and actor states. Add created foundations/units to the frame even though they were absent from the tick's original actor list.
7. Archive actor identity and history before cleanup removes it. Capture end-turn ownership, income, mercenaries, vision, elimination, and next-turn regeneration as boundary events rather than attributing them to the last combat tick.
8. Freeze the completed journal before broadcast and preserve it through reconnects until the next resolution begins. Do not clear it in `refreshStamina` or snapshot reads.

## Replay envelope and state
Proposed snapshot addition: optional `last_turn_replay` containing schema version, resolved turn number, execution tick count, terminal frame metadata, actor catalog, baseline, per-unit action histories, tick state changes, and boundary events.
The actor catalog contains enough display identity for actors that later died. The baseline contains replayable own-unit state (location/garrison, health/max health, stamina, existence, and relevant order context), plus permitted local terrain/entity context.
Per-tick state changes are authoritative before/after values or patches for changed fields. Include incoming effects and deaths even if the unit had no intent. Reconstruct tick N by applying whole frames to the baseline; never infer health solely from damage records.
For contextual board playback, record changed buildings/resources and spawned entities too. Production, repair aggregation, tech stat changes, and garrison state make a unit-only action list insufficient for exact replay.
Shared effects have stable references and one authoritative owner in the frame; action entries may reference them. Do not double-apply a damage event when it appears in attacker and defender inspection.
Retain only one turn in backend memory initially. Avoid full board snapshots per tick and avoid recording every recalculated path when only the next step and chased target are needed for playback. Measure journal bytes and serialized bytes with representative large games.
Do not silently truncate records needed for exact replay. Any later retention limit must expose completeness explicitly. Journal recording must not consume RNG, affect actor ordering, or enter gameplay state hashes.
Repair history records each worker's intent and actual stamina expenditure. Record aggregate settled healing and resource cost on the building, without attributing individual shares of settled healing/cost to workers.

## Visibility contract
Owners receive their units' full recorded histories, including deleted units. Other players receive action details only under a defined spy policy; ordinary vision alone does not reveal intent/order data.
Existing `vision_cache.go:showOrdersTo` grants access by ownership or Spy visibility in the actor's current space. Applying that once to a whole history would reveal earlier activity in other spaces and fails for deleted/garrisoned actors.
Proposed policy: capture per-entry viewer eligibility from the actor's action-time space using the visibility state available to the resolver. For garrisoned actors use the building's space. Keep eligibility on archived records rather than recomputing against surviving units.
Vision currently recomputes at turn end, not every tick. This proposal therefore uses the existing resolution-time visibility values; physically following spies during replay would require a separate vision-system change.
Filter baseline, frames, actor catalog, targets, and effects server-side. Permission to inspect an actor does not automatically reveal a hidden target's full stats or other hidden entities. References may be redacted while retaining the permitted action description.
Visibility gaps must appear as unknown intervals, not inferred motion through hidden spaces. Never serialize the full journal and filter only in the UI. Spectator behavior must follow the game's existing viewer policy explicitly.

## First frontend application: area-attack targets
Read the latest relevant TickAction for the selected unit and originating area order. For an executed attack, draw toward the captured actual target location; for a chase, use the recorded chased target and movement outcome.
Keep area intent visually separate from resolved action: outline/tint the ordered zone or space, and use an arrow for the specific attack/chase. This avoids representing an entire area solely by its center.
Historical action endpoints describe the resolved turn, not a prediction of the next turn. Label their turn/tick and show completed-order actions through history inspection rather than attaching them to unrelated new orders.
If no specific target was chosen, show the area marker without inventing an enemy arrow. Dead targets use archived identity/location. Exact colors, marker geometry, and default visibility are frontend design choices still to review.

## Replay frontend
Add a replay controller/model independent of the live `RisqSession` and pending order model. It owns the journal, reconstructed frame state, selected tick, play/pause, speed, and step/scrub controls.
Render replay through a read-only board data source. Pan/zoom and inspection remain available; order targeting/submission must use live mode only. Leaving replay restores the live snapshot and existing pending orders.
Each tick displays its intents, outcomes/reasons, incoming effects, and settled state together. Playback advances whole frames, not actors in map iteration order. Baseline, terminal attempt frame, and turn-boundary effects are visibly distinct.
Replace or exit replay when a newer resolved-turn journal arrives; preserve the selected frame across same-turn snapshot refreshes. Missing payloads disable replay cleanly. New controls/components follow existing scaffolding and event conventions.

## Implementation stages and affected code
1. Define typed action/intent/outcome/effect records, turn journal, visibility contract, and immutable snapshot helpers in focused new Risq backend files. Add history ownership to units and a journal field to `GameRisq`; add building capture where needed for unit context.
2. Instrument proposal, rejection, intent transformations, execution exits, combat, repair settlement, spawns/deaths, and boundary effects. Main edits: `tick.go`, `intent.go`, `unit_tick.go`, `unit_orders.go`, order queue code, `melee_meeting.go`, `unit_intent.go`, `building.go`, `combat.go`, `repair.go`, `unit.go`, and `risq.go`.
3. Serialize filtered latest-turn replay with the normal snapshot/update path. Audit game/player/space serializers and `vision_cache.go` so histories cannot enter fog caches or bypass visibility. Preserve reconnect and game-ended access.
4. Add frontend wire unions and conversion, history lookup, and resolved area-target rendering. Main edits: transport snapshot types/conversion, model types, session, and `rendering/board/order_overlays.ts`.
5. Implement frame reconstruction and replay controls, then integrate the board renderer and selection/panels with a read-only replay data source. Cover resources/buildings required for context; wider full-board cinematic playback can follow separately.
6. Measure payloads and memory, verify deterministic simulation results, and manually review the target markers and replay interaction. Each stage should be independently reviewable; frontend replay depends on settled backend capture semantics.

## Verification and acceptance
- Extend appropriate existing tests under `backend/game/games/risq/tests/`; create files only for a new package or when the appropriate existing file exceeds 400 lines.
- Capture: multiple ticks/targets for one area order; different targets for subjects sharing an order; original versus melee-replaced intent; insufficient stamina; terminal blocked attempt; order completion and synthetic orders.
- Execution: contested farm capacity, depleted gather source, losing foundation/garrison races, partial gather/build/renew, unaffordable repair, population contention, and garrison swings skipped for an own action.
- Replay: incoming damage on idle units, simultaneous death/actions, damage/heal netting, unit/building creation, garrison movement, tech completion, archived dead units/targets, and turn-start regeneration separated from combat state.
- Privacy/lifecycle: owner versus spy versus ordinary vision; hidden targets and travel gaps; garrisoned/deleted actors; first turn, reconnect, repeated snapshots, next resolution reset, and final game turn.
- Determinism: recording leaves simulation hashes/results unchanged; canonical actor/effect ordering yields consistent serialized frames. Reconstructed tick-end state matches backend snapshots at the same capture boundary.
- Run `gofmt -w .` and `go test ./...` from `backend/` for implementation; run `npm run check` for frontend changes. Verify UI playback, scrubbing, return-to-live behavior, and pending-order preservation in a browser.

## Decisions to settle before implementation
Confirm historical spy eligibility versus granting a current spy the entire latest-turn history; minimum contextual board state for replay; and whether terminal attempts get a separate selectable frame. The proposals above favor bounded historical disclosure, exact own-unit replay, and explicit terminal frames.
