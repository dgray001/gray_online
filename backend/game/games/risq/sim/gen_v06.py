"""Generates config/ai/v0.6.json: a generalist driven only by game state.

Each turn: read state -> remember the enemy -> decide wants -> turn wants into per-turn resource rates, minus the bank
-> size one gather bucket per resource -> spend -> fight. Blunts raid villagers away from defensive buildings;
heavies and piercers form the assault army; nothing keys off the turn number."""
import json, os, sys

VIL, BLUNT, PIERCE, HEAVY = 1, 11, 12, 13
VC, HOUSE, FARM, SMITH, OUTPOST, BARRACKS, REDOUBT = 1, 2, 3, 11, 21, 22, 23
FARMING, ATTACK, ARMOR, MERCS, PIKES = 1, 2, 3, 4, 5
DEFENSIVE = "1_21_23"  # buildings that shoot at their whole space

P = dict(
    vil_share=0.5,     # most of the population ceiling the economy may take
    horizon=10,
    decay=0.3,
    # raiding
    raid_min=3,        # blunts needed before raiding
    raid_odds=1.4,     # we must be this much stronger where we raid
    vil_worth=0.4,     # a villager is worth this many blunts in a fight
    raid_frac=0.3, raid_max=8,
    raid_pause=20,     # turns raids stop after a party is caught, times the number of parties lost so far  # the raid party: this share of our blunts, at most this many
    # assault (heavies + piercers)
    assault_min=10, assault_odds=1.3,
    early_blunts=3,
    invest_share=0.3,
    vil_crush_odds=2,  # villagers fight only with this much more strength than the enemy soldiers in their space
    sustain=15,
    plan_turns=3,      # turns to gather for a building that has been judged worth it        # a production building is only worth it if we can keep it busy this many turns
    lead=5,            # turns ahead we project income when deciding to build  # share of gatherers guaranteed to one-off investments when there are any
)
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'config', 'ai', 'v0.6.json')
for arg in sys.argv[1:]:
    k, v = arg.split('=')
    if k == 'out':
        OUT = v
    else:
        P[k] = float(v)

rules = []
def rule(when, *then): rules.append({"when": when, "then": list(then)})
ALWAYS = {"always": {}}
def setv(name, value, persist=False):
    a = {"action": "set_var", "name": name, "value": value}
    if persist:
        a["persist"] = True
    return a
def ge(expr, n): return {"value_at_least": {"value": expr, "amount": n}}
def le(expr, n): return {"value_at_most": {"value": expr, "amount": n}}
def all_(*c): return {"all": list(c)}
def any_(*c): return {"any": list(c)}
ECO = {"eligible": ["gather"], "exclude_buckets": True}
def build(b, builders): return dict({"action": "build", "building_id": b, "max": builders}, **ECO)
def one(x): return f"min(1, max(0, {x}))"  # 1 when x >= 1, 0 when x <= 0
H = P['horizon']

# ================= state =================
rule(ALWAYS,
     setv("vils", "var(population_1)"),
     setv("B", "var(population_11)"), setv("Pi", "var(population_12)"), setv("Hv", "var(population_13)"),
     setv("mil", "var(B) + var(Pi) + var(Hv)"),
     setv("headroom", "population_limit - population"),
     setv("vcs", "var(building_count_1_complete)"),
     setv("barracks", "var(building_count_22_complete)"),
     setv("redoubts", "var(building_count_23_complete)"),
     setv("smiths", "var(building_count_11_complete)"),
     setv("farming", "var(tech_researched_1)"),
     setv("pikes", "var(tech_researched_5)"),
     setv("rate", "6 + 1.5 * var(farming)"),  # what a villager brings in per turn, walking included
     )
# remembered enemy: the most seen recently, fading slowly
rule(ALWAYS, *(setv(n, f"max(var({n}) - {P['decay']}, var(enemy_units_visible_{src}))", True)
               for n, src in (("eB", 11), ("eP", 12), ("eH", 13), ("eV", 1))),
     setv("eMil", "var(eB) + var(eP) + var(eH)"),
     # fighting power in blunt-equivalents (a heavy ~4 blunts, a piercer ~2 against heavies)
     setv("ePow", "var(eB) + 2 * var(eP) + 4 * var(eH)"))

