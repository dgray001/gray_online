# RISQ replay events: status and remaining work

## Current status

| Event | Current implementation | Remaining work |
| --- | --- | --- |
| Production | Unit creation, building completion and research markers with detail panels | View-mode filtering and report links |
| Battle | Crossed-swords markers, grouped combat, participant icons and proportional kills/razes/damage bars | Filtering and report links |
| Disruption | Exactly one interrupt-current marker per turn-report failure row, with the same order label and reason | Filtering rule and report links |
| Player eliminated | Red X markers from report/journal eliminations; last observed locations or unexplored/fog centroid; seek to replay end | Filtering and report links |

- `REPLAY_EVENTS_ENABLED` in `risq.ts` is currently `true`. One `if` in `createReplayEvents()` gates the entire event layer; `false` removes both drawing and event input handling.
- Current markers are 64 by 64 HUD units, at the centroid of their distinct associated zones. Panel text is 18px; the panel is 280 by 174 HUD units.
- Production Jump seeks to the completion tick. Battle Jump seeks to the first intent stop (`first_tick - 0.5`). Both pause playback and synchronize the slider.
- Marker hover permits wheel zoom. Popup hover consumes wheel input. Existing HUD controls take priority over event input and drawing.
- Events are cached on live snapshot replacement through `RisqSession.getReplayEvents()`. Seeking does not regenerate them; replacing the event array resets panel selection.
- No event view-mode filtering exists yet. Intended filtering: Economy hides Battle/Elimination; Military and Ownership hide Production. Disruption's view-mode rule is not specified.
- Turn-report UI and its existing summaries have not been upgraded. Mercenary arrival events are not implemented; hiring occurs at settlement rather than an execution tick.

## Scope

- Implement entirely in the frontend using existing snapshots, turn journals and reports.
- Draw markers and their detail panel on the existing canvas; route input through the existing canvas component system.
- Preserve existing HTML, CSS layout and dialog positioning.
- Production, Battle, Disruption and Player eliminated markers exist. Filtering and report links remain.

## Event types

| Type | Contents | Icon |
| --- | --- | --- |
| Battle | Related combat actions and outcomes | `icons/swords128` |
| Production | Units created, technologies researched, buildings completed | Corresponding unit, technology or building production image |
| Disruption | One event per `report.orders.failures` row, with its recorded reason | `icons/interrupt_current128` |
| Player eliminated | A player eliminated during the reported turn | `icons/close64`, tinted red |

The current `ReplayEvent` interface contains `kind`, `tick`, `player_id`, `title`, `icon` and `locations`. It has no event IDs, tick intervals, entity references or structured outcome details yet.

## Implemented: Production

1. Derive events on live snapshot replacement, independently of the currently displayed replay snapshot.
2. Detect completed buildings from construction transitions, including foundations started and finished within one tick.
3. Correlate newly created units with successful production actions; entering vision alone must never count as creation. Handle garrisoned spawns.
4. Detect completed research from technology transitions and correlate it with the researching building. Resolve names and images from available frontend data.
5. Store tick, player, title, icon and recorded locations. Separate completions remain individually accessible; producer/product instance references are not stored in the event model.
6. Show production markers in replay mode. Clicking opens a small canvas panel showing the completion, location and tick.
7. A jump action pauses playback, updates the replay slider and displays the state after the completion tick.

## Canvas rendering and interaction

- Compute each marker's centroid from the distinct associated zone centers in board coordinates. Use the shared transform for drawing and hit testing.
- Draw markers above board contents and below existing HUD panels. Keep their icons upright and legible while panning, zooming and rotating.
- Use existing canvas button and panel primitives. Register interaction with `application/input/pointer.ts`; HUD controls take precedence over map markers.
- Consume marker and detail-panel clicks so they cannot select the board, issue orders or start a drag.
- Position the detail panel beside its marker and constrain it to the visible canvas. Overlapping markers must allow access to every event.
- Clear event selection and interaction state on replay exit, new turns, resync to another journal, reinitialization and disconnect.

## Remaining increments

