# Step 4: AI unit test design

Status: design only. This document plans category 4 of [DESIGN.md](DESIGN.md); it does not mark that category complete.

## Scope and test boundary

Create tests in `backend/game/games/risq/tests/risq_ai_tests/unit`, importing `risq/ai`. Exercise `ParseModel`, `Model.ApplyUpdate`, and `Model.DecideOrders` with small JSON configs and hand-built implementations of `ai.View`.

Use behavioral tests: a condition controls an observable action, an expression chooses a distinguishable target, and state changes are exposed by a later rule or decision. Keep private parsers and internals private; do not use reflection or copy their algorithms into tests. The diagnostic injection API proposed below is a runtime dependency, not an export of test-only internals.

Engine receipt, real target encoding, fog construction, AI goroutines, `StopAi`, and `Results` remain category 5. Recorded view calls prove which order constructor the AI selects, not that the real engine accepts its output. No frontend tests are part of this category.

## Fixture and assertion design

- Use JSON unmarshalling into `map[string]any` so numeric input has the same `float64` representation as runtime configs.
- Give fixture entities distinct IDs and locations except in explicit same-location tie cases. Mix matching and nonmatching entities in both positions.
- Supply explicit costs, stockpiles, population, technology flags, visibility summaries, and distance tables. Include a distance that differs from axial distance to prove delegation to `View`.
- Record each order-constructor call with its subject, target, item ID, and clear flag; return a unique sentinel `ai.Order`. Compare the complete `Decision` and the ordered call log, including unexpected calls.
- Implement all `View` methods explicitly; fail on any unconfigured call instead of embedding a nil `View`. Empty values must be intentional fixture inputs.
- Queries return fresh slices. Constructor bookkeeping must explicitly track assigned units, claimed buildings, planned production and techs, foundation changes, cancellation, and assigned attackers when a scenario needs them.
- Keep that bookkeeping limited to the `View` contract in `internal/aibridge/orders.go`; independently test the fixture transitions. Do not implement costs, scoring, rule evaluation, or AI selection in the fake.
- Inspect captured query arguments, including gatherer exclusion IDs, range checks, `SpaceCondition`, and inclusion predicates. Test predicate inclusion and exclusion on named coordinates.
- Assert ordered queues and behavior priority lists exactly. Normalize only collections whose order has no meaning; never sort away a scheduling or clear-flag failure.
- Use a fresh seeded `rand.Rand` per model. For multi-turn cases retain the model but supply a fresh view with deliberate changes; also compare independent models for state isolation.
- Use explicit numeric expectations, with a small stated tolerance only for fractional arithmetic. Preserve the existing `ai/workforce_test.go` coverage.

## Diagnostic isolation

The current API has no diagnostic injection point. Before implementing diagnostic assertions, add `ParseModelWithDiagnostics(raw map[string]any, rng *rand.Rand, diagnostics io.Writer) Model`; preserve `ParseModel(raw, rng)` by delegating with `os.Stderr`. This document proposes that prerequisite; no Go files are changed here.

- Require a non-nil writer; use `io.Discard` in ordinary tests and a distinct `bytes.Buffer` per model in diagnostic tests. Never replace `os.Stderr` or a global logger.
- Pass the writer through the parse context for config errors and unknown variable warnings, then retain it in model internals for expression, condition, and bucket-depth diagnostics. Cover every current AI `os.Stderr` write.
- Assert diagnostic count and identifying expression/variable/rule context, not complete wording. Runtime expression warnings occur once per expression per model; condition errors remain per evaluation; bucket-depth warnings occur once per model.
- Concurrent parses with distinct buffers must not mix diagnostics or variable-name bookkeeping. Different models may run concurrently; one model and its buffer have a single decision owner.
- A failed parse writes only to its supplied writer and returns NoopModel. Following valid parses and independently running models remain unaffected.

## Model and shipped configs