# ================= wants =================
# unit stats the script can't look up yet (production per building per turn at the current stamina numbers)
PR = {VIL: 1.0, BLUNT: 1.0, PIERCE: 0.71, HEAVY: 0.25}
HOUSE_POP, SPACE_GOLD, FARM_WORKERS = 5, 2, 2
HOUSE_BUILD_STAMINA = 14
FARM_RENEW_WOOD, FARM_FOOD = 60, 200  # no renew-cost lookup yet; a farm holds 200 food
MAX_NEW_FARMS = 3  # farm foundations started per turn
WILD_REACH = 2  # spaces from home villagers walk for wild food once farms are possible  # no renew-cost lookup yet
RES = ("food", "wood", "stone", "gold")
def uc(r, i): return f"var(unit_cost_{r}_{i})"
def bc(r, i): return f"var(building_cost_{r}_{i})"
def tc(r, i): return f"var(tech_cost_{r}_{i})"
TECH_GAIN = {ATTACK: 0.25, ARMOR: 0.15}  # rough share an army gains from each Blacksmith tech (hardcoded stats)
def tcost(t): return " + ".join(tc(r, t) for r in RES)
def bcost(b): return " + ".join(bc(r, b) for r in RES)
def gatherable(i): return f"({uc('food', i)} + {uc('wood', i)} + {uc('stone', i)})"  # the part villagers must gather
rule(ALWAYS,
     # how well each unit does against the remembered enemy mix (blunt > piercer > heavy > blunt; 1 = even)
     setv("es", "var(eB) + var(eP) + var(eH) + 1"),
     setv("vB", f"(var(eB) + 1.3 * var(eP) + 0.4 * var(eH) + 1 + {P['early_blunts']} * {one('1 - var(redoubts)')}) / var(es)"),
     setv("vP", "(0.75 * var(eB) + var(eP) + (0.8 + 0.5 * var(pikes)) * var(eH) + 0.5) / var(es)"),
     setv("vH", "(1.3 * var(eB) + 0.8 * var(eP) + var(eH) + 1) / var(es)"),
     # what a busy producer eats per turn, in villager-gathered resources
     setv("barracks_spend", f"max({gatherable(BLUNT)} * {PR[BLUNT]}, {gatherable(PIERCE)} * {PR[PIERCE]})"),
     setv("redoubt_spend", f"{gatherable(HEAVY)} * {PR[HEAVY]}"),
     # every producer we already have (built or being built) and what it eats per turn while busy
     setv("n_barracks", "var(building_count_22)"), setv("n_redoubts", "var(building_count_23)"),
     setv("committed", f"var(vcs) * {gatherable(VIL)} * {PR[VIL]} * {one('var(vil_want_last)')}"
                       " + var(n_barracks) * var(barracks_spend) + var(n_redoubts) * var(redoubt_spend)"),
     # spare capacity: current income plus the bank spread over the sustain window, less that
     setv("income_proj", "var(vils) * var(rate)"),
     setv("bank_rate", f"max(0, food + wood + stone - {uc('food', VIL)} - {bc('wood', HOUSE)}) / {P['sustain']}"),
     setv("spare", "var(income_proj) + var(bank_rate) - var(committed)"),
     # spare capacity split by how well heavies do against the enemy compared to the best barracks unit
     setv("shareH", "var(vH) / (var(vH) + max(var(vB), var(vP)))"),
     setv("u_barracks", "max(0, var(spare)) * (1 - var(shareH)) / max(1, var(barracks_spend))"),
     setv("u_redoubt", "max(0, var(spare)) * var(shareH) / max(1, var(redoubt_spend))"),
     # one more is worth it when keeping it that busy for the sustain window repays its cost
     *(setv(f"bar_{n}", f"({bc('food', b)} + {bc('wood', b)} + {bc('stone', b)} + {bc('gold', b)}) / max(1, var({n}_spend) * {P['sustain']})")
       for n, b in (("barracks", BARRACKS), ("redoubt", REDOUBT))),
     setv("gold_income", f"{SPACE_GOLD} * land"),
     # Redoubts also need the gold that keeps heavies coming and the stone to build them
     setv("redoubt_gold_ok", f"{one(f'(var(gold_income) + max(0, gold) / ' + str(P['sustain']) + ') / max(1, ' + uc('gold', HEAVY) + ' * ' + str(PR[HEAVY]) + ') - var(n_redoubts)')}"),
     setv("barracks_want", f"{one('var(vils)')} * {one('(var(u_barracks) - var(bar_barracks)) * 100')}"),
     setv("redoubt_want", f"{one('var(vils)')} * var(redoubt_gold_ok) * {one('(var(u_redoubt) - var(bar_redoubt)) * 100')}"),
     # start saving when most of the way there
     setv("barracks_save", f"{one('var(vils)')} * {one('(var(u_barracks) / max(0.01, var(bar_barracks)) - 0.7) * 3.4')}"),
     setv("redoubt_save", f"{one('var(vils)')} * var(redoubt_gold_ok) * {one('(var(u_redoubt) / max(0.01, var(bar_redoubt)) - 0.7) * 3.4')}"),
     # villagers: enough income to fill our army room within the sustain window, less what the bank covers, up to our share of the ceiling
     setv("army_target", f"var(population_max) * (1 - {P['vil_share']})"),
     setv("unit_spend", f"max({gatherable(BLUNT)}, {gatherable(PIERCE)}, {gatherable(HEAVY)})"),
     setv("target_spend", f"var(army_target) / {P['sustain']} * var(unit_spend) + var(vcs) * {gatherable(VIL)} * {PR[VIL]}"),
     setv("vil_goal", f"min(var(population_max) * {P['vil_share']}, max(0, var(target_spend) - var(bank_rate)) / var(rate) + 2)"),
     setv("vil_want", f"max(0, var(vil_goal) - var(vils)) * {one('var(vcs)')}"),
     setv("vil_want_last", "var(vil_want)", True),
     setv("vil_rate", f"{one('var(vil_want)')} * var(vcs)"),
     # the army fills the rest of the ceiling; barracks split between blunts and piercers by how well each does
     setv("free_room", "max(0, var(population_max) - var(vil_goal) - var(mil))"),
     setv("sB4", "var(vB) * var(vB) * var(vB) * var(vB)"),
     setv("sP4", "var(vP) * var(vP) * var(vP) * var(vP)"),
     setv("needB", "var(free_room) * var(sB4) / (var(sB4) + var(sP4))"),
     setv("needP", "var(free_room) * var(sP4) / (var(sB4) + var(sP4))"),
     # Redoubts add heavies on top unless piercers would do much better
     setv("tH", f"(var(Hv) + var(free_room)) * {one('(var(vH) - 0.8 * var(vP)) * 10')}"),
     setv("barracks_B", "var(barracks) * var(needB) / max(1, var(needB) + var(needP))"),
     setv("barracks_P", "var(barracks) * var(needP) / max(1, var(needB) + var(needP))"),
     setv("redoubt_H", f"var(redoubts) * {one('var(tH) - var(Hv)')}"),
     setv("vc_want", f"{one('1 - var(building_count_1)')} * {one('var(vils)')}"),
     # the army the Blacksmith techs would improve: what we have plus what our producers turn out over the sustain window
     setv("army_value", f"var(B) * {gatherable(BLUNT)} + var(Pi) * {gatherable(PIERCE)} + var(Hv) * ({gatherable(HEAVY)} + {uc('gold', HEAVY)})"),
     setv("proj_army", f"var(army_value) + (var(n_barracks) * var(barracks_spend) + var(n_redoubts) * var(redoubt_spend)) * {P['sustain']}"),
     # a Blacksmith once its first tech (attack) repays both
     setv("smith_want", f"{one('1 - var(building_count_11)')} * {one('var(vils)')} * {one(f'(var(proj_army) * {TECH_GAIN[ATTACK]} - ({tcost(ATTACK)}) - ({bcost(SMITH)})) / 50 + 1')}"),
     # houses: enough room for ~4 turns of production, more urgent the sooner we are capped
     setv("prod_rate", f"var(vil_rate) * {PR[VIL]} + var(barracks) * {PR[BLUNT]} + var(redoubts) * {PR[HEAVY]}"),
     setv("house_want", f"max(0, (var(prod_rate) * 4 + 1 - var(headroom)) / {HOUSE_POP}) * {one('var(population_max) - population_limit')} * {one('var(vils)')}"),
     setv("turns_left", "var(headroom) / max(0.1, var(prod_rate))"),
     setv("house_urgency", f"min(1, max(0, (4 - var(turns_left)) / 4)) * {one('var(house_want) * 2')}"),
     # enough builders to finish before we're capped: the work over what villagers can do in the turns left (half a turn to walk)
     setv("vil_stamina", "8 + 2 * var(farming)"),
     # enough builders to finish a turn before we're capped (walking there takes part of the first), rounded up
     setv("house_builders", f"{one('var(house_want) * 2')} * (max(1, {HOUSE_BUILD_STAMINA} / (var(vil_stamina) * max(0.5, var(turns_left) - 1))) + 0.49)"),
     # farms for the food workers that nearby wild food can't carry (~300 food per worker, few gather slots)
     # wild food close enough to walk to (villagers carry nothing back, so distance is a one-time walk; farther is enemy-side risk),
     # in villagers it keeps busy over the horizon; farms cover the rest of the food workers
     setv("wild_workers", f"var(resource_remaining_food_within_{WILD_REACH}) / (var(rate) * {H})"),
     setv("farm_want", f"var(farming) * max(0, max(0, var(food_workers) - var(wild_workers)) / {FARM_WORKERS} + 1 - (var(building_count_3) - var(building_count_3_depleted)))"),
     )