- Battle markers are implemented: executed `attack` and `receive_damage` records involving the local player are connected when ticks differ by at most 2 and any associated zones share a space or face each other across a border. Grouping is transitive and may span the entire map. Custom seams replace natural border connections. Distinct zones determine the centroid; the panel currently says `hello battle`.
- Disruption: derive exactly one event per turn-report failure row, including identical rows. Match rejection receipts one-to-one for subject location/tick; mercenary failures use the requested hire zone. Do not generate extra events for blocked/cancelled actions absent from the failure list.
- Player eliminated: use report scores and journal elimination transitions. Use the player's latest recorded zones; otherwise use the unexplored-space centroid, then the fog centroid. If neither exists, use the board centroid. Fallback positions do not create zone highlights.
- End-of-turn events seek to the existing replay endpoint (`tick_count`); no separate settlement replay stops are required.
- Apply the server's existing visibility restrictions. Missing or redacted data must remain unknown; frontend inference cannot establish hidden combat or production.

## Pending: turn report integration

- Reuse event descriptions so report entries can open the corresponding replay detail and seek when a supported stop exists.
- Keep the existing report sections and totals. Add an Events section directly after Summary, ordered chronologically, with expandable event details and a View in replay button. Include locationless elimination events here.
- Existing combat report `target_id` identifies an entity type, not an instance. Do not use it as a replay entity link.

## Validation

- Check construction completion, multiple completions, research, garrisoned creation, later deaths and enemies entering vision against journal fixtures.
- Verify centroid placement, overlapping events, click consumption, slider synchronization and cleanup; run `npm run check` and a frontend bundle build.
- Verify the UI in-browser: pan, zoom, rotation, HUD scaling, replay entry/exit and existing dialog centering. Require user verification when browser access is unavailable.

## Current files and pending changes

All paths below are relative to `frontend/src/components/game/games/risq/`.

| File | Current state or pending change |
| --- | --- |
| `model/replay_event.ts` | Shared Production/Battle event interface |
| `application/replay/production_events.ts` | Production extraction from executed actions and tick deltas |
| `application/replay/battle_events.ts` | Attack extraction, two-tick spatial grouping, border/seam connectivity and distinct zones |
| `application/replay/disruption_events.ts` | One event per report failure row; receipt matching and mercenary target placement |
| `canvas_components/replay_panel/replay_events.ts` | `RisqReplayEvents`: shared canvas markers, selection and detail panel |
| `canvas_components/replay_panel/event_button.ts` | Concrete `DwgRectButton` subclass for marker and panel actions |
| `canvas_components/replay_panel/event_geometry.ts` | Centroid projection and popup positioning clear of canvas edges/HUD rectangles |
| `application/session.ts` | Cache and expose sorted replay events |
| `application/replay/replay_snapshot.ts` | Baseline plus nonterminal execution frames |
| `canvas_components/replay_panel/replay_panel.ts` | `seek(tick: number)` pauses and updates slider/state; end-of-turn events seek to replay end |
| `rendering/board/board_renderer.ts` | Draw markers and the event detail panel as the first two layers inside `drawHud`, before existing HUD components |
| `application/input/pointer.ts` | Route event input after existing HUD controls; retain press ownership through release |
| `risq.ts`, `application/updates.ts` | Compose events, rebuild on snapshot replacement and clear transient state on lifecycle transitions |
| `dialogs/turn_report_dialog/turn_report_dialog.{ts,html,scss}` | Pending: Events section and replay links inside the existing dialog |

## Pending: richer event contracts

The following is a proposed extension, not the current interface. Add only fields required by the remaining events, detailed battle panels or report linking. Existing code uses the smaller `model/replay_event.ts` interface described above.

```ts
type ReplayStop = number;
type EventBase = { turn: number; start: ReplayStop; end: ReplayStop; zones: RisqTickLocation[]; player_ids: number[] };
type ReplayEvent = EventBase & (
  | { kind: 'production'; item_kind: RisqProducibleKind; item_id: number; name: string; producer?: RisqTickTarget; product?: RisqTickTarget; source: 'trained' | 'built' | 'researched' | 'hired' }
  | { kind: 'battle'; effect_ids: number[]; participants: RisqTickTarget[]; damage: number; casualties: RisqTickTarget[] }
  | { kind: 'disruption'; subject?: RisqTickTarget; order_id?: number; reason: string; occurrences: number }
  | { kind: 'eliminated'; player_id: number }
);
function deriveReplayEvents(journal: RisqTurnJournal, report: RisqTurnReport | undefined, game: GameRisq): ReplayEvent[];
```

- Deduplicate zones by space and zone coordinates. Exclude `garrisoned_in` from the zone key; resolve garrison location through its recorded building.
- `RisqSession` already owns events outside reconstructed `game` snapshots.