- Enumerate every `config/ai/*.json` file, sorted by filename, including `tmp_*` configs. Fail if discovery returns no files; decode and require `*ai.RulesModel`, not merely a non-nil `Model`.
- Include `default.json`, every versioned config, and every experimental config found at execution time. Discover dynamically without a fixed file count so concurrent additions cannot silently escape coverage.
- Every discovered shipped config, including `v0.7.json`, must pass both acceptance and runtime checks. Each runtime subtest first requires `*ai.RulesModel`, then runs empty and representative populated views for two decisions; require no panic and structurally valid constructor calls without freezing strategic outputs.
- A shipped config returning NoopModel fails acceptance and runtime prerequisites; it is never skipped or counted as a successful smoke test. `v0.7.json` currently blocks this gate until its obsolete options are migrated. Keep a separate minimal `distribute` rejection test; do not edit configs in test setup to manufacture success.
- Valid empty `rules` produces a rules model and an empty decision. Explicit `NoopModel` stays empty across updates and decisions.
- Missing or wrongly typed `rules`, invalid rule/condition/action nesting, and an invalid action after a valid action must yield `NoopModel`; no partially parsed rule may execute.
- `max_bucket_depth`: missing uses 3; accept positive whole numbers; reject zero, negative, fractional, null, and nonnumeric values. Prove the configured limit with nested bucket execution.
- `ApplyUpdate` currently changes neither model's decision state. Verify that it does not execute rules or clear persistent variables and buckets; `DecideOrders` owns the refresh boundary.

## Parser acceptance and rejection

Maintain an explicit case table for every action name in `parseActionInner`, every counter name in `counterParsers`, and every target set in `targetSetNames`. Review these tables against the source switches when extending the DSL.

- Conditions: exactly one key; `always`, nested `all`, `any`, and `not`; valid comparison suffixes; required numeric IDs, categories, bucket names, and amounts. Reject unknown names, wrong container types, invalid child entries, and malformed expression strings. `always` has the explicit payload exception below.
- Actions: missing/unknown `action`, missing required IDs or task fields, malformed `eligible` and ID lists, invalid type/category/stance/priority names, and unsupported unit or building filters.
- Cover every `numericActionKeys` field with a literal, expression, supported space-query object, omitted default, and malformed value. Test defaults through resulting behavior, not inaccessible struct fields.
- Validate `gather` category/weight exclusion; `add_q` resource/id and producible/cost exclusion; build targets/site/reuse options; move's required targets and forbidden attack order option.
- Validate `unit_when`, nested bucket tasks, drain sources, stop order names, exploration anchors, space anchors, target filters, attack order modes, and `spread` fields and enums.
- `SpaceCondition`: reject unknown fields, wrong JSON types, vision outside 0..4, invalid owners, and invalid nested children. Check zero-valued pointer fields survive parsing.
- Optional-field contract: absence selects the documented default; a present null or malformed value rejects the entire model atomically. No coercion, truncation, or silently ignored recognized option is allowed. Empty arrays have the field-specific meanings below.
- Current optional parsers violate parts of this contract. Write rejection regressions against the specified expectations; keep them failing until the implementation is corrected, without marking permissive behavior as acceptance or skipping the cases.
- Unknown `var(name)` is currently a warning and resolves to zero; it does not invalidate a model. Distinguish this from an unknown bare expression variable, which must fail parsing.
- Capture diagnostic assertions through the injected writer described above; no process-wide stderr interception or serialization workaround is permitted.

