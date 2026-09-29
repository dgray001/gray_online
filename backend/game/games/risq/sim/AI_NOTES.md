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
- Villagers still never garrison or flee.
- Finish faster: a hunter squad for stragglers once the main army has won.
- v0.4's remaining wins are games where its own heavies sit on its Redoubt zone; raze the Redoubt earlier.
- Land and banked gold decide close score games; the ai still almost never razes outside an assault.
