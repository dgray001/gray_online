# RISQ frontend refactoring plan

Research date: 2026-09-29. Scope: `frontend/src/components/game/games/risq/` and its direct frontend integration points. The conservative mechanical subset below is now implemented; the broader architectural proposal remains pending.

## Implementation status — conservative pass completed

The implementation was narrowed following the requirement to preserve all existing logic. Only module boundaries, exports/imports, and directory organization were changed. Existing function/class bodies, conditions, mutation order, initializers, state ownership, and event handlers were preserved. A fresh working-tree baseline was captured before editing, including the user's existing uncommitted changes; comparisons were against that baseline, not Git HEAD.

### Completed

| Original module/area | Current destination | Result |
| --- | --- | --- |
| `risq_data.ts` (908 lines) | `model/types.ts` (406), `model/rules.ts` (23), `transport/snapshot_types.ts` (168), `transport/snapshot_conversion.ts` (341) | Separated existing declarations and conversion functions without changing their bodies. The entity registry remains private to conversion and is still created once per snapshot. |
| `risq_zone.ts` (921 lines) | `model/unit_groups.ts`; `rendering/zones/{geometry,slots,draw,hit_testing}.ts` | Separated existing grouping, geometry/cache, slot assignment, drawing, and hit-testing declarations. Largest resulting zone module is `draw.ts` at 404 lines. Drawing and hover timing are unchanged. |
| Tech-tree dialog (583 lines) | `dialogs/tech_tree_dialog/tech_tree_dialog.ts` (455), `tree_nodes.ts` | Moved existing interfaces, constants, and the two top-level helpers into `tree_nodes.ts`. Every dialog method remains unchanged. |
| RISQ root (24 entries) | `model/`, `transport/`, `application/`, `rendering/`, existing `canvas_components/`, and `dialogs/`, alongside the three `risq.*` entry files | Root now has **9 entries**. Existing small modules were moved intact. No compatibility barrels were left behind. |
| Left-panel action buttons (15 files in one folder) | `canvas_components/left_panel/actions/`, with `building/`, `unit/`, and `shared/` | Kept every button implementation; grouped by subject. The base remains `actions/action_button.ts`. |
| Settings integration and custom elements | Updated imports in the global settings store, shared settings dialog, and RISQ consumers | Preserved entry path, HTML/SCSS contents, asset URLs, and custom-element registration order. Webpack configuration was not changed. |

There are now **69 TypeScript files totaling 14,808 physical lines**. No directory exceeds 10 immediate entries. Only `risq.ts` (3,381 lines) and `left_panel.ts` (2,457) remain above 500, and both remain above 1,000. Their bodies were not refactored: the size differences are import/formatting changes, not progress on decomposing their responsibilities. The implementation-start baseline had 3,378 lines in `risq.ts`, slightly newer than the initial research snapshot below.

### Validation completed

- Baseline and final full-frontend TypeScript (`tsc --noEmit`) and ESLint checks passed.
- A TypeScript AST comparison found all **436 top-level non-import declarations** identical to the captured baseline, ignoring comments, formatting, and top-level export modifiers. This includes every function/class body and initializer, not just the extracted helpers.
- Moved HTML and SCSS files are byte-for-byte identical. The two external settings integration files differ only in imports.
- **3,373 baseline/current comparisons** passed for geometry at multiple radii, clipping draw-call traces, grouping/filtering, slot ordering/coarsening, unit anchors, hover results and mutations, and snapshot conversion. Separate assertions verified shared unit/building/resource references and isolation between snapshots.
- Source-module load smoke checks, using minimal DOM/storage stubs, passed for both the RISQ entry and the shared settings dialog. Custom elements register in the same order, with no duplicate registrations. The exercised runtime import graphs went from three detected cycles to zero. These checks do not instantiate the game or exercise browser layout.
- Baseline and final production webpack builds completed with output redirected to `/tmp`, leaving `backend/static/dist/` untouched.
- Final physical-line/directory audit confirms the counts above. Formatting was scoped to touched TypeScript files.

The final build emits webpack's size warnings for `risq.bundle.js`: **243,709 → 270,979 bytes**, while `index.bundle.js` shrank **199,261 → 168,711 bytes**. Total emitted JavaScript across entries decreased **560,605 → 559,449 bytes**. Moving dependencies changed chunk allocation; this is an observed packaging difference, not a claim of unchanged load performance. No bundler tuning was included.

