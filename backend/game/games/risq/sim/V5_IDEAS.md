# v5 ideas

v4 baseline: 292/300 vs v3 (no infantry exploring or scouting, attack at 12 infantry). Remaining losses are
early military collapses where v4 loses most of its villagers.

## Military (biggest lever)
- Per-villager threat response: garrison in the nearest VC/outpost, flee, or fight back if the raid is weak (single unupgraded blunt vs loom villagers). Needs a per-unit threat check in the ai package; the home-radius check froze the whole economy when tried.
- Defend villagers wherever they're attacked, not only near home.
- Raid v3 villagers once v4's own economy is safe.
- Second barracks vs blacksmith first: measure which gives more military by turn 30.

## Early economy
- Early wood is walking-bound: nearest groves/bushes are an adjacent space away (~2 turns). Measure moving vs gathering stamina; try placing the first house near the nearest grove.
- Start infantry production from a partially staffed bucket plus stockpile instead of waiting for full staffing.
- `research` has no queue depth; the VC idles briefly around Farming.

## Expansion
- Outposts at gathering clusters (N+ villagers more than K spaces from a garrisonable building) as forward bases; needs a new ai action.
- Land and banked gold decide close score games; v4 almost never razes.

## Tooling
- `gatherDemandWeights` fix changed v3's behavior too (v3 uses balanced gather); decide whether v3 should stay on the old math as a frozen baseline.
- CPU profile of a game (~2.5s each, ~1.7x overhead when parallel).