# ---- what gets built this turn, and the villagers each build takes ----
def at_least_1(x): return one(f"({x} - 0.999) * 1000")
def affordable(b):
    """1 once the bank covers every part of the building's cost (a foundation can't start before)"""
    return " * ".join(at_least_1(f"{r} - {bc(r, b)} + 1") for r in RES)
_BUILD_FIRES = {
    "house":    (HOUSE,    at_least_1("(var(house_want) - var(foundation_count_2)) * 2"), "var(house_builders)"),
    "vc":       (VC,       at_least_1("var(vc_want)") + " * " + at_least_1("1 - var(foundation_count_1)"), "4"),
    "barracks": (BARRACKS, at_least_1("var(barracks_want)") + " * " + at_least_1("1 - var(foundation_count_22)"), "3"),
    "redoubt":  (REDOUBT,  at_least_1("var(redoubt_want)") + " * " + at_least_1("1 - var(foundation_count_23)"), "4"),
    "smith":    (SMITH,    at_least_1("var(smith_want)") + " * " + at_least_1("1 - var(foundation_count_11)"), "2"),
    # farms go up in parallel, as many as are wanted (a few per turn)
    **{f"farm{i}": (FARM, at_least_1(f"(var(farm_want) - var(foundation_count_3) - {i}) * 2"), "2") for i in range(MAX_NEW_FARMS)},
    # land pays gold: claim a space for every 2 gold/turn we are short, once we can defend it
    "outpost":  (OUTPOST,  at_least_1("var(barracks)") + " * " + at_least_1("1 - var(foundation_count_21)") + " * " + at_least_1("var(outpost_want)"), "1"),
}