The declaration audit, differential checks, and module-load checks were one-off scripts under `/tmp/risq-refactor-{audit,parity,load}.cjs`, using `/tmp/risq-refactor-baseline/`. They are session verification evidence, **not a committed frontend test suite**. `npm test` remains its existing placeholder. No live browser gameplay, screenshots, or performance measurements were performed, so this pass does not claim exhaustive behavioral equivalence.

### Deferred intentionally

- **Priority 1 remains open:** decomposing `DwgRisq` and `RisqLeftPanel`, including input, selection, commands, action factories, content renderers, and layout. They still own the same state and communicate through the same APIs.
- No selection ownership changes, shared-layout redesign, narrow-capability migration, single/group command consolidation, or gesture unification.
- No changes to spending/capacity prediction, submission, immediate command transport, update recovery, or lifecycle behavior.
- No tech-tree API/type unification with settings, tooltip extraction, request caching, or request-timing changes.
- `model/types.ts` remains one cohesive 406-line file rather than introducing a `model/types/` directory now. It still has type-only dependencies on terrain and turn-report types. Snapshot conversion still imports `resourceType` from the existing resource asset module. These are remaining boundary cleanup opportunities, not completed architectural separation.
- No new permanent test harness. Before another pass touches class internals, establish focused selection/order/event parity tests and perform the browser checks listed below. Continue extracting unchanged bodies before considering architectural changes or deduplication.

The remainder of this document records the original research and broader destination proposal. Its source paths/line anchors refer to the original snapshot unless explicitly updated above; its implementation sequence is **not** a statement that all stages have been completed.

## Original findings and priorities

Counts below reflect the current working tree, including existing uncommitted changes. Line counts include comments and blank lines; directory counts include immediate files **and** subdirectories, not recursive descendants. Generated bundles, dependencies, and backend code are excluded. Source locations refer to this snapshot and will shift during implementation.

There are **61 TypeScript files totaling 14,721 lines**. Five exceed 500 lines; two exceed 1,000. The two largest contain approximately **40% of all RISQ TypeScript**.

| Priority | File relative to RISQ root | Lines | Assessment |
| --- | --- | ---: | --- |
| 1 | `risq.ts` | 3,368 | Composition, lifecycle, network updates, input, selection, commands, prediction, camera, rendering, and dialogs share one class. Split by responsibility. |
| 1 | `canvas_components/left_panel/left_panel.ts` | 2,462 | Selection interpretation, action construction, layout, many content renderers, tooltips, and pointer handling share mutable fields. Split around content and layout contracts. |
| 2 | `risq_zone.ts` | 921 | Geometry, grouping, slot allocation, drawing, and hit testing coexist. Several functions belong below the rendering layer. |
| 2 | `risq_data.ts` | 908 | Domain interfaces/enums, server DTOs, predicates, and snapshot conversion coexist. High dependency fan-in warrants incremental migration. |
| 3 | `tech_tree_dialog/tech_tree_dialog.ts` | 583 | API types, tree layout/status, canvas rendering, and DOM tooltip construction can be separated naturally. |

The next largest files are `right_panel.ts` (374), `risq_space.ts` (364), `order_row_resolve.ts` (341), `risq_minimap.ts` (324), and `order_row.ts` (318). Keep these cohesive modules intact unless an extraction needs a specific boundary. Small modules such as `risq_orders.ts` (242), `risq_hotkeys.ts` (315), and `risq_image_cache.ts` (108) already provide useful seams.

| Directory relative to RISQ root | Immediate entries | Assessment |
| --- | ---: | --- |
| `.` | 24 | Reorganize: 20 files plus 4 directories. |
| `canvas_components/left_panel/action_button/` | 15 | Group by action subject; retain small button implementations. |
| All other existing directories | 1–7 each | No count-driven reorganization needed. |

Apply the requested thresholds as review triggers: aim for files under 500 lines and directories at or below 10 entries; require an explicit rationale for exceptions. Eliminate files over 1,000 and directories over 20. Do not reach these numbers through arbitrary numbered parts, dense formatting, or a single new oversized controller.

## Dependency constraints discovered in the code