| Present optional field | Required acceptance and rejection contract |
| --- | --- |
| Boolean options | Accept only JSON booleans, including false; reject null, numbers, strings, arrays, and objects. Applies to persist, prioritize, resume, together, when_full, allow_reserved, exclude_buckets, reuse_unbuilt_foundation, exclude_reference, and behavior toggles. |
| Enum and bucket strings | Require strings with a recognized enum value or a nonempty bucket name, as appropriate; reject wrong types, empty values, and unknown enums. Covers targets, target, stance, order, anchor, direction, group_by, overflow, in_bucket, held_by_bucket, bucket, and drain destinations. |
| Config ID values and ID filters | Each configuration ID is a whole number in 1..4294967295; reject zero, negatives, fractions, overflow, wrong types, and null. Empty filter arrays mean no restriction; duplicate IDs collapse as a set. Internal entity IDs supplied by View are uint64 and are not restricted to uint32. |
| unit_types, target_unit_types, eligible, orders | Require arrays of recognized strings; reject nonstring entries and unknown names. Empty type filters mean no restriction; empty eligible uses the idle default; empty stop orders means all kinds. |
| target_priority | Require an array of recognized strings; preserve order and allow an empty array to explicitly clear priority. Omission preserves priority; reject null and nonarrays. |
| within | Require a finite nonnegative whole number, including zero; reject fractions, negatives, null, strings, and expressions. A from anchor still requires within in near filters. |
| numeric action amounts | Accept finite numbers, valid expression strings, or supported query objects; reject null, booleans, arrays, malformed syntax, and unsupported objects. Preserve field-specific signed scores and rounding semantics. |
| unit_when, site, spread, where | Require the documented object shape and validate nested fields; reject null and other types. Empty where matches all; an empty condition object is invalid; empty site/spread uses documented defaults. |
| select_space.options | Require a nonempty array of valid picker objects, each with targets; reject malformed/empty entries instead of ignoring them. Reject simultaneous options and top-level targets. |
| set_var.name and select_space output names | Require `[A-Za-z_][A-Za-z0-9_]*`, including the var prefix; reject empty, malformed, null, and nonstring names. Omitted select_space output names remain disabled. |
| add_q resource cost | Require an object containing only food/wood/stone/gold with finite nonnegative numeric values. Missing components mean zero; zero and an empty cost object are valid; reject negative/wrongly typed/null components and unknown names. |
| drain_bucket.from | Accept a nonempty string or an array of nonempty strings; an empty array means no sources. Reject null, mixed-type arrays, and empty names. |

## Conditions and counters

- `always` evaluates true with empty or populated views and adds no condition-specific view queries. Require its action to run exactly once in rule order; `not(always)` must suppress the action.
- Preserve its existing payload-ignored contract: `{"always": {}}`, null, false, numbers, strings, and arrays all mean true. This is the explicit exception to optional-field typing, not a boolean condition value. Missing the key or combining it with another condition key remains invalid.
- Exercise `always` inside all/any/not, including short-circuit cases that would otherwise query an unconfigured view method. Compare query logs after accounting for model lifecycle and action queries.
- Truth tables for `all`, `any`, and `not`, including empty all=true and empty any=false. Use a call-counting view to prove short-circuit evaluation skips later conditions.
- For every numeric counter, exercise `_at_least`, `_at_most`, and `_equals` below, at, and above a threshold. Include an expression threshold and evaluation failure, which must make the condition false.
- `tech_researched`, `resource_available`, and `bucket_full`: true/false, absent bucket, zero desired size, partial/full/overfull membership, missing home, and missing resource.

| Counter family | Required discriminating cases |
| --- | --- |
| turn, map_size, num_players, enemies_found, land, score_lead | Distinct hand-set values; negative and zero score lead. |
| resource, population_headroom | All four resources; earlier action spending and population reservations affect later conditions. |
| population, idle_units, enemy_units_visible | IDs and types independently; combined filters use the current union semantics; military expands to infantry/archer/cavalry; bucket and distance filters where supported. |
| enemy_units_unidentified | Nil count, zero count, own nongarrisoned subtraction, garrison exclusion, clamping to zero, and spatial filters. |
| enemy_buildings_known, resource_remaining | Visible versus supplied last-known buildings; per-category sums and distance exclusion. Real fog filtering is category 5. |
| building_count | All/filtered IDs; complete, under_construction, damaged, depleted; completed healthy and nongatherable controls. |
| foundation_count, foundation_space_x/y | Planned/materialized fixtures, with/without builders, ID filtering, lowest ZoneRef selection under permutation, negative coordinates, and no-match zero. |
| bucket_size, economic_producers, available_gatherers | Missing/existing buckets; each producer counted once; economic/garrison/current-order exclusions. Retain the existing gatherer regression. |
| count_spaces, closest_spaces, base_spaces, value | Parsed query arguments and predicate effects; connected home components, no-center fallback, empty board, and literal/expression/query amounts. |