# a build fires, and holds villagers back, only once it can be paid for; until then the plan gathers for it
BUILD_FIRES = {n: (b, cond + " * " + affordable(b), k) for n, (b, cond, k) in _BUILD_FIRES.items()}

# ================= resource rates and buckets =================
rule(ALWAYS,
     *(setv(f"{r}_rate", f"var(vil_rate) * {uc(r, VIL)} * {PR[VIL]} + var(barracks_B) * {uc(r, BLUNT)} * {PR[BLUNT]}"
                         f" + var(barracks_P) * {uc(r, PIERCE)} * {PR[PIERCE]} + var(redoubt_H) * {uc(r, HEAVY)} * {PR[HEAVY]}"
                         # upkeep that runs as long as production does: housing for it, and renewing the farms that feed it
                         f" + var(prod_rate) * {bc(r, HOUSE)} / {HOUSE_POP}"
                         + (f" + min(var(food_workers), {FARM_WORKERS} * var(building_count_3_complete)) * var(rate) * {FARM_RENEW_WOOD} / {FARM_FOOD}" if r == 'wood' else "")) for r in RES),
     # techs we intend: Farming, and the Blacksmith techs once one stands
     *(setv(f"tech_{r}", f"(1 - var(farming)) * {tc(r, FARMING)} + {one('var(smiths) + var(smith_want)')} * ((1 - var(pikes)) * {tc(r, PIKES)}"
                         f" + (1 - var(tech_researched_2)) * {tc(r, ATTACK)} + (1 - var(tech_researched_3)) * {tc(r, ARMOR)})") for r in RES),
     setv("outpost_want", f"max(0, (var(gold_rate) + var(tech_gold) / {H} - var(gold_income)) / {SPACE_GOLD})"),
     *(setv(f"lump_{r}", f"var(vc_want) * {bc(r, VC)} + var(barracks_save) * {bc(r, BARRACKS)} + var(redoubt_save) * {bc(r, REDOUBT)}"
                         f" + var(smith_want) * {bc(r, SMITH)} + var(farm_want) * {bc(r, FARM)} + min(1, var(outpost_want)) * {bc(r, OUTPOST)}"
                         f" + var(house_want) * {bc(r, HOUSE)} * (1 + ({H} - 1) * var(house_urgency))"
                         + (f" + var(building_count_3_depleted) * {FARM_RENEW_WOOD}" if r == 'wood' else "")) for r in RES),
     # keep enough on hand to start the next unit
     setv("buffer_food", f"max({uc('food', VIL)}, {uc('food', BLUNT)}, {uc('food', PIERCE)}, {uc('food', HEAVY)} * {one('var(redoubts)')})"),
     setv("buffer_wood", f"{bc('wood', HOUSE)} + max({uc('wood', BLUNT)}, {uc('wood', PIERCE)})"),
     setv("buffer_stone", "0"), setv("buffer_gold", "0"),
     *(setv(f"{r}_need", f"max(0, var({r}_rate) + (var(lump_{r}) + var(tech_{r}) + var(buffer_{r}) - {r}) / {H})") for r in RES),
     setv("stone_need", "var(stone_need) * var(resource_available_stone)"),
     # land already pays gold; mines only make up the rest
     setv("gold_need", "max(0, var(gold_need) - var(gold_income)) * var(resource_available_gold)"),
     *(setv(f"fire_{n}", cond) for n, (_, cond, _) in BUILD_FIRES.items()),
     # villagers are held back only for the builds that fire this turn, plus one for a foundation nobody is building
     setv("builders", "min(var(vils), " + " + ".join(f"var(fire_{n}) * {k}" for n, (_, _, k) in BUILD_FIRES.items())
                      + f" + {one('var(foundation_count_without_builders)')})"),
     setv("workers", "max(0, var(vils) - var(builders))"),
     # buildings already judged worth it are part of the plan: funded like continuous production, over a few turns
     *(setv(f"planned_{r}", f"var(vc_want) * {bc(r, VC)} + var(barracks_want) * {bc(r, BARRACKS)} + var(redoubt_want) * {bc(r, REDOUBT)}"
                            f" + var(smith_want) * {bc(r, SMITH)} + var(farm_want) * {bc(r, FARM)} + var(house_want) * {bc(r, HOUSE)}") for r in RES),
     # continuous production and planned buildings are fed first; speculative saving gets a guaranteed share and whoever is left
     *(setv(f"rf_{r}", f"min(var({r}_need), var({r}_rate) + max(0, var(planned_{r}) - {r}) / {P['plan_turns']}) / var(rate)") for r in RES),
     *(setv(f"lf_{c}", f"max(0, var({c}_need) / var(rate) - var(rf_{c}))") for c in RES),
     # the guarantee grows with the economy: a young one puts nearly everyone on production
     setv("maturity", "min(1, var(vils) / max(1, var(vil_goal)))"),
     setv("guaranteed", f"min(var(lf_food) + var(lf_wood) + var(lf_stone) + var(lf_gold), {P['invest_share']} * var(workers) * var(maturity) * var(maturity))"),
     setv("scale_r", "min(1, (var(workers) - var(guaranteed)) / max(0.01, var(rf_food) + var(rf_wood) + var(rf_stone) + var(rf_gold)))"),
     setv("left", "max(0, var(workers) - (var(rf_food) + var(rf_wood) + var(rf_stone) + var(rf_gold)) * var(scale_r))"),
     setv("scale_l", "min(1, var(left) / max(0.01, var(lf_food) + var(lf_wood) + var(lf_stone) + var(lf_gold)))"),
     *(setv(f"w_{c}", f"var(rf_{c}) * var(scale_r) + var(lf_{c}) * var(scale_l)") for c in RES),
     setv("food_workers", "var(w_food)", True),
     # spare gatherers go to food and wood in proportion to how much each is needed
     setv("spare", "max(0, var(workers) - (var(w_food) + var(w_wood) + var(w_stone) + var(w_gold)))"),
     setv("spare_food", "(var(food_need) + 1) / (var(food_need) + var(wood_need) + 2)"),
     setv("w_food", "var(w_food) + var(spare) * var(spare_food)"),
     setv("w_wood", "var(w_wood) + var(spare) * (1 - var(spare_food))"),
     # once farms are possible, food work is only where farms and nearby wild food have room; the rest cuts wood (which builds the farms)
     setv("food_slots", f"var(farming) * ({FARM_WORKERS} * (var(building_count_3_complete) - var(building_count_3_depleted)) + var(wild_workers)) + (1 - var(farming)) * 1000"),
     # food workers without a slot cut wood only as far as the wanted farms are short of it; otherwise they keep on food
     setv("farm_wood_short", f"max(0, var(farm_want) * {bc('wood', FARM)} + var(building_count_3_depleted) * {FARM_RENEW_WOOD} - wood)"),
     setv("food_over", "min(max(0, var(w_food) - var(food_slots)), var(farm_wood_short) / (var(rate) * 2))"),
     setv("w_food", "var(w_food) - var(food_over)"),
     setv("w_wood", "var(w_wood) + var(food_over)"),
     *({"action": "empty_bucket", "bucket": c} for c in RES),
     # sizes round down (food rounds to nearest), so together they never eat into the builders
     *({"action": "set_bucket", "bucket": c, "size": f"max(0, var(w_{c}) - {0 if c == 'food' else 0.49})",
        "task": {"action": "gather", "category": c, "eligible": ["gather"]}} for c in RES),
     *({"action": "fill_bucket", "bucket": c, "eligible": ["gather"], "unit_types": ["economic"]} for c in ("food", "stone", "gold", "wood")),
     )

