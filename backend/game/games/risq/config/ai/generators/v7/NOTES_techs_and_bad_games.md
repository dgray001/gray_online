# v7 notes: techs and bad games

## Techs (folded into v7 defaults)

`techs.py` buys the 7 previously unused techs, one `tune_tech_<id>` each (0 never, 1 as soon as affordable). Defaults: 2 Bow Saw 0.8, 3 Prospecting 0.5, 11 Fertilizer 0.8, 64 Ballistics 0.5, 91 Bivouacs 0.8, 92 Escalades 0.5, 102 Shrapnel 0.5.

- Bow Saw and Prospecting are only bought once villager production is done (`goal_1 - own_1 < 1`). Buying them early displaced villager production at the Village Center and was the whole source of their early harm.
- Ballistics, Escalades and Shrapnel are mostly inert: they need a Redoubt, walls or catapults, which the AI rarely has.
- Per-node tech gather bonus, tech-scaled `worker_income` and a stone share of spare workers were tried and removed; none beat the plain version.

500 games vs v0.6, old v0.7 then new:

| Seed | Defeat | Crush+ | Favorable | Bad | Behind |
|---|---|---|---|---|---|
| 1 | 227 / 208 | 279 / 275 | 396 / 421 | 7 / 9 | 97 / 70 |
| 1234 | 194 / 195 | 239 / 246 | 373 / 422 | 17 / 15 | 110 / 63 |

The gain is behind games becoming ahead or winning. Defeat and crush+ are flat within noise. The same config shows 7 bad games on one seed and 17 on another, so bad counts under about 10 are noise.

## The 9 bad games (seed 1, new v0.7)

1. gr1, cs5, tri16: villagers garrison for around 15 turns when 3 to 4 enemy infantry sit near home early (9 to 15 of about 20 garrisoned). Wood stalls (Barracks and Outpost spend it), no houses get built, pop caps, no army. Likely cause: `civilian_defense` counts allies only in the villager's own space (unverified). gr1 is bad with techs off too.
2. ring4, tr5: the army launches at 1 to 6% over the required power and loses almost all of it. Arrival fights run on stance auto-targeting at `attack_stack=2`, so our damage spreads while the enemy focuses. Focus-fire orders only reach units already in the enemy space at turn start.
3. af1, af6, at6 (no economy): the army engages at about even power and loses. One tick in af1 wasted 74% of damage on two catapults because `FOCUS_CAPACITY` uses a fixed 1.5 damage per hit.
4. gr4: 15 vs 20 infantry when attacked. The Fertilizer buy (100 food) runs before military creation in the same turn and food stays under about 150.

## Measuring early economy

Do not use "villagers with no order at decision time". Every one of them gets an order the same turn (checked t14 to t40 in one game). Use the engine's `VillagerIdlePctEarly` and `VCIdlePctEarly` in `results.json`: under 1% for v0.7 in 18 random games, equal to the opponent's.

## Process lessons

- Keep experiments in a copied generator (`v7_exp`) and `config/ai/tmp_*.json`; v7 stays equal to `v0.7.json`.
- `attack_stack=2` is already swept. Do not sweep it again. A smart stack must be derived from game data or fight state.
- Screen variants at 30 to 80 games first. Run 500 games only for 3 or more promising candidates.

## In progress

`v7_exp/stacks.py` with `stack_mode` 1 to 9 (3 each of kill-count, numbers-based and current-health stacks), in 30-game screening.