## Variables and expressions

- Hand-calculated precedence, parentheses, unary signs, fractions, nested `min`/`max`, and `var(name)`; malformed tokens, missing parentheses, invalid arity, unknown functions, and trailing input.
- Cover every bare name in `expressionVarNames`; resource values include pending spending and population includes pending hires/production. Integer action limits round halves away from zero; coordinate amounts convert by the same current rounding rule.
- Test query amounts for both `count_spaces` and `closest_spaces`, including wrong shape, additional keys, and non-object query payloads.
- Variable lookup precedence: turn variable, persistent variable, built-in counter, then unknown zero. A same-name turn value shadows persistence only until the next decision.
- Earlier `set_var` actions affect later actions and rule conditions immediately. Turn variables clear, persistent variables survive, and separate models never share them.
- Built-in names: unit/type filters, building states, foundation coordinates, categories, bucket names with underscores, `within_N_of_<anchor>`, technology and availability flags, population maximum, and each resource of unit/building/tech cost.
- Built-in cache stores the resolver, not its value: change the view between decisions and prove the same variable returns the new value.
- Runtime division by zero: an action amount becomes zero, while a condition amount fails closed. Parsing may accept a denominator that is zero only for syntax-check stand-ins; verify valid runtime evaluation too.
- A runtime expression warning occurs once per expression per model; unknown-name bookkeeping resets between model parses. A malformed parse must not contaminate the next parse.
- Parse independent configs concurrently with distinct declared/undeclared variables and separate diagnostic buffers; race checks must preserve model behavior, diagnostic ownership, and the mutex-protected name bookkeeping.
- Probe all contextual names in `targetDistanceCounter`: distances, health, stamina, item/unit IDs, assignments, round, vision, ownership/building counts, enemy distance, and region progress. Verify documented fallbacks where context or home is absent.

## Rule execution and decision lifecycle

- Preserve rule and action order; false rules produce no constructor or behavior calls. Earlier reservations/refunds and variable/bucket changes affect later rules.
- Each decision resets queue entries, pending resources/population, behavior outputs, and turn variables; buckets, persistent vars, army membership, and remembered start remain where intended.
- Use two successive populated decisions to detect stale output and state leakage, then an empty view to prune removed bucket/army members. Compare every decision channel, not only `Orders`.

## Targets, space queries, and scoring

- Cover all target sets: enemy_units, enemy_buildings, known_resources, own_buildings, unexplored, home, retreat, unidentified_units, closest_spaces, and coordinate. Test empty and filtered-out candidates for each.
- For `best`, use negative, zero, positive, and tied scores. `min_score` is inclusive; equal target scores choose the lowest ZoneRef. Build-site ties first choose shorter location distance, then ZoneRef.
- Reorder candidates to prove the deterministic tie contracts below, including equal scores and distinct IDs at the same location. Assert selected identity as well as location.
- Separate subject filters from target filters. Check unit IDs/types, building IDs, resource categories, visible enemy units, supplied known enemy buildings, and unidentified-count candidates.
- Default attack prefers the requested unit kind with its defined fallback; picker attack supports unit, zone, and space modes. Check the exact selected constructor and target object.
- Missing home, bucket, unit, or target anchors produce no applicable target. Explicit coordinate anchors evaluate x/y expressions, including negatives. Remembered start enables retreat after buildings are lost.
- Near filters use inclusive space distance; group centers use `View.SpaceDistance`. Test self/home/center exploration anchors, minimum-distance fallback, already-claimed destinations, and repeatability with a fixed seed and candidate order.
- `together`: nearby units share the group-selected target; units more than two spaces away move to the center. Distinguish distance 2 from 3 and exercise no candidates.
- Space queries: preserve nested `where`; reference exclusion; held-by-bucket restriction; toward/away half-plane boundaries; missing direction anchors; explicit versus per-unit `from`.
- `face` chooses an adjacent edge zone only when the actual coordinate direction qualifies; graph-linked but geometrically distant spaces stay at center.
- `select_space` compares options, writes x/y/score or prefixed variables, honors action and option minimum scores, and respects persistence. No target leaves prior values unchanged.
- Follow picker, site-picker, spread, and `unit_when` evaluation with a second context-dependent action to detect context leaking into later work. Verify nested context restoration where observable.
- Test home selection with center/other buildings/unit fallback and base-component selection with and without centers. Reversed candidates must preserve the tie contracts below; retain input order only for explicitly sequential or seeded-random operations.