# enemy soldiers at our door (the raid comes home for them)
rule(ALWAYS,
     setv("threat_home", "var(enemy_units_visible_infantry_within_1)"),
     setv("guard_home", "var(population_infantry_within_1)"))

# ================= spending, in priority order =================
# Farming first: the Village Center is otherwise always busy with villagers, and farms depend on it
# (8 villagers is roughly when Farming's villager bonuses start paying for themselves)
rule(all_(ge("var(vils)", 8), le("var(farming)", 0)), {"action": "research", "tech_id": FARMING, "queue": 2})
rule(ge("var(vil_want)", 1), {"action": "create", "unit_id": VIL, "queue": 2})
for n in ("house", "vc", "barracks", "redoubt", "smith", *(f"farm{i}" for i in range(MAX_NEW_FARMS))):
    b, _, k = BUILD_FIRES[n]
    rule(ge(f"var(fire_{n})", 1), build(b, k))
rule(ge("var(fire_outpost)", 1), build(OUTPOST, 1))
rule(ALWAYS, dict({"action": "build_foundations"}, **ECO), dict({"action": "renew"}, **ECO), dict({"action": "repair", "max": 1}, **ECO))
rule(all_(ge("var(smiths)", 1), any_(ge("var(eH)", 2), ge("var(needP)", 4))), {"action": "research", "tech_id": PIKES})
# attack/armor: worth it once the army they improve is worth several times their cost (each adds roughly 15%)
for t in (ATTACK, ARMOR):
    rule(all_(ge("var(smiths)", 1), ge(f"var(proj_army) * {TECH_GAIN[t]} - ({tcost(t)})", 0)),
         {"action": "research", "tech_id": t})
