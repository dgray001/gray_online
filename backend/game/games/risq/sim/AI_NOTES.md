# v0.5 (shipped) and what's next

## v0.5 result
Starting resources are now food 200, wood 200, stone 100, gold 0. On that setup v0.5 (`config/ai/v0.5.json`) crushes v0.4 in
95/96 games (both seat orders, fresh seeds, 400 turn cap, avg ~187 turns). That result used the old crush rule
(loser has no infantry left while the winner has 20+). The sim now calls it a crush when the loser has no units left
at all or the winner has at least 20x as many units; the sim stops there. `SIM_NO_CRUSH_STOP=1` keeps playing: in a 12 game sample 9 reached full elimination
by turn 450 and the rest were down to a few hidden villagers/houses.

What v0.5 does differently from v0.4:
- Eco capped at 44 villagers. Gold comes from owned land, and heavies (120 food + 30 gold) are the gold sink.
  A stone scout (the first infantry explores until a stone mine is known) feeds an early Redoubt (250 stone);
  the blacksmith and techs wait for it. Stone is the binding constraint (~600 per start).
- Heavy infantry are the core army. Up to 44 blunts early for defence, cut to ~24 (blunts are deleted when pop-capped)
  once a Redoubt exists. Mercenary Contracts is researched (`research` now takes `queue: 2` because the Redoubt is
  never idle) and spare gold hires blunts (`hire` action).
- New `army` action (ai/army.go) replaces v0.4's "everyone attacks the nearest thing":
  - defenders converge on threats near home with targets spread out (<=2 attackers per target, no overkill);
  - an assault group launches once 36 units are gathered at home, or right after a big enemy army near home breaks;
    it regroups around an anchor unit, kills enemy units in its own space first, then production buildings
    (VC/barracks/redoubt), remembers seen enemy buildings and explores when nothing is known;
  - enemy units standing on a building's zone can't be reached in melee, so they are skipped until the building falls.
- Outposts (25 wood) go only in unowned bordering spaces (`aibridge.nearestClaimSite`) once the first Redoubt stands:
  more land means more gold.
- Food buckets switch to wood when food piles up (wood limits Redoubts and outposts).

Tried and rejected (measured against v0.4): an early rush (v0.4's home defence, with ~60 armed villagers, wins),
skipping techs/barracks for a second Redoubt, early/unlimited outposts, deleting and re-buying blunts, attacking with
15-30 heavies before v0.4's army broke, and per-unit `attack any` orders (overkill plus piecemeal arrival).

## Sim tooling
`sim/inputs/*.json` is gitignored. A scenario looks like
`{"seed":1000,"iterations":24,"map":"","max_turns":400,"players":[{"ai_config":"v0.5"},{"ai_config":"v0.4"}]}`;
run both seat orders. Results carry a 5-turn `Timeline` (units, buildings, resources, army positions) and `Crush`.
Debug logs (`-debug`) now tag each combat line with the turn and unit/building ids.

## Ideas for v0.6
- Instead of deleting surplus blunts when pop-capped (to make room for heavies), send them on an attack and let them die.
- Villagers still never garrison or flee.
- Finish faster: a hunter squad for stragglers once the main army has won.
- v0.4's remaining wins are games where its own heavies sit on its Redoubt zone; raze the Redoubt earlier.
- Land and banked gold decide close score games; the ai still almost never razes outside an assault.

## Script variables and expressions
- Any numeric action input (`max`, `queue`, `weight`, `depth`, `size`, `reserve`, `move_penalty`, army `launch`/`retreat`/
  `defend_radius`/`strike`) takes a number or an expression string, like condition `amount`s already did.
- Expressions support `+ - * /`, parentheses, `min(a, b, ...)`, `max(a, b, ...)`, the plain variables (`turn`, `food`,
  `population_limit`, ...) and `var(name)`.
- `{"action": "set_var", "name": "x", "value": "<expr>"}` sets a per-turn variable (cleared every turn);
  add `"persist": true` for a global one that lasts the whole game. Rules run in order, so later rules see earlier sets.
- `var(name)` looks up this turn's variables, then globals, then built-in counters named after the count conditions with
  their filters after underscores: `population_13`, `population_infantry`, `idle_units_1`, `enemy_units_visible_13_within_2`,
  `building_count_23_complete`, `foundation_count_2_without_builders`, `resource_food`, `resource_remaining_food_within_1`,
  `bucket_size_<bucket>`, `tech_researched_5` / `resource_available_gold` (1 or 0), `enemy_buildings_visible_1_23`, `land`, ...
  Values a script can derive are left out (headroom is `population_limit - population`, `food` is `resource_food`).
  A script variable read before it is set is 0; names that are neither built-in nor set anywhere are reported at load.
- `within_N` counts around home by default, or around an anchor: `_within_N_of_<bucket>` (the unit nearest the rest of
  the bucket) or `_within_N_of_target` (the candidate being scored, see below). In conditions: `"within": N, "from": "<bucket>"`.
- Target picking on `attack` and `move`: `"targets"` is `enemy_units`, `enemy_buildings`, `known_resources` (with
  `"category"`), `own_buildings`, `unexplored` or `home`, narrowed by `target_unit_ids` / `target_unit_types` /
  `target_building_ids`. `"score"` is evaluated per candidate (it may use `var(target_distance)`, from the group's center,
  `var(target_distance_home)`, and any `_of_target` counter); the highest wins, candidates under `"min_score"` are skipped.
  `"together": true` sends the whole group to one target (units over two spaces from its center regroup first).
  On `attack`, `"order": "zone"` or `"space"` attacks everything in the chosen candidate's zone/space instead of chasing it;
  with a defensive stance the units then stay in that space.
- `{"value_at_least": {"value": "<expr>", "amount": <expr>}}` (and `_at_most` / `_equals`) branches on any expression.