## Extraction: implemented behavior and remaining work

1. Implemented Production: seed cumulative unit/building/player maps from `journal.baseline`, index executed build/production actions by tick, then walk nonterminal frames. Previously observed unit IDs remain recorded even when visibility changes.
2. Building completion: emit for `under_construction: true -> false`. For a new already-complete building, require a same-tick executed build action targeting that building instance. Include every contributing builder's recorded work zone and the building zone; deduplicate locations. Record completion even if the building dies later in that turn.
3. Implemented unit creation: match a newly recorded unit to an executed producer action by owner, unit type, tick and spawn zone or garrison building. A unit appearing without the matching execution does not generate an event.
4. Production completion uses recorded execution plus that tick's new unit/completed building/technology transition. It does not require backend changes, a completion flag or `order_completed` queue bookkeeping.
5. Implemented research: match a technology becoming true to an executed research action. Automatic grants without research actions are excluded. Names come from current building producibles, with `Technology <id>` as fallback.
6. Mercenary arrivals are not implemented. If added later, use end-turn boundary data; do not confuse settlement hires with unit creation during execution ticks.
7. Implemented Battle: collect executed `attack` and `receive_damage` actions from owned actors, plus visible enemy attacks whose execution target belongs to the local player. Use recorded source/target zones. Space/zone/tick buckets and union-find merge spatially connected records within 2 ticks, including transitive chains and custom seams. Repeated zones are deduplicated, so duplicate attack/receive records do not bias marker centroids. No damage/casualty totals are currently derived.
8. Implemented Disruption: map the matching-turn report's failure array directly to events. Consume one matching private `order_rejected` receipt per row, matching order type, target and exact reason. Use recorded subject locations, or the requested hire zone for mercenaries. Missing location evidence uses board center without a zone highlight. Failures without a supported tick seek to replay end. Identical report rows remain separate; journal-only blocked/cancelled actions add no events.
9. Implemented Elimination: deduplicate report `eliminated_this_turn` rows and recorded elimination transitions. Seek to replay end. Place at the player's latest recorded zones, or the unexplored centroid, then fog centroid, then board centroid. Unknown-area placement has no associated zone highlight.
10. The current combined list is sorted by event tick. Locationless events and richer report ordering remain pending. Derive nothing from redacted fields or replace recorded event locations with current entity positions.

## End-of-turn replay seeking

- Preserve the existing baseline, intent and executed-tick stops. Events without an execution tick seek to `tick_count` using the existing numeric `seek` action. Do not add settlement phases to playback for these events.

## Implemented canvas interfaces and ordering

- `RisqReplayEvents` extends `DwgRectButton`, implements the inherited canvas contract and exposes `reset(): void`. Its private layout reads `session.getReplayEvents()`; its `_draw` renders the markers and panel. There are no public `setEvents`, ID-based selection or separate draw-marker/draw-panel methods.
- Composition: `createReplayEvents()` in `risq.ts` conditionally creates one controller and passes it to `RisqBoardRenderer` and `RisqPointer`, separately from the existing HUD component array.
- Layout: calculate `p = mean(coordinateToCanvas(space) + zoneCenterOffset(zone, hexR))` over distinct zones. Convert with `canvasToScreen(p, transform)`, divide by HUD scale and center a 64 by 64 HUD-unit marker on that point. Canvas clipping handles off-screen drawing.
- Drawing: the renderer draws the event layer first inside `drawHud`, after board content and before existing HUD components. Buttons use `fixed_position: true` and the same HUD-unit bounds for input. No event DOM overlay or layout/containing-block changes exist.
- Overlap: reverse-order marker hit testing chooses the topmost icon. The panel's Next button cycles through all markers whose rectangles overlap that icon. Selection survives pan/zoom/seeking, but a replaced event array resets it.
- Panel: fixed 280 by 174 HUD units, with 18px text and Jump, Next and Close buttons. Candidate positions are tested against canvas bounds and HUD rectangles. Production shows title/player/tick/location; Battle shows `hello battle`. Outside left presses dismiss and are consumed. There is no internal scrolling or expanded battle information yet.
- `RisqEventButton` is the concrete `DwgRectButton` subclass used for icons and text actions. Its release callback runs only while still hovered. The red X asset treatment is pending with Player eliminated.
- Pointer move: first run the existing HUD hit tests. If any HUD component claims hover, clear event hover; otherwise hit-test the detail panel, then reverse-order markers, then board hover. Include event hover in pan suppression and default-cursor selection.
- Pointer press uses first-consumer dispatch: existing HUD, event UI, then board logic. The consuming component owns the press. Event UI consumes all mouse buttons; only left release activates a button.
- Pointer release returns after the stored owner handles it, preventing board click-through and drag selection. Cancellation clears ownership and button states. Wheel input is consumed over the popup or existing HUD; hovering an event icon allows normal map zoom.
- Re-layout before event hit testing using the current transform, not the previous draw's bounds. After seek or snapshot replacement, refresh hover once through the existing `pointer.recalculate()` path.