# mercenaries: once gold piles up beyond what heavies and techs will use; price mirrors the engine's markup
MERC = "1.3 * ({f} + {w} + {s} + 1.5 * {g})"
def merc(i): return MERC.format(f=uc('food', i), w=uc('wood', i), s=uc('stone', i), g=uc('gold', i))
rule(ALWAYS, setv("gold_surplus", f"gold - var(gold_rate) * {H} - var(tech_gold)"))
rule(all_(ge("var(redoubts)", 1), ge(f"var(gold_surplus) - {merc(BLUNT)}", 0)), {"action": "research", "tech_id": MERCS, "queue": 2})
rule(ALWAYS, setv("hire_B", f"var(gold_surplus) / {merc(BLUNT)}"), setv("hire_P", f"var(gold_surplus) / {merc(PIERCE)}"))
rule(all_(ge("var(tH) - var(Hv)", 1), ge("var(headroom)", 1)), {"action": "create", "unit_id": HEAVY, "queue": 2})
# each barracks queues piercers only for their share of what the barracks should make; blunts fill the rest
rule(all_(ge("var(needP)", 1), ge("2 * var(needP) / max(1, var(needB) + var(needP))", 0.5), any_(ge("var(pikes)", 1), ge("var(eH)", 2))),
     {"action": "create", "unit_id": PIERCE, "queue": "2 * var(needP) / max(1, var(needB) + var(needP))"})
rule(ge("var(needB)", 1), {"action": "create", "unit_id": BLUNT, "queue": 2})
rule(all_(ge("var(hire_P)", 1), ge("var(needP) - var(needB)", 0.01)), {"action": "hire", "unit_id": PIERCE, "max": "var(hire_P)"})
rule(all_(ge("var(hire_B)", 1), ge("var(needB) - var(needP)", 0)), {"action": "hire", "unit_id": BLUNT, "max": "var(hire_B)"})

# ================= gathering =================
rule(ALWAYS, *({"action": "run_bucket", "bucket": c} for c in ("food", "wood", "stone", "gold")))
# renewing a farm is food work: food gatherers do it (renewers already on it are left alone)
rule(ALWAYS, {"action": "renew", "in_bucket": "food", "eligible": ["gather"]})
rule(ALWAYS, dict({"action": "gather", "move_penalty": 2}, **ECO))

# ================= raiding: a few blunts harass the economy and never fight soldiers =================
vw, odds, rmin = P['vil_worth'], P['raid_odds'], P['raid_min']
RAID_ORDERS = ["move", "attack_space", "attack_unit", "attack_zone", "attack_building"]
def soldiers_near(anchor, within):
    return f"(var(enemy_units_visible_11_within_{within}_of_{anchor}) + 2 * var(enemy_units_visible_12_within_{within}_of_{anchor}) + 4 * var(enemy_units_visible_13_within_{within}_of_{anchor}))"
rule(ALWAYS,
     # a party that lost more than half its members since last turn was caught: raids pause, longer after each failure
     setv("raid_lost", "max(0, var(raid_prev) - var(bucket_size_raid))"),
     setv("raid_wiped", f"{one('var(raid_prev)')} * {one('(var(raid_lost) - var(raid_prev) / 2) * 100')}"),
     setv("raid_fails", "var(raid_fails) + var(raid_wiped)", True),
     setv("raid_pause", f"max(var(raid_pause) - 1, var(raid_wiped) * {P['raid_pause']} * var(raid_fails))", True),
     # a small party: a share of our blunts, never fewer than raid_min, and it all comes home if an army is at our door
     setv("raid_size", f"{one(f'var(B) - {rmin} + 1')} * min({P['raid_max']}, max({rmin}, var(B) * {P['raid_frac']}))"
                       f" * {one('3 - var(threat_home) + var(guard_home)')} * {one('1 - var(raid_pause)')}"),
     {"action": "set_bucket", "bucket": "raid", "size": "var(raid_size)", "task": {"action": "explore"}},
     {"action": "fill_bucket", "bucket": "raid", "unit_ids": [BLUNT]},
     # any enemy soldier within reach of the party (two spaces: they close that in a turn): it is a raid, not a fight, so it leaves
     setv("raid_danger", soldiers_near('raid', 2)))
rule(le("var(raid_size)", 0), {"action": "empty_bucket", "bucket": "raid"})
rule(ALWAYS, setv("raid_prev", "var(bucket_size_raid)", True))
rule(ge("var(bucket_size_raid)", 1), {"action": "set_unit_behavior", "in_bucket": "raid", "stance": "defensive"})
rule(all_(ge("var(bucket_size_raid)", 1), ge("var(raid_danger)", 1)),
     {"action": "move", "in_bucket": "raid", "eligible": RAID_ORDERS, "targets": "home", "together": True})
# not under their guns, not near their soldiers, and not near where their army musters (barracks, Redoubts)
SAFE = (f"100 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_target) + 100 * {soldiers_near('target', 2)}"
        f" + 100 * var(enemy_buildings_known_{BARRACKS}_{REDOUBT}_within_2_of_target)")
