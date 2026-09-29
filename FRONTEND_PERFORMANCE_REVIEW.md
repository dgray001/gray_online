# Frontend performance review

Reviewed 2026-09-29 against the current working tree, after the mechanical RISQ refactor. This review changed no application code. Scope: RISQ rendering, selection, orders, image caches, and dialogs; shared canvas, game lifecycle/loading, message handling, chat, and lobby timers. Other games received a lighter lifecycle/timer inspection, not a comprehensive rendering audit.

The strongest findings concern **work repeated unnecessarily**, not source-file size. The first things to address are the hidden game's draw loop, chat's retained message references/timers, and RISQ's repeated selection/order/geometry calculations.

## Evidence and limits

This is a source review supplemented with small probes of the actual TypeScript functions transpiled in Node. There was no live browser, real multiplayer session, GPU measurement, heap snapshot, or network waterfall. “Confirmed” below means the code path/work or probe result is established; it does not mean a production slowdown was measured.

The current backend initializes the population cap to **100 per player** (`backend/game/games/risq/create_game.go:71`). Default recommended board radii are 4, 5, and 6 for up to 3, 4, and 6 players respectively (`internal/mapgen/script.go:36`). For a full hex board these are 61, 91, and 127 spaces. Scenarios with 500/1,000 owned units below are scaling stress tests beyond that current cap. Multi-player visible units and long order queues can still exceed 100 entries.

Priorities are relative investigation/fix order, not claims of measured production severity.

## Findings

### 1. The previous game keeps drawing after exit — highest priority

**Evidence:** `frontend/src/components/game/game.ts:531` removes the `show` class and clears shell state, but does not remove `game_el` or empty `game_container`. `game.scss:7` hides the shell with `display: none`. The page keeps the game shell mounted (`page_home.html`). The canvas board's `initialize` starts a `setInterval(..., 20)` (`util/canvas_board/canvas_board.ts:188`), and its cleanup occurs only in `disconnectedCallback` (`:441`). The callback has no visibility/active-game guard. RISQ's own snapshot remains on the retained component, so its `draw` still has game data.

**Impact:** after leaving RISQ for the lobby, the old board remains scheduled at a nominal 50 draws/second until it is replaced, disconnected, or the page closes. A CSS-hidden canvas can avoid presentation work while its JavaScript calculations and canvas calls still run. This also retains its game state and image caches. Do not interpret this as one new accumulating board per exit: replacement disconnects the previous board; the problem is the last hidden board continuing to run.

**Suggested change:** explicitly suspend/dispose rendering on exit, or remove the old game element when it is no longer needed. Validate exit, reconnect, waiting room, and re-entry. Treat network/socket lifecycle separately so a rendering fix does not accidentally change connection behavior.

**Verify:** record a performance trace in RISQ, exit to the lobby, then observe whether `DwgRisq.draw` continues. Repeat exit/re-entry and inspect retained objects and timer counts.

### 2. Chat retains every message in its animation list and schedules overlapping full-list cleanup — high priority

**Evidence:** `frontend/src/components/chatbox/chatbox.ts:70–104` pushes each message element into `new_chat_elements` but never removes entries. `clear()` (`:129`) clears the DOM, not this array. Each `addChat` schedules a timeout that walks the entire array. Although `last_new_messages_button_timer` is checked and cleared, the new `setTimeout` result is never assigned to it.

**Confirmed probe:** 100 messages added before timeout execution produced 100 timers. Running their callbacks caused **10,000 message-element class removals**. Calling `clear()` left **100 references to the removed message elements**. The probe used actual chat methods with lightweight DOM/timer stubs.

**Impact:** retained detached DOM after clearing; growing repeated cleanup work over long sessions; bursts produce quadratic cleanup work. Separately, displayed chat history itself has no bound, so its live DOM also grows until cleared.