## Lifecycle status and pending report implementation

- Implemented: `RisqSession.replaceSnapshot` derives sorted Production/Battle/Elimination events from the converted live snapshot and journal. Seeking reconstructs display state without regenerating events.
- Implemented: `DwgRisq.toggleReplay` resets the optional event layer, starts replay at baseline or restores live state on exit. All current journal markers are shown throughout replay, not just at their event tick. The layer reads session events during layout.
- Implemented: `applyStartTurn` exits replay and resets event UI. A resync replaces the event array and resets selection.
- `initialize` and `disconnectedCallback`: reset controller state and pending event presses. Reuse existing initialization cancellation and listener cleanup; add no document-level marker listeners.
- Add `getReplayEvents(): readonly ReplayEvent[]` to the session and `openReplayEvent(event: ReplayEvent): void` to `DwgRisq`. The latter enters replay if needed, selects the event, seeks to its start stop and centers its centroid only when invoked from a report link. A marker's own Jump action does not move the camera.
- Report HTML: add `section-events` with an `event-lines` list directly after `section-scores`. Register that list in the existing dialog class and call `renderEvents()` from `setData`. Scope any new list styles to this section; preserve dialog dimensions and positioning.
- `renderEvents`: use session events only when their turn matches `report.turn`. Each row contains the agreed icon, title, player names and first/last tick or settlement phase. Use native `details`/`summary` for expanding rows within this existing DOM dialog, and `textContent` for dynamic names/reasons.
- Production details show item, producer and completion location; Battle shows participants, observed damage and supported casualties; Disruption shows subject, reason and duration; Elimination names the player. Do not label total attack damage as actual health lost when same-tick healing or overkill prevents that inference.
- A View in replay button closes the existing report dialog and calls `openReplayEvent(event)`. Enable it only when the corresponding journal and stop still exist. Locationless elimination events can seek settlement without moving the camera.
- Preserve existing detailed report sections and totals in this increment. Add matching event links to existing production/combat/failure rows only where instance/order/tick evidence uniquely identifies an event; never match an instance using the combat report's config-type `target_id`.

## Remaining implementation sequence

1. Disruption: implemented as exactly one map event per failure line item in the turn report. No repeat grouping or additional journal-only failures. Remaining filtering/report links are covered below.
2. Player eliminated: implemented; remaining filtering and report links are covered below.
3. Battle details: implemented with colored names, unit/building icon counts and proportional kills, razes and damage bars.
4. View-mode filtering: apply the agreed Production/Battle/Elimination rules to both drawing and interaction, and dismiss any panel hidden by a mode change. Clarify Disruption's mode rule before implementing that part. Preserve the single global event-layer feature gate.
5. Report integration: add the planned Events section/links. Test expired journals, locationless elimination and opening replay from report entries. Preserve existing report totals and dialog positioning.

## Verification status

- One-off fixture checks passed for Production: training, partial work, garrison spawning, one-tick and multi-tick construction, research, automatic grants and fog entry without a matching production execution.
- Battle fixture checks passed for two-tick joins, three-tick splits, same-space grouping, facing border zones, distant fights, transitive chains, bridge merges, local temporal constraints, incoming attacks, enemy-only exclusion, distinct zones and custom seams.
- Canvas/input fixture checks passed for centroid transforms, overlapping events, popup HUD avoidance, placeholder battle text, seeking, wheel behavior, press/release ownership and reset. These are not committed regression-test files.
- `npm run check` and frontend bundle builds passed for the event implementation. Later icon-size, text-size, wheel, feature-flag and intent-jump edits also passed frontend checks.
- Automated visual verification did not complete: headless Firefox crashed. Continue in-browser verification for future event changes, including HUD scaling, pan/rotation, lifecycle transitions and dialog centering.