## Deterministic tie contracts

These are required expectations for the planned suite, including gaps in current code. A mismatch is a regression to fix; no identity contract is left for the test writer to choose.

Define the candidate key as `(Space.X, Space.Y, Zone.X, Zone.Y, kind, InternalID)` ascending, with kind order unit, building, location-only. Location-only candidates have no identity beyond their ZoneRef. Compare uint64 IDs directly, never after conversion to float64; include IDs above 2^53.

| Selection path | Required tie resolution |
| --- | --- |
| Picker best, open spread slot, select_space options | Score descending, then candidate key ascending. Same-location entities of the same kind choose the lower internal ID. |
| Crowded spread overflow | Assigned load ascending, score descending, then candidate key ascending. |
| Spread group_by space | Choose the lowest candidate key as the representative of each space before computing slot capacity/score. |
| Nearest unit/building; garrison building | Location distance ascending, then internal ID ascending. Preserve this identity-first distance tie rule. |
| Nearest zone; build site | Distance ascending then ZoneRef ascending; scored build sites compare score first, then distance and ZoneRef. |
| Group center; home | Group centers minimize summed space distance, then ZoneRef and unit internal ID. Home prefers village centers, then other buildings, choosing the lowest building internal ID in either tier; unit-only home retains owned-space/count/Cantor-key ranking. |
| Enemy space; retreat base | Enemy spaces compare distance then axial x/y. Retreat bases compare distance from unit, distance from home, then Cantor key, preserving existing fallback behavior. |
| Queue; spread unit order; army target loads | Equal queue weights retain insertion order. Equal spread unit_order uses internal ID. Army targets compare load, location distance, then target internal ID. |

Seeded exploration intentionally draws among nearest supplied candidates. Test a fixed seed and candidate order; do not impose identity sorting on that random path or reorder sequential subjects to hide scheduling effects.

## Queue selection and economy accounting

- `add_q` supports unit, building, technology, and resource entries. Item costs come from the view; resource entries use explicit cost. Entries themselves produce no orders.
- Selection admits only the requested kinds and weights strictly greater than the threshold. Prioritized selection sorts descending with stable equal-weight order; otherwise insertion order wins.
- Depth limits affordable entries, not examined entries; test an unaffordable first entry and an affordable later one. Nonpositive depth currently means unlimited.
- `createNextInQ`, `researchNextInQ`, `buildNextInQ`, and `produce` try later affordable candidates when earlier ones cannot execute; stop at the first successful candidate. Distinguish all three item kinds.
- Resource entries contribute gather demand but do not become production orders. Cover each deficit, pending spend, satisfied demand's equal-weight fallback, and strict weight threshold.
- Balanced gather assigns idle workers first, considers existing gatherers, skips unavailable categories, and reassigns only beyond `2 + move_penalty`. Test equality and both sides with hand-calculated worker counts.
- Gather demand sums unscaled costs, then subtracts available stock; weight selects entries without multiplying costs. Include gold-only demand and no workers/resources.
- Affordability tests each resource independently, exact equality, and insufficiency after an earlier action. Production and hire share population reservations; a final slot admits exactly one of two competing requests.
- New foundation construction charges once for all assigned builders; reuse charges zero. Renewal charges once when not already renewing; foundation cancellation credits the later decision budget.
- Queue occupancy includes active create/research orders and decision-local `PlannedProduction`, but not unrelated building orders. Test queue default, limit 1, and larger limits across multiple producers.
- Use constructor bookkeeping to expose duplicate production/research, repeated foundation refunds, and repeated renew/build actions. Do not let a stateless fake hide oversubscription.