**Suggested change:** track only messages awaiting animation cleanup, remove processed references, manage timer handles correctly, and clear transient state on clear/disconnect. Choose whether animation expiry is per-message or shared before consolidating timers: that affects current highlight duration. Limiting visible history is a separate UX decision, not a prerequisite to fixing the stale animation list.

### 3. Selection membership rebuilds the entire selection repeatedly — high confidence, size-dependent

**Evidence:** `risq/risq.ts:1148` implements `isUnitSelected(id)` as `this.selectedUnitIds().has(id)`. `selectedUnitIds` (`:1087`) creates a new Set and, for groups, flattens/copies selected IDs. Zone rendering invokes it while checking unit slots (`rendering/zones/draw.ts`, selected calculation in `drawRisqZone`); minimap rendering invokes it for every non-garrisoned player unit (`canvas_components/minimap/risq_minimap.ts:214`).

**Impact:** one pass through U displayed units with S selected units performs roughly O(U × S) work rather than O(U + S), plus short-lived allocations. With no selection, it still creates empty Sets. The existing main order pass already computes selection once, demonstrating a useful local pattern.

**Suggested change:** compute a selection Set once for a draw/frame and share it with renderers, or maintain a revisioned selection-derived value. A frame-local value is a smaller first change than relocating selection ownership. Account for selection/snapshot changes in event-driven reads; do not leave a stale cache between frames.

**Verify:** compare no selection, one unit, 100 selected units, and multi-player group views; measure allocations and time in `selectedUnitIds`/Set construction.

### 4. Effective orders are recomputed by scanning every order for every subject — high confidence, workload-dependent

**Evidence:** `application/orders/orders_model.ts:221` copies all orders, filters cancellation orders into a Set, scans every order, and checks `subjects.includes` for each `effectiveForSubject` call. RISQ calls this for each unit's arrows (`risq.ts:1178` onward), idle-unit recomputation (`:446`), and every unit while calculating gatherers (`:1990`). The selected resource/building worker count calls `gatherers` during panel drawing (`left_panel.ts:1336`, `:1975`, `:2069`).

**Impact:** approximately O(U × O × A) subject-matching work in the worst case, where O is order count and A is subject-list length, plus repeated cancellation/all-order allocations. A visible worker-count panel adds another all-unit pass. Long queues and large multi-subject orders matter even under the 100-unit cap. Arrow drawing iterates owned units/buildings without an off-screen guard, although safely culling arrows must consider segments crossing the viewport.

**Suggested change:** build cancellation and subject/kind indices once per order revision, or compute a frame-local effective-order view. Cache gatherer summaries against the same snapshot/order revision. Preserve cancellation, clear-previous, clear-immune, order identity, and ordering semantics. Unit and building IDs can collide: index by both kind and ID. Existing model mutations are not all signaled identically (`setSubmitted`, for example), so do not assume the current change callback is a complete cache-invalidation contract.

**Verify:** exercise pending/submitted/cancelled orders, Ctrl queues, production/research, gather capacity, start-turn, and unsubmit. Compare exact output order sequences as well as timings.

### 5. Slot assignment is rebuilt per ordered unit; forests repeat geometry searches — high confidence

**Slots:** `rendering/zones/slots.ts:5` calls `buildZoneUnitSlots` on every `unitSlotWorldPosition` request. `risq.ts:1263` uses that function for unit arrow anchors. `drawRisqZone` independently builds the same zone's slots during painting. The builder sorts groups and allocates individual-unit representations before falling back to coarser groups. N ordered units in one zone repeat that work N times; anchors for attack targets can add calls. This work also occurs when space-level arrows do not need detailed slot coordinates.

**Suggested change:** share a zone slot layout and ID-to-slot lookup for a frame or an explicitly invalidated snapshot/layout revision. Include active player and slot count in the key; keep resize-dependent geometry separate. Do not blindly reuse `zone.unit_slots`, which currently depends on whether/when the zone was drawn.