# hunt: resources among enemy buildings, where villagers work, away from their guns and soldiers; else explore
hunt_score = (f"5 * min(1, var(enemy_buildings_known_within_2_of_target)) + 3 * var(enemy_buildings_known_within_1_of_target)"
              f" - {SAFE} - var(target_distance) / 2")
RAIDING = all_(ge("var(bucket_size_raid)", 1), le("var(raid_danger)", 0))
rule(RAIDING, {"action": "run_bucket", "bucket": "raid"})
for cat in ("food", "wood"):
    rule(RAIDING, {"action": "move", "in_bucket": "raid", "eligible": RAID_ORDERS, "targets": "known_resources", "category": cat,
                   "score": hunt_score, "min_score": 5, "together": True})
# unguarded buildings to burn, then villagers we outnumber (they fight back): nearest first, never near soldiers or defenses
raze_score = f"20 - var(target_distance) - {SAFE}"
raid_score = (f"20 - var(target_distance) - {SAFE}"
              f" - 100 * max(0, {vw} * var(enemy_units_visible_1_within_0_of_target) - var(bucket_size_raid) / {odds})")
rule(RAIDING, {"action": "attack", "in_bucket": "raid", "eligible": RAID_ORDERS, "targets": "enemy_buildings",
               "score": raze_score, "min_score": 0, "together": True})
rule(RAIDING, {"action": "attack", "in_bucket": "raid", "eligible": RAID_ORDERS, "targets": "enemy_units", "target_unit_ids": [VIL],
               "score": raid_score, "min_score": 0, "together": True, "order": "space"})

# ================= scouting: one soldier finds the enemy and stone =================
rule(all_(ge("var(mil)", 1), any_({"enemies_found_at_most": {"amount": 0}}, le("var(resource_available_stone)", 0))),
     {"action": "set_bucket", "bucket": "scout", "size": 1, "task": {"action": "explore"}},
     {"action": "fill_bucket", "bucket": "scout", "unit_types": ["infantry"]},
     {"action": "run_bucket", "bucket": "scout"})
rule(all_({"enemies_found_at_least": {"amount": 1}}, ge("var(resource_available_stone)", 1)), {"action": "empty_bucket", "bucket": "scout"})

# ================= the army: every other soldier; attacks only while locally stronger, else regroups at home =================
ao = P['assault_odds']
def pow_near(side, anchor, within):
    """fighting power (blunt-equivalents) of one side's soldiers near an anchor"""
    if side == 'enemy':
        c = lambda uid: f"var(enemy_units_visible_{uid}_within_{within}_of_{anchor})" if anchor != 'home' else f"var(enemy_units_visible_{uid}_within_{within})"
    else:
        c = lambda uid: f"var(population_{uid}_within_{within}_of_{anchor})" if anchor != 'home' else f"var(population_{uid}_within_{within})"
    return f"({c(11)} + 2 * {c(12)} + 4 * {c(13)})"
rule(ALWAYS,
     # the strongest enemy army we have seen: an army we haven't beaten is still out there, so it fades only slowly
     setv("ePowSeen", "max(var(ePowSeen) * 0.99, var(enemy_units_visible_11) + 2 * var(enemy_units_visible_12) + 4 * var(enemy_units_visible_13))", True),
     setv("homePow", pow_near('mine', 'home', 1)),
     # launch: the soldiers gathered at home beat what we have seen, and are a real army
     setv("go", one(f"var(homePow) - {ao} * var(ePowSeen) + 1") + " * " + one(f"var(homePow) - {P['assault_min']} + 1")),
     # the army's local fight: soldiers within a space of it, plus defensive buildings in its space
     setv("aPowLocal", pow_near('mine', 'army', 1)),
     setv("aThreat", f"{pow_near('enemy', 'army', 1)} + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_army) + {vw} * var(enemy_units_visible_1_within_0_of_army)"),
     setv("a_outnumbered", f"{one(f'(var(aThreat) * {ao} - var(aPowLocal)) * 10')} * {one('var(bucket_size_army)')}"),
     # keep attacking until locally outnumbered (then everyone regroups at home) or too few are left
     setv("assault", f"min(1, max(var(assault) * (1 - var(a_outnumbered)) * {one('var(bucket_size_army) - 3')}, var(go)))", True),
     {"action": "set_bucket", "bucket": "army", "size": "var(assault) * var(mil)", "task": {"action": "explore"}},
     {"action": "fill_bucket", "bucket": "army", "unit_types": ["infantry"], "eligible": ["move", "gather", "attack_unit", "attack_building", "attack_space", "attack_zone"]})