## Action-to-constructor matrix

For each family below include a successful case, empty candidates, relevant eligibility/filter controls, and boundary limits. Assert exact targets, subjects, clear flags, ordered follow-ups, behavior payloads, and the absence of unrelated output.

| Action names | Required behavior |
| --- | --- |
| gather | Chosen category/resource and gatherer exclusion ID; explicit eligible tasks versus idle default; balanced mode covered above. |
| create, research | Correct completed producer/producible, item ID, stockpile and queue checks; unavailable item/tech and no producer yield no order. |
| build | Builder capability and availability; nearest or scored legal site; site predicates; reuse toggle; builder max and one shared target. Current build max 0 still selects one builder. |
| build_foundations | Only matching foundations without builders; distinct nearest available builders per foundation, without double assignment. |
| cancel_foundations | Only planned, builderless matching foundations; cancellation target and refund accounting; materialized/occupied controls. |
| renew, repair | Completed depleted gatherable or completed damaged building; skip already assigned targets; nearest worker, one worker per target, renewal cost handling. |
| hire | Tech 4 gate, home, price `(food + wood + stone + 1.5 * gold) * 1.3`, gold reserve, max, and population; check exact affordability boundary. |
| explore, move | Correct move target and clear flag; exploration claims/anchors/random ties; picker movement and grouping covered above. |
| attack, attack_space, attack_zone | Unit/building preference and fallback, explicit targeting mode, max and eligibility, nearest target and no-target controls. |
| garrison, ungarrison | Completed building with room, nearest candidate, same-decision capacity consumption; skip already-garrisoned units; ungarrison only occupied units. |
| delete_unit, delete_building | Eligibility and filters; exact delete subjects and unit clear flag; no unrelated cancellations. |
| building_attack | Completed attack-capable building with no active orders; range boundary, unit preference, building fallback, filters and max. |
| set_unit_behavior, set_building_behavior | Only differing subjects; unit default military filter and building CanAttack gate; explicit false differs from omission, empty priority differs from nil, unchanged settings emit nothing. |
| unit_stop, building_stop | Scoped subjects, order-kind/item filters, all matching queued orders, shared unit order cancellation once, max counts affected entities rather than orders. |
| createNextInQ, researchNextInQ, buildNextInQ, produce, add_q | Queue tests above, including no successful candidate and resource-only queue. |
| set_var, select_space | Variable tests above; no direct order output; later rule/action proves the write. |
| set_bucket, fill_bucket, run_bucket, drain_bucket, empty_bucket | Bucket lifecycle below, including scoped task output and wrapper combinations. |
| army | Stateful defense/assault cases below. |

- Attack `resume`: attack first clears, then gather/repair/renew/build resumes with clear=false and the original target. Cover planned and materialized construction, lost construction target, unsupported/no prior job, and already-fighting subjects whose queue must survive.
- Verify omitted `max` versus positive limits separately per action; defaults are not uniform. A configured but unusable first candidate must not conceal a usable later one unless the DSL explicitly limits candidate selection first.

## Buckets and wrappers