**Forests:** `rendering/zones/draw.ts:172` recomputes tree offsets each time `drawForestTrees` runs. An edge forest uses six outer and three inner points. Each calls `edgeSlotAtAngle`, whose `distanceToEdgeZoneBoundary` performs 30 iterations (`geometry.ts:85`). That is **270 boundary-search iterations per visible edge forest per frame**, before canvas painting. These positions depend on center/edge shape and radius, not pointer position or elapsed time. Space and zone detail views both draw forests (`rendering/space.ts:171`, `:196`, `:224`).

**Suggested change:** cache the two local tree-offset layouts by radius, similarly to the existing unit-offset cache. Continue applying map rotation during drawing. This is one of the smallest candidate optimizations with a clear invalidation boundary.

### 6. Static terrain/minimap/region work is repainted at a fixed 50 Hz — confirmed design, impact needs browser profiling

**Evidence:** the board's interval clears and draws everything even while idle. `risq.ts:988` traverses every board cell and recalculates space centers before visible-space culling. The minimap (`risq_minimap.ts:152`) repaints the whole map, allocates `spaces.flat().filter(...)`, redraws region borders, and scans all player units on every frame. Region border drawing recomputes six-neighbor membership/edge geometry per included space (`rendering/region.ts:32`).

**Impact:** sustained CPU/canvas work for largely unchanged turn-based state. Main-map culling does exist; this is not a claim that all off-screen terrain is painted. Larger maps increase traversal/minimap cost independently of visible detail.

**Suggested sequence:** first address hidden/idle scheduling and the derived-data duplication above. Then profile caching minimap terrain/ownership/region layers separately from the viewport outline and selection dots. Cache board-space positions until geometry changes. Consider animation-frame scheduling plus explicit invalidation, retaining timed redraws for tooltips, messages, movement, and late image loads.

**Risk:** merely replacing a 20 ms interval with unrestricted `requestAnimationFrame` can increase work on high-refresh displays. The goal should be fewer unnecessary frames and reusable drawing layers, not a scheduler substitution alone. Dirty rendering must invalidate on image completion and preserve hover geometry.

### 7. Bulk order cancellation repeatedly refreshes the whole UI — confirmed amplification

**Evidence:** `canvas_components/right_panel/orders_list.ts:72` loops over `orders.cancel` for “cancel all.” Each cancellation triggers the model change callback, which calls `refreshPanels` (`risq.ts:437`). That synchronizes foundations, recomputes spending/idle units, rebuilds order-list rows, refreshes the left panel, and recalculates hover. `cancelForSubject` can similarly issue multiple change callbacks.

**Confirmed probe:** cancelling 100 submitted orders generated 100 change notifications. The probe counted callbacks; it did not execute 100 full browser refreshes or time them.

**Suggested change:** batch presentation refreshes around one user operation, while preserving mutation order and final orders. Keep immediate feedback where required. This is a better target than micro-optimizing individual row constructors.

### 8. Every game-update payload is serialized again for logging — confirmed unnecessary serialization

**Evidence:** shared `frontend/src/components/game/message_handler.ts:172` executes `JSON.stringify(update_message)` inside an unconditional console-log template after parsing the payload. RISQ updates can include full snapshots. This string is constructed regardless of whether the console is being viewed. There are additional ungated message logs.

**Impact:** extra main-thread traversal/allocation per snapshot; its actual size/frequency must be measured. Do not claim console object retention without a browser heap trace. The existing bundler configuration does not explicitly strip these logs.

**Suggested change:** gate expensive diagnostics before constructing the string, or log bounded metadata. Benchmark representative update payloads before considering workers or protocol changes; those are much larger interventions.

### 9. Image cache growth and first-use work deserve a memory profile — candidate, not a measured leak

**Evidence:** `rendering/assets/image_cache.ts` keeps canvas and cursor maps without eviction. Player recoloring reads/writes every pixel on a cache miss (`:69`), and cursor creation encodes a data URL (`:104`). Order-row cluster cache keys include unit types/counts and total (`rendering/zones/draw.ts:57`, `canvas_components/order_row/order_row.ts:90`), so changing group compositions can create new entries over a session.