1. **The component is an oversized dependency surface.** Within RISQ, 43 files directly import `risq.ts`, including type-only imports; this is not a count of runtime cycles. Panels and render helpers receive `DwgRisq` and call back into it. Extracted modules should accept narrow capabilities or explicit values, not the entire component under a new interface name.
2. **Selection currently lives in panel data.** `risq.ts:1084` onward derives selected IDs and groups from the left panel. Order dispatch accepts `BuildingData`, `UnitData`, and group variants from `left_panel_data.ts`. Input and commands should eventually consume neutral selection types; the panel should render that selection. Introduce an adapter before changing ownership.
3. **Conversion reaches into a rendering-heavy module.** `risq_data.ts:7` imports `organizeZoneUnits` from `risq_zone.ts`. Move pure grouping below both conversion and rendering. `risq_data.ts` has 32 direct importing files within RISQ, so temporary explicit re-exports can keep migration reviewable.
4. **Entity identity is deliberate.** `risq_data.ts:598–645` creates one registry per snapshot. Converters reuse objects by internal ID across player, space, and zone containers. `applyUnitBehaviorSet` and `applyGatherPointSet` (`risq.ts:950–982`) mutate player-owned objects and rely on shared references. Never give each converter its own registry or clone objects independently.
5. **Order prediction already has an owner.** Keep `RisqOrdersModel` as the pending/submitted order authority. `gatherers`/`fitGatherCapacity` (`risq.ts:1977–2006`) and spending calculations (`2119–2194`) should consume its effective orders. Do not create independent panel/cursor copies of order state.
6. **Selection gestures have intentional differences.** Board `mouseup` (`risq.ts:3013–3201`) handles whole-slot Ctrl toggling and double/triple clicks. Panel `handleUnitGridClick` (`left_panel.ts:2247–2273`) handles Shift filtering and Ctrl removal of an individual unit. Share grouping operations, but do not simply replace the board branch with the panel handler suggested in the older `PLANS.md` note.
7. **Rendering and hit testing share mutable geometry.** The left panel explicitly records that hover rectangles remain stale until drawing occurs (`left_panel.ts:128`). Zone drawing also supplies slot/hover information. Splitting these without a shared layout result risks invisible or misplaced click targets.
8. **External entry points must remain connected.** `frontend/webpack.config.js:23` names `risq.ts`; `frontend/src/scripts/settings_store.ts` imports `risq_hotkeys`; the shared settings dialog imports `risq_settings/risq_settings` for registration. HTML/SCSS imports and custom-element registration are side effects that must survive moves. Keep image URL strings unchanged: they refer to backend static assets, not source paths.

## Proposed organization

Keep `risq.ts`, `risq.html`, and `risq.scss` at the current entry path. Group internals into six directories, leaving **nine root entries**. The following is a destination outline, not a demand for one file per listed concept.

```text
risq/
  risq.ts                         # composition, integration, lifecycle
  risq.html
  risq.scss
  model/                          # domain types, predicates, queries, grouping
    types/                        # game/player, world, entities, combat, economy, orders
    queries.ts
    unit_groups.ts
    rules.ts
  transport/                      # wire types, conversion, update/command boundary
    snapshot_types.ts
    snapshot_conversion.ts
    update_types.ts
    game_updates.ts
    game_commands.ts
    tech_tree_api.ts
  application/                    # local interactive state and use cases
    selection/                    # state, queries, gestures, control groups
    orders/                       # existing model, resolution, dispatch, predictions
    input/                        # pointer, keyboard, bindings, armed action state
    session.ts
  rendering/
    board/                        # viewport, board pass, order paths, overlays
    zones/                        # geometry, layout/slots, draw, hit testing
    assets/                       # images/cache, unit/building/resource/tech helpers
    space.ts
    region.ts
    terrain.ts
    cursor.ts
  canvas_components/
    left_panel/
    right_panel/
    bottom_panel/
    minimap/
    order_row/
    risq_tooltip.ts
    message_queue.ts
  dialogs/
    risq_settings/
    tech_tree_dialog/
    turn_report_dialog/
```

Preserve healthy canvas component folders. For mixed existing modules, separate domain code before moving rendering: e.g. the terrain/view enums and `resourceType` classification must be available without loading renderers. Put turn-report types/conversion under `model/types/` and `transport/` respectively, keeping the turn-report dialog together with its template/styles. Move the mercenary drawing helpers to `rendering/board/`; move hotkey definitions to `application/input/` while keeping their dependency graph free of live session/UI imports because the global settings store loads them.