- Set desired size and task; shrink deterministically by retaining lower internal IDs; zero empties membership. Fill only the deficit, excluding units already in any bucket.
- Gather buckets prefer workers on the matching category, then idle, then other eligible jobs. Include each rank and a nonmatching filter control.
- `run_bucket`: missing bucket/task, empty membership, partial versus full with `when_full`, nested tasks, and membership pruning after units disappear from the next view.
- Direct self-recursion and a two-bucket cycle terminate at the configured limit, emit no unintended work, and restore depth for a later independent bucket task. Exercise exactly at and beyond the limit.
- `drain_bucket`: named/list/any sources, destination deficit, overflow-only versus `allow_reserved`, max, deterministic source/ID order, absent source, and source=destination. Each transferred unit remains in exactly one bucket.
- Emptying releases units but preserves configured task/desired size; later fill and run reuse the bucket. Buckets persist through decisions and do not leak between models.
- Compose `in_bucket`, `exclude_buckets`, and `unit_when`; distinguish `Units` from `ScopedUnits`. Membership wrappers narrow unit selection while population/enemy queries still see the broader view.
- Test `in_bucket: "none"` wherever supported: counters interpret it as unbucketed membership, while action wrappers currently look up a literal bucket. Record this distinction for contract review.
- Per-unit conditions set the unit anchor only during evaluation and restore it afterward; test a busy matching unit and idle nonmatching unit, plus behavior/stop actions using scoped units.
- Building behavior has no unit-scoped selection contract; unit wrappers must not be assumed to filter buildings.

## Spread assignment

- Parse all options: capacity, unit_score, unit_order, weight, group_by, queue, overflow. Reject unknown options/types, unsupported grouping, and invalid overflow names.
- Compare omitted capacity with zero, one, fractional, and expression capacity. Existing attackers count toward capacity; exclude the units being reassigned from that existing load.
- Unit selection descends by `unit_order`, then internal ID; target scoring combines picker and unit score. Verify unit_id is the config ID, target_id is the internal entity ID, and contextual health/stamina/distance values are correct.
- Highest score wins among open slots; score ties use the complete candidate key from the tie table. `min_score` rejects below-threshold candidates even when overflow could otherwise admit them.
- At capacity, `overflow: none` stops assigning; round_robin permits first-round overflow using least load, then score. Cover all slots full, no slots, and one unusable slot alongside a usable slot.
- Assignment weight changes later load/capacity decisions; zero and negative weights clamp to zero. Queued assignments also contribute load.
- `group_by: space` merges candidates in the same space; verify the chosen representative and that this mode does not reuse per-entity attacker counts.
- Queue 0 means one round; queue N adds N follow-up rounds. A unit never repeats a taken slot; its first actual assignment clears, subsequent assignments append with clear=false.
- Grouped execution regroups far units and excludes them from assignment rounds. Probe `assign_round`, `target_assigned`, and all scoring-context restoration after success and no-match exits.
- Permute tied candidate lists and assert the tie-table contracts for open slots, crowded overflow, and same-space representatives. Distinct entities sharing a zone must be compared by selected ID as well as position.

## Army state across decisions

- No military units or no home produces no orders; economic, garrisoned, and bucketed units are excluded from controllable military subjects.
- Launch uses the eligible assault IDs gathered within two spaces of home. Test launch-1/launch/launch+1 and gather distance 2/3.
- After launch, membership persists; removed units are pruned and retreat triggers at `members <= retreat`. Check membership changes through later emitted orders, not private state.
- Defenders select threats at the home-radius boundary; distant idle defenders return home only when there are no threats. Mix economic/military enemies and busy/idle defenders.
- Assault target sequence: same-space military combat, nearest reachable enemy unit, visible building fallback, then unexplored movement. Military enemies on enemy building zones are excluded from direct assault targeting.
- Verify balanced target loads, nearest-distance selection, and internal-ID ties using distinct enemies. Hold unit order fixed unless testing a defined permutation contract.
- Carry the same model across gathering, launch, casualties, retreat, and renewed launch. Use a second model to prove independent state.
- `army.strike` is an explicit temporary characterization: compare fresh models with strike omitted, 0, 1, and a larger positive threshold over identical multi-decision views, including enemy disappearance. Require identical complete decisions and constructor logs; separately reject malformed strike amounts.
- Implement this characterization now rather than waiting for strike behavior. It records an unimplemented option, not a validated strategy feature. When strike is implemented, replace the equality test with its specified trigger/boundary tests; step 4 completion does not require that new strategy feature.