**Impact:** first-use frame spikes are plausible; historical composite canvases can accumulate. A 64×64 RGBA surface is roughly 16 KiB of pixel storage before implementation overhead; 1,000 distinct such surfaces would be about 15.6 MiB, **an illustration, not an observed cache size**. The cache belongs to the RISQ instance, not a global permanent map; the hidden-game retention in finding 1 extends its lifetime.

**Suggested change:** measure entries, dimensions, hit/miss rate, and retained bytes first. If needed, bound dynamic composites separately from the small stable icon set. Avoid evicting frequently reused icons so aggressively that it increases recoloring/encoding work. There is already image caching; recoloring is not repeated for every pixel on every frame.

## Lower-priority observations and existing mitigations

- **Lobby lifecycle:** lobby, room, and users components start periodic ping/UI-update timers without their own disconnect cleanup. `DwgGame.parsedCallback` installs an anonymous document keyup listener without removal. These warrant lifecycle cleanup, but the normal page mounts those shells once; do not claim each ordinary game launch duplicates those listeners/timers. Hidden lobby UI also keeps its ping updates running. Preserve necessary connection heartbeats when reducing hidden DOM work.
- **Polling startup:** game loading waits for script/element readiness through `until` (`game.ts:404`, `:410`; `scripts/util.ts:19`), whose default poll interval is 300 ms. Successful transitions can wait up to a polling interval; failures have no cancellation/timeout on that path. Prefer load/readiness promises with cleanup. No network latency was measured.
- **Pointer/layout work:** canvas mousemove reads its bounding rectangle on every event; the tech-tree tooltip also reads bounds and writes position (`tech_tree_dialog.ts:258`, `:419`). Repeated synchronous layout is a profiling candidate, not a proven forced-layout bottleneck. The dialog already skips content rebuilding for an unchanged hovered node.
- **Order-list painting:** the shared listbox draws only visible rows, though it still walks/translates over the complete list. Avoid reporting missing row culling. Refresh recreates rows, which is more relevant to bulk operations.
- **Existing wins:** main-map visible-space culling; cached image recoloring; a single-entry unit geometry cache; conversion-time grouping; cleanup of canvas timers on actual disconnect; event-driven tech-tree redraws. Preserve these.
- **Packaging:** the preceding refactor's production build emitted about 271 KB for RISQ and 169 KB for index (uncompressed), with a RISQ size warning. Index plus RISQ decreased slightly compared with the old split. Bundle boundaries moved; that warning alone does not establish slower loading. No compression/cache headers, asset waterfall, or real load times were measured in this review. Avoid bundler tuning as the first response to the CPU findings above.

## Probe results

Actual source was transpiled with the installed TypeScript compiler and run under Node v26.9.0. Each timed result is a median of nine runs after four warmups. Module loading/transpilation was outside timed regions. No canvas rasterization, browser DOM layout, or network work was included. These are isolated CPU probes, **not measured frame times**, and should not be added together as a production frame estimate.

| Fixture size N | N selection checks, N selected IDs | N effective-order queries, N single-subject orders | N anchors, all N same-type units in one center zone |
| ---: | ---: | ---: | ---: |
| 100 | 0.317 ms | 0.156 ms | 0.730 ms |
| 500 (stress) | 4.629 ms | 1.191 ms | 14.859 ms |
| 1,000 (stress) | 20.244 ms | 5.001 ms | 69.046 ms |

The selection fixture uses `UNITS_BY_TYPE` with the same selection object for every lookup. Orders are submitted unit-move orders with one subject each, without cancellations or clear-previous flags. Anchor probes repeatedly call `unitSlotWorldPosition` at radius 60. Dense-zone and above-cap fixtures are deliberately adversarial. They demonstrate scaling; the 100-unit results do not establish a major frame-budget problem by themselves.