Within `application/orders/`, budget at most ten immediate modules: `orders_model`, `order_presentation`, `target_resolution`, `subject_resolution`, `unit_commands`, `building_commands`, `placement`, `spending`, `gather_capacity`, and `submission`. If actual extracted code needs more, group commands or predictions into a meaningful subdirectory. Local foundation state belongs with placement; `LocalRisqFoundation` must no longer be exported from the web component.

Within `left_panel/`, retain the facade and panel data adapter; group additions into `content/`, `layout/`, `controls/`, and `actions/`. Move close/space-row/target-priority controls into `controls/`. Keep `actions/action_button.ts` as the base and arrange its 14 concrete buttons into:

- `actions/building/`: attack, delete, gather point, ungarrison, create, research.
- `actions/unit/`: build, delete, garrison, stance, toggle.
- `actions/shared/`: order, stop, delete foundation. Stop currently also handles building subjects.

These groups remain comfortably below ten entries. Count new files and folders together when finalizing the layout; colocated tests count too. Avoid permanent compatibility barrels at the root, which would recreate the directory problem.

## Concrete extraction boundaries

### 1. Decompose `DwgRisq`

| Destination | Existing source anchors | Ownership and interface |
| --- | --- | --- |
| Session and update handling | `initialize`, `setNewGameData`, `applyStartTurn`, submitted/unsubmitted updates, behavior/gather-point updates | Own current snapshot/player context and snapshot replacement. Notify composition code when selection, panels, or reports need updating. Preserve update ordering and existing error handling. |
| Selection modules | `selectedUnitIds`, `currentSingleSelection`, `currentUnitTypeGroups`, `selectOrderSubjects`, idle selection, control groups, drag finalization | Own neutral selection state and resolve IDs against the current snapshot. Panel data becomes an adapter. Keep dead IDs and visibility handling consistent after refresh. |
| Input modules | key event callbacks, `handleHotkeyAction`, `handleHotkeyProducible`, mouse handlers, arm/disarm methods | Pointer and keyboard handlers emit explicit actions; shared armed state retains callback/disarm semantics. Do not duplicate state between controllers. |
| Order targeting/dispatch | `resolveActiveOrderType`, target-validity helpers, `unitOrder`, `unitGroupOrder`, `buildingOrder`, gather-point targeting | Resolve target and eligible subjects separately, then construct orders. Reuse common single/group logic only after recording their current differences. |
| Planning/prediction | local foundations, mercenary placement, `updateResourceSpending`, `gatherers`, `fitGatherCapacity` | Own local plans and derive costs/capacity from snapshot plus `RisqOrdersModel`. Return results/messages through small interfaces. |
| Command transport/submission | production/research, deletion, stance/toggle/priority, gather-point messages, submit methods | Preserve immediate messages versus queued turn orders, payloads, confirmations, submission lock, and submit count. |
| Board rendering | `draw`, unit/building orders, paths, target points, gather-point flag, pending mercenaries | Accept a frame context: snapshot, selection, orders, viewport, asset access. Keep draw order and transform behavior. Separate paths/target geometry from board orchestration. |
| Viewport | resize, canvas/coordinate conversion, visible bounds, region label position, navigation | Own transform and coordinate calculations. Existing canvas board remains responsible for its generic behavior. |

Target roughly 200–450 lines per cohesive module and under 500 for the remaining `risq.ts`. These are planning targets, not measured extraction sizes. Keep public delegation methods temporarily to migrate callers in small steps, then remove unused forwarding APIs. Do not use inheritance, mixins, `any`, or access to private fields to imitate a split class.

### 2. Decompose the left panel