rule(le("var(assault)", 0), {"action": "empty_bucket", "bucket": "army"})
ARMY = dict(in_bucket="army", eligible=["move", "attack_unit", "attack_building", "attack_space", "attack_zone"], together=True)
rule(ge("var(bucket_size_army)", 1), {"action": "set_unit_behavior", "in_bucket": "army", "stance": "aggressive"})
# lowest priority first; each later rule that finds a target overrides
rule(ge("var(bucket_size_army)", 1),
     # nothing known: head for the far side of the map, where the enemy is
     dict({"action": "move", "targets": "known_resources", "category": "wood", "score": "var(target_distance_home) - var(target_distance) / 2"}, **ARMY),
     dict({"action": "attack", "targets": "enemy_units", "target_unit_ids": [VIL], "score": "20 - var(target_distance)"}, **ARMY),
     dict({"action": "attack", "targets": "enemy_buildings", "score": "25 - var(target_distance)"}, **ARMY),
     dict({"action": "attack", "targets": "enemy_buildings", "target_building_ids": [VC, BARRACKS, REDOUBT, OUTPOST], "score": "30 - var(target_distance)"}, **ARMY),
     dict({"action": "attack", "targets": "enemy_units", "target_unit_types": ["infantry"], "score": "40 - 5 * var(target_distance)", "min_score": 30}, **ARMY))
rule(all_(ge("var(bucket_size_army)", 1), ge("var(a_outnumbered)", 1)),
     dict({"action": "move", "targets": "home"}, **ARMY))

# ================= home defence: soldiers not raiding or attacking hold the base and fight only there =================
DEF = dict(exclude_buckets=True, eligible=["move", "attack_unit", "attack_building", "attack_space", "attack_zone"], unit_types=["infantry"])
# every soldier fights back when hit, whatever it was doing (walking home, hitting a building)
rule(ALWAYS, {"action": "set_unit_behavior", "unit_types": ["infantry"], "interrupt_current": True})
rule({"building_count_at_least": {"amount": 1}},
     # soldiers that left the army or the raid come home, whatever they were still doing
     dict({"action": "move", "targets": "home", "together": False, "eligible": DEF["eligible"]}, exclude_buckets=True, unit_types=["infantry"]),
     dict({"action": "attack", "targets": "enemy_units", "score": "10 - 10 * var(target_distance_home)", "min_score": 0}, **DEF))
# no buildings at all (army-only starts): nothing to defend, so hunt with everything
rule({"building_count_equals": {"amount": 0}},
     {"action": "move", "unit_types": ["infantry"], "targets": "unexplored", "together": True},
     {"action": "attack", "unit_types": ["infantry"], "eligible": ["move", "attack_unit", "attack_space"], "targets": "enemy_units",
      "score": "0 - var(target_distance) - 2 * var(enemy_units_visible_infantry_within_1_of_target)", "together": True})


# ================= villagers under attack: every villager sharing a space with enemy soldiers fights or runs =================
# last, so it overrides whatever they were told this turn. They fight together only when they crush what is there:
# villagers at their worth (hardcoded from unit stats: ~0.6 of a blunt, ~1 with Farming) plus our soldiers there,
# against the enemy's soldiers within a space; otherwise all of them take cover, however few the enemy are.
def power_at_unit(side, within):
    c = (lambda uid: f"var(enemy_units_visible_{uid}_within_{within}_of_unit)") if side == 'enemy' else (lambda uid: f"var(population_{uid}_within_{within}_of_unit)")
    return f"({c(11)} + 2 * {c(12)} + 4 * {c(13)})"
VIL_ORDERS = ["gather", "build", "move", "repair", "renew", "attack_unit", "attack_space"]
rule(ALWAYS, setv("vil_worth_now", "0.6 + 0.4 * var(farming)"))
# soldiers a space away reach them within the turn, so they count; attacking their space also takes on the enemy villagers
# there, who fight back (counted at a full villager's worth)
threat_near = f"({power_at_unit('enemy', 1)} + var(enemy_units_visible_1_within_0_of_unit))"
ours_here = f"(var(vil_worth_now) * var(population_1_within_0_of_unit) + {power_at_unit('mine', 0)})"
crush_margin = f"{ours_here} - {P['vil_crush_odds']} * {threat_near}"
soldiers_near = {"value_at_least": {"value": power_at_unit('enemy', 1), "amount": 1}}
soldiers_here = {"value_at_least": {"value": power_at_unit('enemy', 0), "amount": 1}}
rule(ALWAYS,
     {"action": "attack", "unit_ids": [VIL], "eligible": VIL_ORDERS, "targets": "enemy_units", "target_unit_types": ["infantry"],
      "score": "0 - var(target_distance)", "min_score": 0, "order": "space", "resume": True,
      "unit_when": all_(soldiers_here, {"value_at_least": {"value": crush_margin, "amount": 0}})},
     {"action": "garrison", "unit_ids": [VIL], "eligible": VIL_ORDERS,
      "unit_when": all_(soldiers_near, {"value_at_most": {"value": crush_margin, "amount": -0.01}})},
     # out again once no enemy soldier is within a space of them
     {"action": "ungarrison", "unit_ids": [VIL], "unit_when": {"value_at_most": {"value": "var(enemy_units_visible_infantry_within_1_of_unit)", "amount": 0}}})

with open(OUT, 'w') as f:
    f.write('{\n  "rules": [\n' + ',\n'.join('    ' + json.dumps(r) for r in rules) + '\n  ]\n}\n')
print(len(rules), 'rules ->', OUT)