Additional probes:

- 100 edge forest draws with no-op canvas methods: median **0.726 ms**, excluding real painting.
- 100-chat burst: **100 scheduled timers**, **10,000 message-element cleanup operations**, **100 retained element references after clear**.
- 100 submitted-order cancellations: **100 change callbacks**.

The one-off probe is `/tmp/frontend-performance-probes.cjs`; it changes no source. These are session diagnostics, not a newly installed test harness.

## Fix clarity and difficulty for the six highlighted issues

All six findings have clear corrective approaches. Difficulty reflects the work needed to preserve existing behavior, not just the number of lines changed. These fixes are proposed, not implemented.

| Issue | Proposed fix | Difficulty | Main verification requirement |
| --- | --- | --- | --- |
| Hidden game keeps drawing after exit | Stop/dispose the board when exiting and ensure re-entry initializes it correctly. | **Easy–moderate** | Test exit, reconnect, and re-entry; verify the hidden board stops drawing without changing connection behavior. |
| Chat retains message elements and overlapping timers | Remove processed message references, track timer handles correctly, and clear transient state when chat is cleared or disconnected. | **Easy** | Preserve highlight timing; test message bursts, clearing, and retained references. |
| Selection checks repeatedly construct the selected-unit Set | Build the Set once per rendering pass and reuse it. Keep the first implementation frame-local rather than introducing a persistent cache. | **Easy** | Preserve selection gestures and ensure event-driven reads never use stale selection. |
| Effective-order queries repeatedly scan all orders | Compute effective orders once per revision or frame, indexed by subject ID and subject kind. | **Moderate–hard** | Preserve cancellation, clear-previous behavior, order identity, sequencing, and unit/building ID separation. Test snapshot and submit/unsubmit transitions. |
| Unit slots and forest geometry are recalculated unnecessarily | Cache forest offsets by radius. Separately, reuse slot assignments and unit-to-slot lookups. | **Forests: easy. Slots: moderate.** | Forests must preserve rotation and resize behavior. Slots must remain consistent with drawing, hit testing, active player, slot count, and snapshot changes. |
| Bulk cancellation refreshes the whole UI after every cancellation | Preserve cancellation order, but defer the expensive presentation refresh until the batch finishes. | **Moderate** | Verify intermediate refreshes do not supply state needed by subsequent mutations; compare final orders, spending, foundations, and selection. |

## Recommended next pass

1. Capture browser traces for active idle RISQ and exit-to-lobby to establish a baseline.
2. Start with **forest offsets, frame-local selection reuse, and chat cleanup**: narrow changes with straightforward verification. Preserve chat highlight timing and test bursts and clearing.
3. Fix **hidden rendering**, which may provide the biggest practical benefit. Explicitly test exit/re-entry, reconnect, and disconnect behavior.
4. Profile realistic 100-unit games with short versus long queues, dense zones, and selected gatherable objects. Introduce **slot reuse** without changing slot allocation or hover timing.
5. Address **order indexing and bulk-refresh batching** as separate, carefully tested changes; they interact with mutable order state and derived UI state.
6. Investigate idle redraw/minimap layers, logging, and cache lifetime using the measurements below. Do not combine scheduler, cache, and gameplay-state redesign in one patch.

For each browser scenario, capture median/p95 draw duration, scripting/painting time, allocation rate, GC pauses, long tasks, and cache counts. Use an idle board, 100 selected units, a dense moving army, a gatherable building selected, long production queues with cancel-all, forest-heavy pan/zoom, a tooltip/dialog, repeated snapshot updates, and exit/re-entry. Use the default map sizes first, then larger/multi-player stress cases.

Any optimization must preserve selection gestures, order identity and sequencing, hit-test geometry, fog/view modes, submit/unsubmit behavior, and snapshot entity sharing. Reuse the prior refactor's parity approach, and add browser verification before claiming unchanged gameplay or improved frame rate.