- **Action construction:** extract `pushBuildingActionRow`, `pushUnitActionRow`, villager/military actions, and `refreshActionButtons` (`147–476`). Return button/control configurations from current selection and capabilities. Preserve order eligibility, technology checks, and layout coordinates.
- **Selection queries:** move `isOrderable`, unit/building predicates, and group normalization (`608–874`) toward the shared selection layer. Keep panel-specific presentation adaptation local.
- **Content renderers:** split unit/group/garrison rendering (`1006–1315`), resource/building/foundation rendering (`1317–1399`), and world rendering (`1401–1857`). World content should itself split into space and zone/region modules if needed; its current block is already about 457 lines before imports.
- **Stats and tooltips:** extract stat-group preparation, health/stamina/stat drawing, and associated hit rectangles (`1903–2101`). Keep title/image/separator primitives in a small panel-local module rather than duplicating them across renderers.
- **Layout and input:** return a shared panel layout containing content bounds, button positions, stat rectangles, and hover targets. Drawing and pointer handling must consume the same layout revision. Keep the existing stale-hover guard during initial extraction; make layout timing changes a separate, validated step.

The facade should coordinate showing/closing, select the content renderer, route events, and implement `CanvasComponent`, with a target under 500 lines. Content renderers receive explicit data and drawing capabilities rather than the complete mutable panel.

### 3. Split data and zones before they become new hubs

For `risq_data.ts`, separate domain types/enums (currently roughly `12–434`), wire snapshot types (`436–596`), and conversion (`598–908`). Domain types should be grouped by subject rather than placed in another growing catch-all. Preserve enum numeric values, optional fields, wire names, map construction, and per-snapshot identity. Move `canAffordCost`, `meetsTechRequirement`, and `canHaveGatherPoint` into domain rules.

For `risq_zone.ts`, extract:

- Pure unit grouping (`organizeZoneUnits`, `unitsByPlayerFiltered`, `groupUnitsByType`) into `model/unit_groups.ts`.
- Geometry and cached offsets (`105–348`, plus associated constants) into `rendering/zones/geometry.ts`.
- Slot allocation (`bandGroupsByUnitId`, `buildZoneUnitSlots`) into `rendering/zones/slots.ts`.
- Hit testing (`resolveHoveredZones`, `hoveredZoneObject`, `unhoverRisqZone`, `814–921`) into `rendering/zones/hit_testing.ts`.
- Drawing into `rendering/zones/draw.ts`, with terrain composition/forest rendering extracted further only if the resulting size warrants it.

Retain one geometry implementation for drawing, hit tests, movement anchors, and mercenary ghosts. Preserve slot cache keys, slot ordering, active-player prioritization, view-mode filtering, and visibility semantics.

### 4. Split the tech-tree dialog modestly

Extract API response types and a small fetch function to `transport/tech_tree_api.ts`; both the tech-tree dialog and RISQ settings currently declare shapes for the same `risq/tech-tree` endpoint. Settings may use a narrow derived type. Do not introduce caching or change request timing as part of this move.

Move node types/status/layout and tooltip construction to dialog-local files. Keep lifecycle, canvas event wiring, icon-load redraw handling, and display coordination in the dialog. Aim for three or four TS files plus its existing HTML/SCSS, below the directory threshold. Preserve current tooltip positioning and async redraw behavior.

## Dependency direction

Domain types/rules/grouping must not import UI, `DwgRisq`, transport, or rendering. Conversion depends on domain modules. Application logic consumes domain state and small command/query interfaces. Rendering and panels consume domain data plus narrow view/command interfaces. `risq.ts` creates and connects the concrete collaborators.

Use type-only imports where appropriate, but recognize that type-only references to the whole component still couple modules to its API. Avoid a global event bus or generic service container for this refactor. Keep state ownership explicit: session owns the snapshot, selection owns selected subjects, orders model owns pending/submitted orders, placement owns local foundations, input owns armed interaction state, and view/layout modules own geometry.

## Implementation sequence and review gates

1. **Establish baseline.** Record current dirty changes and read this plan against the latest tree. Run non-mutating type/lint checks and capture existing failures. Establish a small frontend test harness for pure logic; `npm test` is currently a placeholder that exits with failure. Capture representative gameplay states for manual parity checks.
2. **Extract foundational modules.** Move grouping, domain types/rules, wire DTOs, and conversion into final destinations. Use short-lived explicit re-exports if needed; no behavior changes. Verify entity identity and conversion before proceeding.
3. **Extract zone geometry/rendering and panel content.** Move leaf renderers and action factories while facades retain existing state ownership. Validate canvas parity and hover/click behavior. Keep pure moves separate from new layout behavior.
4. **Introduce neutral selection.** Adapt existing panel data first, then migrate board/panel/hotkey consumers to a single selection owner. Preserve different gestures rather than unifying their UX. Validate snapshot replacement and control groups.
5. **Extract order planning and command boundary.** Start with spending/capacity helpers; separate target resolution, subject filtering, placement, and dispatch. Consolidate single/group branches only with parity tests. Keep immediate commands distinct from turn-order submission.
6. **Extract remaining input, viewport, frame rendering, and session orchestration.** Migrate remaining `DwgRisq` consumers to narrow interfaces; remove temporary facade methods when no longer needed. Verify registration, listeners, resize, and reconnect/reinitialization behavior.
7. **Finish organization and dialog split.** Move remaining small modules and dialog triplets, update external settings imports, remove compatibility shims, and recount all files/directories. Each previous extraction should already use its final destination to avoid repeated moves.