## Known concerns to turn into regressions

These are observations from source inspection, not failures demonstrated by newly executed tests. Recheck them when implementing; do not silently bless, skip, or fix them during this documentation task.

1. `v0.7.json` contains four `distribute` options; `parseTargetPicker` explicitly rejects that removed option. The all-config acceptance test should expose the fallback to NoopModel. Keep acceptance required until config/code compatibility is resolved.
2. Optional parsers have the permissiveness described above. The optional-field table specifies required rejection behavior; current violations must fail those regressions until corrected.
3. `army.strike` is parsed but unused. The army section specifies its temporary characterization and replacement rule; this unimplemented strategy option is not a blocker for the unit suite.
4. Home/group-center choices and some nearest/crowded ties depend on supplied order. The deterministic tie table specifies expected identity selection; seeded exploration remains exempt from permutation requirements.
5. Behavior unit filters can bypass the default military fallback. Include an explicit economic filter and compare the intended AI behavior contract with the server's economic-unit exclusion.
6. Repeated refunds, renewals, production, and garrison assignments rely on decision-local view bookkeeping. Verify interactions with a stateful fake and reserve real bridge parity for category 5.

## Implementation sequence and proposed files

1. Implement the diagnostic writer prerequisite before diagnostic assertions; add `diagnostics_test.go`. Then add `view_test.go`, `view_orders_test.go`, `fixture_test.go`, `assertions_test.go` for typed views, checked bookkeeping, JSON models, and complete decision/call assertions.
2. `model_test.go`, `configs_test.go`, `parse_test.go`, `parse_options_test.go`: fallback/atomic parsing, every shipped config including v0.7, always and all other DSL names, defaults, and the required optional-field rejection table.
3. `conditions_test.go`, `counters_test.go`, `vars_test.go`, `expressions_test.go`, `lifecycle_test.go`: boundary tables and state across decisions.
4. `targets_test.go`, `spaces_test.go`, `spread_test.go`: scoring, queries, tie contracts, context restoration and assignment sequences.
5. `queue_test.go`, `gather_test.go`, `production_test.go`, `build_test.go`, `maintenance_test.go`, `hire_test.go`: exact accounting and interaction scenarios.
6. `combat_test.go`, `resume_test.go`, `garrison_test.go`, `behavior_test.go`, `stop_delete_test.go`, `buckets_test.go`, `army_test.go`, `army_strike_characterization_test.go`: remaining action families, persistent lifecycle, and the explicitly temporary ignored-strike contract.

Split tables further only when readability requires it. Use helpers that express fixture setup and assertions; avoid a generic DSL-testing framework.

## Completion and verification

- Every parser/action/counter/target family has a positive case and relevant rejection, no-op, boundary, and mixed-candidate cases. A no-op assertion checks the entire decision and call log.
- Interactions have multi-action or multi-decision fixtures that fail if reservations, refunds, pruning, scope restoration, or queued clear flags are removed. Hand-computed expectations must be independent of production selection helpers.
- Review source switches against the explicit test inventory. All shipped-config, optional-field, identity-tie, and diagnostic-isolation assertions must pass; unresolved violations block completion. Do not weaken assertions, skip configs, or silently accept NoopModel. The explicitly characterized unused strike option is the stated exception.
- Future implementation checks, from `backend/`: `gofmt -w .`, `go test ./...`, `go vet ./...`, and `go test -race -shuffle=on -count=10 ./game/games/risq/tests/risq_ai_tests/unit ./game/games/risq/ai`.
- Parallel execution is optional with separate models, views, RNGs, and diagnostic buffers. Include concurrent parsing to check name bookkeeping; never intercept stderr. No global game config reload, real engine startup, sleeps, or goroutine lifecycle assertions are needed here.
- For this design-only change, review Markdown structure, relative links, source references, and coverage inventory. Do not run whole-tree formatting or modify existing files.