Keep changes reviewable by responsibility, and make each step compile independently. Do not mix these commits with backlog features, image changes, balance changes, new update-recovery behavior, or a renderer/framework rewrite. `PLANS.md` already tracks broader tasks; this plan does not replace or edit it.

## Verification plan

### Automated checks during implementation

- Run `npm run ts` from `frontend/` independently of webpack: webpack uses `ts-loader` with `transpileOnly: true`, so a successful bundle does not establish type correctness.
- Run `npm run lint-check`; compare pre-existing failures with the baseline. Use scoped checks for changed files where necessary.
- Build with `npm run build` at implementation integration gates, inspecting generated changes separately. It writes to `backend/static/dist/`.
- Add focused behavior tests for conversion identity, selection/group operations, effective orders and cancellation, spending/refunds, gather capacity, target/subject resolution, and geometry/slot agreement. Keep test setup minimal and avoid importing custom elements into pure model tests.
- Inspect the import graph after migration: no model-to-UI dependencies, no new runtime cycles, and no stale path imports or duplicate custom-element registrations.
- Re-run the physical-line and immediate-entry audit, including test files. Review every remaining file over 500 or directory over 10; document any deliberate exception.

Avoid `npm run check` when only checking: it invokes `format` and `lint`, which write files (`prettier --write`, `eslint --fix`).

### Behavior cases that matter

| Area | Required parity checks |
| --- | --- |
| Snapshot/update | Shared unit/building/resource identity; null/absent spaces; start-turn clearing/report; own versus another player's submit/unsubmit; behavior/gather-point updates reflected everywhere. |
| Selection | Single toggle, whole-slot Ctrl toggle, panel Ctrl removal, panel Shift filter, double/triple click scopes, additive drag threshold, garrisoned units, control groups with missing IDs, region selection, fog/visibility. |
| Order dispatch | Single/group commands, economic-unit filtering for implicit attacks, armed versus implicit actions, Ctrl queuing, empty subject groups, building attack/gather point, production/research, deletion, stance/toggles/priority. |
| Prediction | Local/server foundations, cancellation/refunds, renewal cost deduplication, mercenary costs/placement, already-assigned gatherers and capacity limits, submit/unsubmit preservation. |
| Canvas | Pan/zoom/resize, each detail/view mode, zone center/edge slots, overlays/order arrows/paths, minimap, panel overlap/event routing, stale hover rectangles, tooltips, late icon loads. |
| Lifecycle/integration | Normal game load, settings accessible outside a RISQ game, leave/re-enter without duplicate listeners, dialogs, submit lock/count behavior, unchanged asset URLs and bundle entry. |

For selection and command tests, assert user-visible selection or emitted order/message results, rather than private method sequences. For rendering, compare representative screenshots and hit targets; perform a brief pan/zoom and large-selection responsiveness check to catch allocations or recomputation introduced into the frame loop.

## Completion criteria and research limits

Complete when the two oversized classes are decomposed, all five flagged files are addressed, no RISQ TS file exceeds 1,000 lines, no directory exceeds 20 immediate entries, and remaining soft-threshold exceptions are justified. The goal is cohesive ownership and smaller dependency surfaces as well as smaller files. Tests and gameplay checks must support behavior parity, and all temporary migration shims should be removed.

The original research inspected source structure, imports, major method bodies, state/conversion flows, package scripts, webpack configuration, and the existing backlog, and initially changed only this Markdown file. The subsequent authorized implementation and its checks are recorded at the top. Remaining proposed boundaries and size targets are recommendations, not completed work. Pre-existing working-tree changes were preserved.
