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
    power_odds=1.05,   # attack odds once home power reaches 60, falling linearly from assault_odds at power 30
    cap_ramp=0.8,     # share of the population ceiling where the attack odds start falling toward cap_odds
    cap_odds=0.9,      # attack odds (our power vs the enemy's) once at the population ceiling
    assault_odds_noeco=1.1, power_odds_noeco=0.85, cap_odds_noeco=0.8,  # the same three while the army has no economy behind it (-1 = same)
    early_blunts=3,
    invest_share=0.3,
    vil_crush_odds=2,  # villagers fight only with this much more strength than the enemy soldiers in their space
    sustain=15,
    # Redoubt pressure from idle gold: while no Redoubt exists, smoothed idle gold covering merc_batch mercenary hires wants one (it opens Mercenary Contracts and heavies)
    merc_batch=3,
    gold_smooth=0.25,  # weight of the newest turn in the idle gold average, so a one-turn spike from a planned tech does not count
    urg_max=2,         # once wanted, the Redoubt's saved cost is weighted up to this multiple (like housing), by idle gold and barracks beyond rax_free, less with Redoubts owned
    urg_tau=10,        # turns of sustained want for full weight; a flickering want stays near zero
    def_idle=1,        # 1: every free soldier fights an enemy in the base, not only as many as the power model says are needed
    def_far=30,        # def_idle: zone steps (a space crossing counts 6) a soldier will come at exactly enough defenders; extra defenders shrink it in proportion
    smith_push=1,      # 1: the Blacksmith's saved cost grows with how long it is wanted and with the desire for Pikes (2+ heavies seen over time), and it builds first once urgent
    smith_urg_max=3,   # smith_push: saved-cost multiple at full urgency
    smith_tau=6,       # smith_push: turns wanted for full urgency
    rax_free=2,        # barracks that add no urgency
    overwhelm=0,       # 0 = off; else army power over this multiple of the local threat marches on without staging or stopping for contact
    overwhelm_global=0,  # 1: overwhelm compares against the larger of the local threat and the remembered total enemy army (ePowEst)
    entry=3,             # the final order: 0 move into the target space, 1 attack-space order on it, 2 attack-zone order on it
    focus=0,            # soldier targeting: 0 nearest, 1 wounded first, 2 low max health first, 3 high max health first, 4 healthy first
    rest_dist=1,        # spaces from the target where the army regroups and rests (1 = adjacent, 2 = hold a space further out)
    army_aggr=0,         # 1: the army marches and rests in the aggressive stance (attacks anything within 2 spaces)
    army_ab=1,           # attack back when hit (0 = ignore hits and keep following orders)
    army_int=1,          # a hit or sighting may interrupt the current order
    rest_turns=2,       # turns the army rests grouped one space from its target before the attack order
    overwhelm_known=0,  # 1: overwhelm only applies once there is a real enemy army estimate (ePowEst of at least 1)
    spill_vils=1,      # 1: units a fight doesn't need attack enemy villagers in the same space
    spill_bldgs=1,     # 1: then enemy buildings in the same space
    adj_hold=1,        # 0 = off; else leftover units may attack an adjacent space, holding this multiple of the odds-adjusted strength here (1 = just enough, 1.5 = keep a third extra)
    house_tries=1,     # failed attempts at a house foundation before it is cancelled and rebuilt elsewhere
    tries=3,           # the same for every other building
    resend_every=4,    # turns an abandoned foundation waits (with no replacement under way) before a villager is sent back
    # site selection base weights, all signed; site_<building>_<weight> overrides one building type. dist: nearness to the builders,
    # expand: sites in spaces with none of our buildings, aggr: sites toward the closest enemy building, safe: sites away from enemy
    # soldiers and shooting buildings, spread: sites in spaces with fewer of our buildings (negative clusters), vision: poorly
    # seen spaces, region: spaces that bring us closer to holding a region
    site_dist=1.0, site_expand=0.0, site_aggr=0.0, site_safe=1.0, site_spread=0.25, site_vision=0.0, site_region=0.0,
    site_barracks_aggr=0.25, site_redoubt_aggr=0.25,
    hit_damage=1.5,
    fwd_probe=0,      # 1: forward-mode debug variables
    skip_rest_margin=0.3,   # army power above (retreat odds + this) x the known enemy skips the rest turn; 99 disables
    skip_stage_margin=0.6,  # above (retreat odds + this) it skips staging too and goes straight in; 99 disables
    retreat_margin=0.05,  # the retreat tests use the launch odds minus this, so once launched the army retreats only when it falls this far below the strength it took to launch
    retreat_hold=1,   # 1: a retreat keeps the army bucket together, passive and ordered away until no enemy soldier is within two spaces
    memory_drop=1,    # 1: the remembered enemy army shrinks by the enemy power seen vanishing around the army while it was in melee
    stage_adj=1,      # 1: adjacent spaces are taken like the first target: stand on the base's edge zone facing each, rest a turn, then attack, all six directions at once
    stage_after_select=1,# 1: staging can start the same turn a target is chosen
    army_first=45,    # base score of a space holding unidentified enemy units as a target; above 30 an enemy army outranks any building
    forward=1,        # 1: after going in the target is a forward base; the army clears adjacent spaces from it, then takes a new target or goes home
    unknown_mix=1,     # each unidentified enemy unit counts as 0: 1, 1: the average power of the enemy mix seen so far (at least 1), 2: that mix smoothed with one unit of each type
    contact_radius=1, # spaces within which enemy soldiers count as contact (they stop target selection and the march); 0 = only in the army's own space
    entry_clears=1,   # 1: the target and staging are cleared the turn the army is ordered in
    scout_home=1,     # 1: when nothing is left to scout the scouts walk home and their bucket closes there
    approach_near=1,  # 1: the approach space is the neighbor of the target nearest the army (else the first by coordinate, which can be the far side)
    relevant=1,       # 1: launch and the odds thresholds count army-bucket and unbucketed troops only (not scouts or raiders)
    stage_dist=3,     # spaces from the target where staging (regroup, approach, rest) starts; fast units cover about 3 a turn
    army_only=1,      # 1: staging triggers, regroup points, grouping tests and army strength count army-bucket units only (not scouts or raiders)
    army_ab_entry=-1, # -1 = same as army_ab; 0 or 1: attack_back on the turn the army goes in (army_ab applies otherwise)
    edge=1,           # 1: the army rests on the edge zone facing the target, which shows the target space's soldiers in full
    hit_mode=1,       # 0: constant hit_damage; 1: damage per hit computed from unit stats and the engine's damage formula, per attacker and target
    kill_mode=0,       # 0: attackers on a target kill it within kill_ticks ticks; 1: within one turn, counting each attacker's stamina
    kill_ticks=1,     # ticks the attackers on one target spend killing it; about the turn's tick count for one kill per turn
    spread_queue=10,   # follow-up targets each soldier queues, so it keeps script-chosen targets longer within a turn
    engaged_odds_noeco=0.8,  # -1 = same as engaged_odds; else the engaged threshold used while the army has no economy behind it
    engaged_odds=1.0,  # once in contact the army leaves only if the enemy there is stronger    # damage one soldier deals per hit, for sharing attackers over targets
    plan_turns=3,      # turns to gather for a building that has been judged worth it        # a production building is only worth it if we can keep it busy this many turns
    lead=5,            # turns ahead we project income when deciding to build  # share of gatherers guaranteed to one-off investments when there are any
    est_vil_w=0.2,     # a villager counts as this much of a blunt among unidentified enemy units
    est_unid_w=1.0,    # weight of unidentified units in the remembered enemy army
    est_decay=0.98,    # per-turn fade of the remembered enemy army
    est_inspect=1,     # 1: one home soldier stands on the edge zone facing unidentified enemies within est_inspect_reach of home
    est_inspect_reach=2, est_inspect_min=2,  # spaces from home, enemy units needed in the space
    est_close=0,       # military priority: the economy's share of the population ceiling drops by this much when the enemy is within est_close_ref spaces
    est_close_ref=6,
    est_behind=0,      # same, scaled by how far the remembered enemy army exceeds ours
    # round 2: both enemy-army bounds are computed from the start state only (est_bounds=1 computes them; the est_cap* flags choose how they are used)
    est_bounds=0, est_rate_th=8, est_rate_rs=7,  # gather per villager per turn assumed for the theoretical and the reasonable bound
    est_rax2_th=6, est_rax2_rs=10,                 # turn a second barracks could be producing in each bound
    est_rs_n1=10, est_rs_n2=20, est_rs_n3=40, est_mil_share=0.9,  # reasonable: one villager a turn to n1, two every three turns to n2, one every other turn to n3; share of what is left that goes to soldiers
    est_u_rate=0.4, est_u_drop=10,                 # uncertainty grows this much per turn; at est_u_drop the bounds are dropped
    est_widen_th=0.1,  # the theoretical bound is widened by this fraction at the uncertainty limit (linearly before it)
    est_interp=1.0,    # 1: the reasonable bound slides toward the theoretical one as uncertainty grows
    est_capT_est=1, est_capR_est=0,  # hard cap on the army estimate (ePowEst) by the theoretical / reasonable bound
    est_capT_mem=0, est_capR_mem=1,  # the same on the remembered army (ePowSeen) itself
    est_vil2=1,        # 1: unidentified units near a known enemy base are partly villagers (their minimum villager count less those in view)
    est_vil2_max=0.7,  # most of those units that can be counted as villagers
    est_prio=1,        # 1: pressure reduces the village center's claim on food in the barracks planning; 2: it also makes the village center wait for a reserve
    est_prio_close=0.5, est_prio_behind=0.1,  # pressure weights: enemy close, enemy ahead
    est_behind_src=1, est_recent_w=0.3,  # 1: the enemy-ahead term uses a moving average of the current enemy estimate (newest turn weighted est_recent_w) instead of the remembered army
    est_insp_safe=1,   # inspector score drops 100 per visible enemy soldier within a space of the target
    est_insp_cool=2.0,  # turns the home inspector rests after it fires
)
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'config', 'ai', 'v0.6.json')
for arg in sys.argv[1:]:
    k, v = arg.split('=')
    if k == 'out':
        OUT = v
    else:
        P[k] = float(v)

if any(int(P[k]) for k in ('est_capT_est', 'est_capR_est', 'est_capT_mem', 'est_capR_mem', 'est_vil2')):
    P['est_bounds'] = 1
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
IN_FIGHT = {"value_at_least": {"value": "var(enemy_units_visible_military_within_0_of_unit)", "amount": 1}}
# a behavior change only reaches units that differ from the snapshot, so default-behavior rules must skip units the fight rule owns
NOT_IN_FIGHT = {"not": IN_FIGHT}
ECO = {"eligible": ["gather"], "exclude_buckets": True}
def build(b, builders): return dict({"action": "build", "building_id": b, "max": builders}, **ECO)
def eco_odds(name):
    """An odds threshold with its no-economy value blended in by has_eco"""
    noeco = P[name + '_noeco'] if P[name + '_noeco'] >= 0 else P[name]
    return f"({P[name]} * var(has_eco) + {noeco} * (1 - var(has_eco)))"
def one(x): return f"min(1, max(0, {x}))"  # 1 when x >= 1, 0 when x <= 0
H = P['horizon']
# building types whose foundations are tracked for failed build attempts: name -> (building id, failures allowed)
TRACKED = {"house": (HOUSE, int(P['house_tries'])), "vc": (VC, int(P['tries'])), "farm": (FARM, int(P['tries'])),
           "barracks": (BARRACKS, int(P['tries'])), "smith": (SMITH, int(P['tries'])),
           "redoubt": (REDOUBT, int(P['tries'])), "outpost": (OUTPOST, int(P['tries']))}
def foundations_with_builders(b): return f"(var(foundation_count_{b}) - var(foundation_count_{b}_without_builders))"
# a foundation given up on (it could not be cancelled) stops counting, so a replacement can be placed
def pending(n, b): return f"(var(foundation_count_{b}) - var({n}_abandoned))"

# ================= state =================
rule(ALWAYS,
     setv("has_eco", "min(1, var(population_economic) + var(economic_producers))"),
     setv("vils", "var(population_1)"),
     setv("B", "var(population_11)"), setv("Pi", "var(population_12)"), setv("Hv", "var(population_13)"),
     setv("mil", "var(B) + var(Pi) + var(Hv)"),
     setv("headroom", "population_limit - population"),
     setv("vcs", "var(building_count_1_complete)"),
     setv("barracks", "var(building_count_22_complete)"),
     setv("redoubts", "var(building_count_23_complete)"),
     setv("smiths", "var(building_count_11_complete)"),
     setv("farming", "var(tech_researched_1)"),
     setv("vil_worth_now", "0.6 + 0.4 * var(farming)"),
     setv("pikes", "var(tech_researched_5)"),
     setv("rate", "6 + 1.5 * var(farming)"),  # what a villager brings in per turn, walking included
     )
if int(P['est_bounds']):
    # our turn-1 state, read before any rule spends: the bounds assume the enemy mirrors it
    first_turn = one("2 - turn")
    def start(name: str, value: str) -> dict:
        return setv(name, f"{first_turn} * ({value}) + (1 - {first_turn}) * var({name})", True)
    rule(ALWAYS, start("st_pool", "food + wood"), start("st_vils", "var(vils)"), start("st_mil", "var(B) + 2 * var(Pi) + 4 * var(Hv)"),
         start("st_rax", "var(barracks)"), start("st_red", "var(redoubts)"), start("st_vc", "var(vcs)"), start("st_free", "max(0, population_limit - population)"),
         start("st_eco", "var(has_eco)"))
# failed build attempts between turns (builders died or lost the order), read before any rule changes orders
def replacement_exists(n): return f"(var({n}_exist) - var({n}_abandoned))"
rule(ALWAYS, *(a for n, (b, _) in TRACKED.items() for a in (
     setv(f"{n}_exist", f"var(foundation_count_{b})"),
     setv(f"{n}_abandoned", f"min(var({n}_abandoned), var({n}_exist))", True),
     # turns since giving up with nothing else being built; a villager is only sent back after a wait
     setv(f"{n}_resend_wait", f"(var({n}_resend_wait) + 1) * {one(f'var({n}_abandoned)')} * {one(f'1 - {replacement_exists(n)}')}", True),
     setv(f"{n}_lost", f"max(0, min(var({n}_exist), var({n}_with_post)) - {foundations_with_builders(b)})"),
     setv(f"{n}_tries", f"(var({n}_tries) + var({n}_lost)) * {one(replacement_exists(n))}", True),
     # the ban on the failed site ends once a replacement foundation exists, or after a few turns without one
     setv(f"{n}_ban_wait", f"(var({n}_ban_wait) + 1) * var({n}_ban) * {one(f'1 - {replacement_exists(n)}')}", True),
     setv(f"{n}_ban", f"var({n}_ban) * {one(f'1 - {replacement_exists(n)}')} * {one(f'4 - var({n}_ban_wait)')}", True))))
# remembered enemy: the most seen recently, fading slowly
rule(ALWAYS, *(setv(n, f"max(var({n}) - {P['decay']}, var(enemy_units_visible_{src}))", True)
               for n, src in (("eB", 11), ("eP", 12), ("eH", 13), ("eV", 1))),
     setv("eMil", "var(eB) + var(eP) + var(eH)"),
     # fighting power in blunt-equivalents (a heavy ~4 blunts, a piercer ~2 against heavies)
     setv("ePow", "var(eB) + 2 * var(eP) + 4 * var(eH)"))
if int(P['smith_push']):
    # each time more heavies are in view than last turn counts as new ones, so a stream of single heavies adds up
    rule(ALWAYS, setv("heavies_seen", "var(heavies_seen) + max(0, var(enemy_units_visible_13) - var(heavy_vis_last))", True),
         setv("heavy_vis_last", "var(enemy_units_visible_13)", True))
ST_VILS, ST_VCS = "var(st_vils)", "max(1, var(st_vc))"
def rs_made(y: str) -> str:
    """enemy villagers made in y village-center-turns on the reasonable plan: one a turn up to est_rs_n1 villagers, two every three turns up to est_rs_n2, then one every other turn up to est_rs_n3"""
    n1, n2, n3 = f"max(0, {P['est_rs_n1']} - {ST_VILS})", f"max(0, {P['est_rs_n2']} - max({ST_VILS}, {P['est_rs_n1']}))", f"max(0, {P['est_rs_n3']} - max({ST_VILS}, {P['est_rs_n2']}))"
    return f"(min({y}, {n1}) + min(max(0, {y} - {n1}), 1.5 * {n2}) * 0.6667 + min(max(0, {y} - {n1} - 1.5 * {n2}), 2 * {n3}) * 0.5)"
def rs_vils(turns_ago_made: str) -> str:
    """enemy villagers at the least once that many turns of production have passed, from our own start count"""
    return f"({ST_VILS} + {rs_made(f'({turns_ago_made} * {ST_VCS})')})"
EUNID = "var(eUnidentified) * var(eUnknownPower)"
VIL2_RULES = []
if int(P['est_vil2']):
    # unidentified units at least (distance to the nearest known enemy building - 3) spaces from home are near that building, taken as their base
    EUNID = "var(eUnidPow)"
    dm = "(var(enemy_distance) - 3)"
    excluded = " + ".join(f"max(0, 1 - max({n} - {dm}, {dm} - {n})) * var(uw{n})" for n in range(13))
    VIL2_RULES = [*(setv(f"uw{n}", f"var(enemy_units_unidentified_within_{n})") for n in range(13)),
                  setv("eUnidNear", f"{one('var(enemy_buildings_known)')} * max(0, var(enemy_units_unidentified) - ({excluded}))"),
                  # their villagers: at least what a village center making one a turn to 8, then one per two turns, has made, less those in view
                  setv("eVilMin", rs_vils("(turn - 1)")),
                  setv("eVilShare2", f"min({P['est_vil2_max']}, max(0, var(eVilMin) - var(enemy_units_visible_1)) / max(1, var(eUnidNear)))"),
                  setv("eUnidPow", f"var(eUnidNear) * (var(eUnknownPower) * (1 - var(eVilShare2)) + {P['est_vil_w']} * var(eVilShare2)) + (var(enemy_units_unidentified) - var(eUnidNear)) * var(eUnknownPower)")]
rule(ALWAYS,
     setv("eMixB", "(var(eB) + 1) / (var(eMil) + 3)"),
     setv("eMixP", "(var(eP) + 1) / (var(eMil) + 3)"),
     setv("eMixH", "(var(eH) + 1) / (var(eMil) + 3)"),
     setv("eUnknownPower", {0: "1", 1: "max(1, (var(eB) + 2 * var(eP) + 4 * var(eH)) / max(1, var(eB) + var(eP) + var(eH)))", 2: "(var(eB) + 2 * var(eP) + 4 * var(eH) + 7) / (var(eB) + var(eP) + var(eH) + 3)"}[int(P['unknown_mix'])]),
     setv("eUnidentified", "var(enemy_units_unidentified)"),
     *VIL2_RULES,
     setv("ePow", f"max(var(ePow), var(enemy_units_visible_11) + 2 * var(enemy_units_visible_12) + 4 * var(enemy_units_visible_13) + {EUNID})"))

def unknown_power_near(anchor: str, within: int) -> str:
    return f"var(enemy_units_unidentified_within_{within}_of_{anchor}) * var(eUnknownPower)"

# ================= where buildings go: every buildable zone is scored per building type =================
SITE_WEIGHTS = ("dist", "expand", "aggr", "safe", "spread", "vision", "region")
def site_weight(n, w): return P.get(f"site_{n}_{w}", P[f"site_{w}"])
def site_threat():
    """enemy soldier power (seen and unidentified) within a space of the site, plus shooting buildings at triple weight"""
    c = lambda uid: f"var(enemy_units_visible_{uid}_within_1_of_target)"
    return f"({c(11)} + 2 * {c(12)} + 4 * {c(13)} + {unknown_power_near('target', 1)} + 3 * var(enemy_buildings_known_{DEFENSIVE}_within_1_of_target))"
def site_score(n):
    w = lambda name: f"var(site_{name}_{n})"
    return (f"{w('dist')} * (0 - var(target_distance)) + {w('expand')} * (1 - min(1, var(target_own_buildings)))"
            f" + {w('aggr')} * (var(enemy_distance) - var(target_enemy_distance)) + {w('safe')} * (0 - {site_threat()})"
            f" + {w('spread')} * (0 - var(target_own_buildings)) + {w('vision')} * (4 - var(target_vision))"
            f" + {w('region')} * var(target_region_progress)")
SITE_SCORE = {n: site_score(n) for n in TRACKED}
# base weights as variables, so any later rule can override one with set_var before the build rules run
rule(ALWAYS, *(setv(f"site_{w}_{n}", site_weight(n, w)) for n in TRACKED for w in SITE_WEIGHTS))

def power_at_unit(side: str, within: int) -> str:
    c = (lambda uid: f"var(enemy_units_visible_{uid}_within_{within}_of_unit)") if side == 'enemy' else (lambda uid: f"var(population_{uid}_within_{within}_of_unit)")
    unknown = f" + {unknown_power_near('unit', within)}" if side == 'enemy' else ""
    return f"({c(11)} + 2 * {c(12)} + 4 * {c(13)}{unknown})"
threat_near = f"({power_at_unit('enemy', 1)} + var(enemy_units_visible_1_within_0_of_unit))"
ours_here = f"(var(vil_worth_now) * var(population_1_within_0_of_unit) + {power_at_unit('mine', 0)})"
crush_margin = f"{ours_here} - {P['vil_crush_odds']} * {threat_near}"
VIL_SOLDIERS_NEAR = ge(power_at_unit('enemy', 1), 1)
VIL_SOLDIERS_HERE = ge(power_at_unit('enemy', 0), 1)
ECO_DEFENDING = any_(all_(VIL_SOLDIERS_HERE, ge(crush_margin, 0)), all_(VIL_SOLDIERS_NEAR, le(crush_margin, -0.01)))

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
GOLD_IDLE = f"var(gold_idle_last) * {1 - P['gold_smooth']} + {P['gold_smooth']} * max(0, var(gold_surplus))"
MERC_PRICE = f"(1.3 * ({uc('food', BLUNT)} + {uc('wood', BLUNT)} + {uc('stone', BLUNT)} + 1.5 * {uc('gold', BLUNT)}))"
# 0 once a Redoubt exists or is going up, rising to 1 as idle gold passes merc_batch hires and then twice that
MERC_DRIVE = f"min(1, max(0, var(gold_idle_last) / ({P['merc_batch']} * {MERC_PRICE}) - 1)) * (1 - min(1, var(building_count_23)))"
def redoubt_urgency() -> str:
    excess = f"max(0, var(n_barracks) - {P['rax_free']})"
    gold_term = f"min(1, var(gold_idle_last) / ({P['merc_batch']} * {MERC_PRICE}))"
    return f"min(1, {gold_term} + {excess} / ({excess} + 2)) / (1 + 2 * var(n_redoubts)) * var(redoubt_want) * min(1, var(redoubt_age) / {P['urg_tau']})"
SMITH_WEIGHT = f" * (1 + ({P['smith_urg_max']} - 1) * var(smith_urgency))" if int(P['smith_push']) else ""
def redoubt_lump(r): return f" + var(redoubt_save) * {bc(r, REDOUBT)} * (1 + ({P['urg_max']} - 1) * var(redoubt_urgency))"
def tc(r, i): return f"var(tech_cost_{r}_{i})"
TECH_GAIN = {ATTACK: 0.25, ARMOR: 0.15}  # rough share an army gains from each Blacksmith tech (hardcoded stats)
def tcost(t): return " + ".join(tc(r, t) for r in RES)
def bcost(b): return " + ".join(bc(r, b) for r in RES)
def gatherable(i): return f"({uc('food', i)} + {uc('wood', i)} + {uc('stone', i)})"  # the part villagers must gather
VC_CLAIM = " * (1 - var(est_press))" if int(P['est_prio']) >= 1 else ""
def workforce_projection() -> tuple[dict[str, object], ...]:
    return (
        dict(setv("available_workers", "var(available_gatherers)"), unit_when={"not": ECO_DEFENDING}),
        setv("income_proj", "var(available_workers) * var(rate)"),
        setv("bank_rate", f"max(0, food + wood + stone - {uc('food', VIL)} - {bc('wood', HOUSE)}) / {P['sustain']}"),
        setv("committed", f"var(vcs) * {gatherable(VIL)} * {PR[VIL]} * {one('var(vil_want_last)')}{VC_CLAIM}"
                          " + var(building_count_22) * var(barracks_spend) + var(building_count_23) * var(redoubt_spend)"),
        setv("production_surplus", "var(income_proj) + var(bank_rate) - var(committed)"),
        setv("spare", "var(production_surplus)"),
        setv("u_barracks", "max(0, var(spare)) * (1 - var(shareH)) / max(1, var(barracks_spend))"),
        setv("u_redoubt", "max(0, var(spare)) * var(shareH) / max(1, var(redoubt_spend))"),
    )
BEHIND_SRC = "var(eRecent)" if int(P['est_behind_src']) else "var(ePowSeen)"
rule(ALWAYS,
     setv("est_close", f"{one('var(enemy_buildings_known)')} * max(0, 1 - var(enemy_distance) / {P['est_close_ref']})"),
     *([setv("eRecent", f"{1 - P['est_recent_w']} * var(eRecent) + {P['est_recent_w']} * var(ePow)", True)] if int(P['est_behind_src']) else []),
     setv("est_behind", f"min(1, max(0, ({BEHIND_SRC} - (var(B) + 2 * var(Pi) + 4 * var(Hv))) / max(4, {BEHIND_SRC})))"),
     setv("est_press", f"min(1, {P['est_prio_close']} * var(est_close) + {P['est_prio_behind']} * var(est_behind))"),
     setv("vil_share_now", f"max(0.2, {P['vil_share']} - {P['est_close']} * var(est_close) - {P['est_behind']} * var(est_behind))"))
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
     setv("shareH", "var(vH) / (var(vH) + max(var(vB), var(vP)))"),
     *workforce_projection(),
     # one more is worth it when keeping it that busy for the sustain window repays its cost
     *(setv(f"bar_{n}", f"({bc('food', b)} + {bc('wood', b)} + {bc('stone', b)} + {bc('gold', b)}) / max(1, var({n}_spend) * {P['sustain']})")
       for n, b in (("barracks", BARRACKS), ("redoubt", REDOUBT))),
     setv("gold_income", f"{SPACE_GOLD} * land"),
     # Redoubts also need the gold that keeps heavies coming and the stone to build them
     setv("redoubt_gold_ok", f"{one(f'(var(gold_income) + max(0, gold) / ' + str(P['sustain']) + ') / max(1, ' + uc('gold', HEAVY) + ' * ' + str(PR[HEAVY]) + ') - var(n_redoubts)')}"),
     setv("barracks_want", f"{one('var(vils)')} * {one('(var(u_barracks) - var(bar_barracks)) * 100')}"),
     setv("redoubt_want", f"{one('var(vils)')} * var(redoubt_gold_ok) * {one('max((var(u_redoubt) - var(bar_redoubt)) * 100, ' + MERC_DRIVE + ' * 100)')}"),
     setv("redoubt_urgency", redoubt_urgency()),
     setv("redoubt_age", "max(0, var(redoubt_age) + 2 * var(redoubt_want) - 1)", True),
     # start saving when most of the way there
     setv("barracks_save", f"{one('var(vils)')} * {one('(var(u_barracks) / max(0.01, var(bar_barracks)) - 0.7) * 3.4')}"),
     setv("redoubt_save", f"{one('var(vils)')} * var(redoubt_gold_ok) * {one('max((var(u_redoubt) / max(0.01, var(bar_redoubt)) - 0.7) * 3.4, ' + MERC_DRIVE + ' * 100)')}"),
     # villagers: enough income to fill our army room within the sustain window, less what the bank covers, up to our share of the ceiling
     setv("army_target", "var(population_max) * (1 - var(vil_share_now))"),
     setv("unit_spend", f"max({gatherable(BLUNT)}, {gatherable(PIERCE)}, {gatherable(HEAVY)})"),
     setv("target_spend", f"var(army_target) / {P['sustain']} * var(unit_spend) + var(vcs) * {gatherable(VIL)} * {PR[VIL]}"),
     setv("vil_goal", f"min(var(population_max) * var(vil_share_now), max(0, var(target_spend) - var(bank_rate)) / var(rate) + 2)"),
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
     *((setv("smith_age", "max(0, var(smith_age) + 2 * var(smith_want) - 1)", True),
        setv("pikes_want", f"{one('var(heavies_seen) - 1')} * (1 - var(pikes))"),
        setv("smith_urgency", f"var(smith_want) * max(min(1, var(smith_age) / {P['smith_tau']}), var(pikes_want))")) if int(P['smith_push']) else ()),
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
     setv("food_urgency", f"max(0, 1 - food / max(1, {uc('food', VIL)}))"),
     setv("wild_food_safe", f"var(resource_remaining_food_within_2) * (1 - var(food_urgency)) + var(resource_remaining_food_within_4) * var(food_urgency)"),
     setv("wild_workers", f"var(wild_food_safe) / (var(rate) * {H})"),
     setv("farm_want", f"var(farming) * max(0, max(0, var(food_workers) - var(wild_workers)) / {FARM_WORKERS} + 1 - (var(building_count_3) - var(building_count_3_depleted)))"),
     )


# ---- what gets built this turn, and the villagers each build takes ----
def at_least_1(x): return one(f"({x} - 0.999) * 1000")
def affordable(b):
    """1 once the bank covers every part of the building's cost (a foundation can't start before)"""
    return " * ".join(at_least_1(f"{r} - {bc(r, b)} + 1") for r in RES)
_BUILD_FIRES = {
    "house":    (HOUSE,    at_least_1(f"(var(house_want) - {pending('house', HOUSE)}) * 2"), "var(house_builders)"),
    "vc":       (VC,       at_least_1("var(vc_want)") + " * " + at_least_1(f"1 - {pending('vc', VC)}"), "4"),
    "barracks": (BARRACKS, at_least_1("var(barracks_want)") + " * " + at_least_1(f"1 - {pending('barracks', BARRACKS)}"), "3"),
    "redoubt":  (REDOUBT,  at_least_1("var(redoubt_want)") + " * " + at_least_1(f"1 - {pending('redoubt', REDOUBT)}"), "4"),
    "smith":    (SMITH,    at_least_1("var(smith_want)") + " * " + at_least_1(f"1 - {pending('smith', SMITH)}"), "2"),
    # farms go up in parallel, as many as are wanted (a few per turn)
    **{f"farm{i}": (FARM, at_least_1(f"(var(farm_want) - {pending('farm', FARM)} - {i}) * 2"), "2") for i in range(MAX_NEW_FARMS)},
    # land pays gold: claim a space for every 2 gold/turn we are short, once we can defend it
    "outpost":  (OUTPOST,  at_least_1("var(barracks)") + " * " + at_least_1(f"1 - {pending('outpost', OUTPOST)}") + " * " + at_least_1("var(outpost_want)"), "1"),
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
     *(setv(f"lump_{r}", f"var(vc_want) * {bc(r, VC)} + var(barracks_save) * {bc(r, BARRACKS)}{redoubt_lump(r)}"
                         f" + var(smith_want) * {bc(r, SMITH)}{SMITH_WEIGHT} + var(farm_want) * {bc(r, FARM)} + min(1, var(outpost_want)) * {bc(r, OUTPOST)}"
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
                      + f" + {one('var(foundation_count_without_builders) - ' + ' - '.join(f'var({n}_abandoned)' for n in TRACKED))})"),
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
     # protect base sustenance: only shift workers beyond what is needed for continuous production
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
     setv("threat_home", "var(enemy_units_visible_military_within_1) + var(enemy_units_unidentified_within_1)"),
     setv("guard_home", "max(0, var(population_military_within_1) - var(bucket_size_raid) - var(bucket_size_army))"))

# ================= spending, in priority order =================
# Farming first: the Village Center is otherwise always busy with villagers, and farms depend on it
# (8 villagers is roughly when Farming's villager bonuses start paying for themselves)
rule(all_(ge("var(vils)", 8), le("var(farming)", 0)), {"action": "research", "tech_id": FARMING, "queue": 2})
def build_rule(tracker, when, b, builders):
    """Builds normally, or on the side of home away from the last failed site while that site is banned"""
    scored = dict(build(b, builders), targets="build_sites", score=SITE_SCORE[tracker])
    rule(all_(when, le(f"var({tracker}_ban)", 0), le(f"var({tracker}_abandoned)", 0)), scored)
    # an unattended foundation may be the abandoned one, so it must not be reused for the replacement
    rule(all_(when, le(f"var({tracker}_ban)", 0), ge(f"var({tracker}_abandoned)", 1)), dict(scored, reuse_unbuilt_foundation=False))
    rule(all_(when, ge(f"var({tracker}_ban)", 1)),
         dict(scored, reuse_unbuilt_foundation=False, direction="away",
              direction_from={"x": f"var({tracker}_bad_x)", "y": f"var({tracker}_bad_y)"}, **{"from": "home"}))
VIL_WAIT = [ge(f"food - var(est_press) * min(2, var(barracks)) * {uc('food', BLUNT)}", 50)] if int(P['est_prio']) >= 2 else []
rule(all_(ge("var(vil_want)", 1), *VIL_WAIT), {"action": "create", "unit_id": VIL, "queue": f"1 + {one(str(uc('food', VIL)) + ' - var(w_food) * var(rate)')}"})
if int(P['smith_push']):
    # wanted long enough (or Pikes wanted), the Blacksmith builds before the farms and anything else that would spend its wood
    build_rule("smith", all_(le("var(smiths)", 0), ge("var(smith_urgency)", 1), ge("var(fire_smith)", 1)), BUILD_FIRES["smith"][0], BUILD_FIRES["smith"][2])
for n in ("house", "vc", *(f"farm{i}" for i in range(MAX_NEW_FARMS))):
    b, _, k = BUILD_FIRES[n]
    build_rule(n.rstrip("0123456789"), ge(f"var(fire_{n})", 1), b, k)
b_rax, _, k_rax = BUILD_FIRES["barracks"]
build_rule("barracks", all_(le("var(barracks)", 0), ge("var(fire_barracks)", 1)), b_rax, k_rax)
b_smith, _, k_smith = BUILD_FIRES["smith"]
build_rule("smith", all_(ge("var(mil)", 8), le("var(smiths)", 0), ge("var(fire_smith)", 1)), b_smith, k_smith)
rule(all_(ge("var(tH) - var(Hv)", 1), ge("var(headroom)", 1)), {"action": "create", "unit_id": HEAVY, "queue": 2})
rule(all_(ge("var(needP)", 1), ge("2 * var(needP) / max(1, var(needB) + var(needP))", 0.5), any_(ge("var(pikes)", 1), ge("var(eH)", 2), *([ge("var(heavies_seen)", 2)] if int(P['smith_push']) else []))),
     {"action": "create", "unit_id": PIERCE, "queue": "2 * var(needP) / max(1, var(needB) + var(needP))"})
rule(ge("var(needB)", 1), {"action": "create", "unit_id": BLUNT, "queue": 2})
build_rule("outpost", ge("var(fire_outpost)", 1), OUTPOST, 1)
# types with an abandoned foundation are skipped: it must not absorb the builders meant for its replacement
for n, (b, _) in TRACKED.items():
    rule(le(f"var({n}_abandoned)", 0), dict({"action": "build_foundations", "building_ids": [b]}, **ECO))
rule(ALWAYS, dict({"action": "renew"}, **ECO), dict({"action": "repair", "max": 1}, **ECO))
def site_clear(n):
    """No enemy soldiers, seen or unidentified, within a space of the failed site: the flee rule would not send a builder away"""
    site = {"x": f"var({n}_bad_x)", "y": f"var({n}_bad_y)"}
    return all_({"enemy_units_visible_at_most": {"unit_types": ["military"], "within": 1, "from": site, "amount": 0}},
                {"enemy_units_unidentified_at_most": {"within": 1, "from": site, "amount": 0}})
# after a wait, with no replacement under way and the site clear, one villager goes back to the abandoned foundation
for n, (b, _) in TRACKED.items():
    rule(all_(ge(f"var({n}_abandoned)", 1), le(pending(n, b), 0), ge(f"var({n}_resend_wait)", P['resend_every']), site_clear(n)),
         # not limited to unbucketed villagers: nobody is reserved for an abandoned foundation, so a gatherer is pulled and its bucket refills
         {"action": "build_foundations", "building_ids": [b], "max": 1, "eligible": ["gather"]}, setv(f"{n}_resend_wait", 0, True))
rule(all_(ge("var(smiths)", 1), any_(ge("var(eH)", 2), ge("var(needP)", 4), *([ge("var(heavies_seen)", 2)] if int(P['smith_push']) else []))), {"action": "research", "tech_id": PIKES})
# attack/armor: worth it once the army they improve is worth several times their cost (each adds roughly 15%)
for t in (ATTACK, ARMOR):
    rule(all_(ge("var(smiths)", 1), ge(f"var(proj_army) * {TECH_GAIN[t]} - ({tcost(t)})", 0)),
         {"action": "research", "tech_id": t})
# mercenaries: once gold piles up beyond what heavies and techs will use; price mirrors the engine's markup
MERC = "1.3 * ({f} + {w} + {s} + 1.5 * {g})"
def merc(i): return MERC.format(f=uc('food', i), w=uc('wood', i), s=uc('stone', i), g=uc('gold', i))
rule(ALWAYS, setv("gold_surplus", f"gold - var(gold_rate) * {H} - var(tech_gold)"), setv("gold_idle_last", GOLD_IDLE, True))
rule(all_(ge("var(redoubts)", 1), ge(f"var(gold_surplus) - {merc(BLUNT)}", 0)), {"action": "research", "tech_id": MERCS, "queue": 2})
rule(ALWAYS, setv("hire_B", f"var(gold_surplus) / {merc(BLUNT)}"), setv("hire_P", f"var(gold_surplus) / {merc(PIERCE)}"))
rule(ALWAYS, *workforce_projection())
build_rule("barracks", all_(ge("var(barracks)", 1), ge("var(fire_barracks)", 1),
                          ge("(var(u_barracks) - var(bar_barracks)) * 100", 1)), b_rax, k_rax)
b_red, _, k_red = BUILD_FIRES["redoubt"]
rule(ALWAYS, *workforce_projection())
build_rule("redoubt", all_(ge("var(fire_redoubt)", 1), ge(f"max((var(u_redoubt) - var(bar_redoubt)) * 100, {MERC_DRIVE} * 100)", 1)), b_red, k_red)
build_rule("smith", all_(le("var(smiths)", 0), ge("var(fire_smith)", 1)), b_smith, k_smith)
rule(all_(ge("var(hire_P)", 1), ge("var(needP) - var(needB)", 0.01)), {"action": "hire", "unit_id": PIERCE, "max": "var(hire_P)"})
rule(all_(ge("var(hire_B)", 1), ge("var(needB) - var(needP)", 0)), {"action": "hire", "unit_id": BLUNT, "max": "var(hire_B)"})

# ================= gathering =================
rule(ALWAYS, *({"action": "run_bucket", "bucket": c} for c in ("food", "wood", "stone", "gold")))
# renewing a farm is food work: food gatherers do it (renewers already on it are left alone)
rule(ALWAYS, {"action": "renew", "in_bucket": "food", "eligible": ["gather"]})
rule(ALWAYS, dict({"action": "gather", "move_penalty": 2}, **ECO))

LOCAL_SCOUT_NEEDED = {"count_spaces_at_least": {"where": {"vision": 0}, "within": 4, "from": "home", "amount": 1}}
ENEMY_SCOUT_NEEDED = {"enemies_found_at_most": {"amount": "num_players - 2"}}
GENERAL_SCOUT_NEEDED = any_(all_(ge("var(has_eco)", 1), le("var(resource_available_stone)", 0)),
                            all_(ge("var(mil)", 21), {"count_spaces_at_least": {"where": {"vision": 0}, "amount": 1}}))
rule(ALWAYS, setv("scout_phase", 0))
rule(all_(ge("var(mil)", 1), LOCAL_SCOUT_NEEDED),
     setv("scout_phase", 1),
     {"action": "set_bucket", "bucket": "scout", "size": 1, "task": {"action": "explore", "anchor": "home", "min_distance": 0}})

rule(all_(ge("var(mil)", 1), {"not": LOCAL_SCOUT_NEEDED}, ENEMY_SCOUT_NEEDED),
     setv("scout_phase", 2),
     {"action": "set_bucket", "bucket": "scout", "size": f"1 + {one('var(mil) - 40')}",
      "task": {"action": "explore", "anchor": "center", "min_distance": "max(0, map_size - 2)"}})
rule(all_(ge("var(mil)", 1), {"not": LOCAL_SCOUT_NEEDED}, {"not": ENEMY_SCOUT_NEEDED}, GENERAL_SCOUT_NEEDED),
     setv("scout_phase", 3),
     {"action": "set_bucket", "bucket": "scout", "size": f"1 + {one('var(mil) - 40')}",
      "task": {"action": "explore", "anchor": "self", "min_distance": 0}})
rule(ge("var(scout_phase)", 1),
     {"action": "fill_bucket", "bucket": "scout", "unit_types": ["military"]},
     {"action": "drain_bucket", "to": "scout", "from": ["army", "raid"], "max": 2, "allow_reserved": True},
     {"action": "run_bucket", "bucket": "scout"})
if int(P['scout_home']):
    # nothing left to scout: scouts walk home, and the bucket closes once they are all there
    rule(le("var(scout_phase)", 0), {"action": "move", "in_bucket": "scout", "eligible": ["move"], "targets": "home"})
    rule(all_(le("var(scout_phase)", 0), le("var(bucket_size_scout) - var(population_military_bucket_scout_within_1)", 0)), {"action": "empty_bucket", "bucket": "scout"})
else:
    rule(le("var(scout_phase)", 0), {"action": "empty_bucket", "bucket": "scout"})

# ================= raiding: a few blunts harass the economy and never fight soldiers =================
vw, odds, rmin = P['vil_worth'], P['raid_odds'], P['raid_min']
RAID_ORDERS = ["move", "attack_space", "attack_unit", "attack_zone", "attack_building"]
def soldiers_near(anchor: str, within: int) -> str:
    return f"(var(enemy_units_visible_11_within_{within}_of_{anchor}) + 2 * var(enemy_units_visible_12_within_{within}_of_{anchor}) + 4 * var(enemy_units_visible_13_within_{within}_of_{anchor}) + {unknown_power_near(anchor, within)})"
rule(ALWAYS,
     # a party that lost more than half its members since last turn was caught: raids pause, longer after each failure
     setv("raid_lost", "max(0, var(raid_prev) - var(bucket_size_raid))"),
     setv("raid_wiped", f"{one('var(raid_prev)')} * {one('(var(raid_lost) - var(raid_prev) / 2) * 100')}"),
     setv("raid_fails", "var(raid_fails) + var(raid_wiped)", True),
     setv("raid_pause", f"max(var(raid_pause) - 1, var(raid_wiped) * {P['raid_pause']} * var(raid_fails))", True))
rule(le("var(has_eco)", 0), setv("raid_size", 0))
rule(ge("var(has_eco)", 1),
     # a small party: a share of our blunts, never fewer than raid_min, and it all comes home if an army is at our door
     setv("raid_size", f"{one(f'var(B) - {rmin} + 1')} * min({P['raid_max']}, max({rmin}, var(B) * {P['raid_frac']}))"
                       f" * {one('2 - var(threat_home) + var(guard_home)')} * {one('1 - var(raid_pause)')}"),
     {"action": "set_bucket", "bucket": "raid", "size": "var(raid_size)", "task": {"action": "explore"}},
     {"action": "fill_bucket", "bucket": "raid", "unit_ids": [BLUNT]},
     # any enemy soldier within reach of the party (two spaces: they close that in a turn): it is a raid, not a fight, so it leaves
     setv("raid_danger", soldiers_near('raid', 2)))
rule(le("var(raid_size)", 0), {"action": "empty_bucket", "bucket": "raid"})
rule(ALWAYS, setv("raid_prev", "var(bucket_size_raid)", True))
rule(all_(ge("var(bucket_size_raid)", 1), le("var(raid_danger)", 0)), {"action": "set_unit_behavior", "in_bucket": "raid", "stance": "defensive", "attack_back": True, "unit_when": NOT_IN_FIGHT})
rule(all_(ge("var(bucket_size_raid)", 1), ge("var(raid_danger)", 1)),
     {"action": "move", "in_bucket": "raid", "eligible": RAID_ORDERS, "targets": "retreat", "together": True})
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

# ================= the army: every other soldier; attacks only while locally stronger, else regroups at home =================
ao = "var(ao)"
# share attackers over the enemies instead of all hitting one: about enough per target to kill it within a hit
# (a blunt deals ~1.5 a hit to a blunt, ~2.1 with Attack; hardcoded from unit stats), with a couple of follow-up targets queued
SPREAD_ELIGIBLE = ["move", "auto_attack_unit", "auto_attack_building", "attack_space"]  # spreading replaces engine-chosen attacks and walking, not an attack the script assigned
def hit_damage(attacker, target, attack_tech, pikes):
    """Damage of one tick-long hit (3 stamina) by the engine's formula; Attack and Pikes apply to infantry attackers only"""
    boost = attacker['unit_id'] != VIL
    kinds = attacker['attack_type'].split('_')
    attack = sum(attacker.get(f'attack_{k}', 0) + (2 * attack_tech if boost else 0) for k in kinds)
    defense = sum(target.get(f'defense_{k}', 0) * (1 - (attacker.get(f'penetration_{k}', 0) + (40 * pikes if boost and k == 'piercing' else 0)) / 100) for k in kinds)
    reference = attack - defense if defense / attack <= 0.5 else attack * 4 ** (-defense / attack)
    return 3 * max(0.1, reference / 10)
def is_value(expr, value): return f"max(0, 1 - max({expr} - {value}, {value} - {expr}))"
def hit_table():
    """Expression for the damage per hit of the unit being assigned (var(unit_id)) against the candidate (matched by its max health)"""
    units = [u for u in json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'config', 'units.json'))) if u['unit_id'] in (VIL, BLUNT, PIERCE, HEAVY)]
    assert len({u['max_health'] for u in units}) == len(units), "candidates are told apart by max health"
    terms = []
    for a in units:
        for t in units:
            d = {(x, p): hit_damage(a, t, x, p) for x in (0, 1) for p in (0, 1)}
            gain = f"var(tech_researched_{ATTACK}) * {d[1, 0] - d[0, 0]:.4f} + var(tech_researched_5) * {d[0, 1] - d[0, 0]:.4f} + var(tech_researched_{ATTACK}) * var(tech_researched_5) * {d[1, 1] - d[1, 0] - d[0, 1] + d[0, 0]:.4f}"
            terms.append(f"{is_value('var(unit_id)', a['unit_id'])} * {is_value('var(target_max_health)', t['max_health'])} * ({d[0, 0]:.4f} + {gain})")
    return " + ".join(terms)
HIT = f"({P['hit_damage']} + 0.6 * var(tech_researched_{ATTACK}))"
# a target absorbs the attackers that kill it in kill_ticks ticks (1: within one tick; the turn's tick count: one kill per turn)
SPREAD = {"capacity": f"max(1, var(target_health) / ({HIT} * {P['kill_ticks']}))", "queue": P['spread_queue'], "overflow": "round_robin"}
if int(P['hit_mode']) == 1:
    HITS_PER = "var(unit_stamina) / 3" if int(P['kill_mode']) == 1 else str(P['kill_ticks'])
    SPREAD = dict(SPREAD, capacity="max(0.1, var(target_health))", weight=f"({hit_table()}) * {HITS_PER}")
ONLY = {"in_bucket": "army"} if int(P['army_only']) else {}  # staging and army strength count army-bucket units, not scouts or raiders
HELD = {"held_by_bucket": "army"} if ONLY else {}  # space queries: only spaces holding army units
LAUNCH = {"x": "var(launch_x)", "y": "var(launch_y)"}
HAS_BUILDINGS = {"building_count_at_least": {"amount": 1}}
BKT = "_bucket_army" if ONLY else ""
def pow_near(side: str, anchor: str, within: int) -> str:
    """fighting power (blunt-equivalents) of one side's soldiers near an anchor"""
    if side == 'enemy':
        c = lambda uid: f"var(enemy_units_visible_{uid}_within_{within}_of_{anchor})" if anchor != 'home' else f"var(enemy_units_visible_{uid}_within_{within})"
    else:
        c = lambda uid: f"var(population_{uid}{BKT if anchor == 'army' else ''}_within_{within}_of_{anchor})" if anchor != 'home' else f"var(population_{uid}_within_{within})"
    unknown = f" + {unknown_power_near(anchor, within)}" if side == 'enemy' else ""
    return f"({c(11)} + 2 * {c(12)} + 4 * {c(13)}{unknown})"
def relevant_pow(suffix):
    """power of the troops that are or would be in the army (its bucket and the unbucketed), not scouts or raiders"""
    return "(" + " + ".join(f"{w} * (var(population_{uid}_bucket_none{suffix}) + var(population_{uid}_bucket_army{suffix}))" for uid, w in ((BLUNT, 1), (PIERCE, 2), (HEAVY, 4))) + ")"
EST_BOUNDS = []
if int(P['est_bounds']):
    # Both bounds assume the enemy mirrors our own start: everything below reads the turn-1 state (bank, villagers, soldiers, village centers, barracks,
    # Redoubts, free housing); nothing about a particular start is written in. A barracks costs 150 wood and 30 build stamina (villager-turns lost) and
    # produces 1 blunt a turn once built; a Redoubt adds 1 power a turn (a heavy a quarter turn). Units cost food and wood plus a share of housing
    # (5 population per 30 wood and 14 build stamina). A villager made this turn gathers from the next.
    # Theoretical: the best of three villager plans (none, up to 8, a villager every turn per village center), each with one barracks more or not from est_rax2_th,
    # paying for what it builds. Reasonable: the villager plan is one a turn to 8, then one per two turns, at est_rate_rs gather each.
    v0, vc, k8 = "var(st_vils)", "max(1, var(st_vc))", "max(0, 8 - var(st_vils))"
    def made(plan: str, x: str) -> str:
        return {"none": "0", "to8": f"min({x} * {vc}, {k8})", "pump": f"{x} * {vc}",
                "rs": rs_made(f"({x} * {vc})")}[plan]
    def plan_bound(plan: str, rate: float, rax2: float, share: float = 1.0) -> str:
        """start power + the most blunts this villager plan can fund (resources) and produce (producers), with the producers it started with or one more from rax2; a start with no economy does not grow"""
        h = (30 + rate * 14 / 8) / 5
        labor, started = f"{rate} * 30 / 8", one(f"turn - {rax2} + 1")
        build_turns = "(1 + " + " + ".join(one(f"(30 / (8 * max(1, {v0})) - {n}) * 1000") for n in (1, 2, 3)) + ")"
        no_rax = f"(1 - min(1, var(st_rax)))"
        first = f"{no_rax} * (150 + {labor})"
        produce = f"(var(st_rax) + var(st_red)) * max(0, turn - 1) + {no_rax} * max(0, turn - {build_turns})"
        per_power = f"({70 + h:.3f} - {one('var(st_red)')} * {40 + 0.75 * h:.3f})"
        income, vil_cost = f"var(est_acc_{plan})", f"{50 + h:.3f} * {made(plan, 'turn')}"
        pool = f"({share} * (var(st_pool) + {h:.3f} * var(st_free) - {first} + {income} - {vil_cost}))"
        one_set = f"min({produce}, max(0, {pool}) / {per_power})"
        two_set = f"min({produce} + max(0, turn - {rax2}), max(0, {pool} - {started} * (150 + {labor})) / {per_power})"
        return f"var(st_mil) + var(st_eco) * max({one_set}, {two_set})"
    r_th, r_rs = P['est_rate_th'], P['est_rate_rs']
    unc = f"min({P['est_u_drop']}, var(est_u))"
    EST_BOUNDS = [setv("est_u", f"var(est_u) + {P['est_u_rate']}", True)]
    for plan, rate in (("none", r_th), ("to8", r_th), ("pump", r_th), ("rs", r_rs)):
        EST_BOUNDS.append(setv(f"est_acc_{plan}", f"var(est_acc_{plan}) + {rate} * ({v0} + {made(plan, '(turn - 1)')})", True))
    EST_BOUNDS += [setv("est_bound_th", "max(" + ", ".join(plan_bound(pl, r_th, P['est_rax2_th']) for pl in ("none", "to8", "pump")) + ")"),
                   setv("est_bound_rs", plan_bound("rs", r_rs, P['est_rax2_rs'], P['est_mil_share'])),
                   # in use until the uncertainty reaches its limit. The theoretical bound widens a little with uncertainty (est_widen_th at the limit);
                   # with est_interp the reasonable bound slides toward the theoretical one as uncertainty grows
                   setv("est_bound_on", one(f"({P['est_u_drop']} - var(est_u)) * 100")),
                   setv("est_frac", f"{unc} / {P['est_u_drop']}"),
                   setv("est_cap_th", f"var(est_bound_on) * var(est_bound_th) * (1 + {P['est_widen_th']} * var(est_frac)) + (1 - var(est_bound_on)) * 100000"),
                   setv("est_cap_rs", f"var(est_bound_on) * (var(est_bound_rs) + {P['est_interp']} * (var(est_bound_th) - var(est_bound_rs)) * var(est_frac)) + (1 - var(est_bound_on)) * 100000")]
CAP_MEM = [t for t, flag in (("th", 'est_capT_mem'), ("rs", 'est_capR_mem')) if int(P[flag])]
CAP_EST = [t for t, flag in (("th", 'est_capT_est'), ("rs", 'est_capR_est')) if int(P[flag])]
rule(ALWAYS,
     # the strongest enemy army we have seen: an army we haven't beaten is still out there, so it fades only slowly
     # the enemy army keeps growing after we last saw it (about 0.2 power a turn per known barracks or Redoubt, from
     # watching v0.5) and loses some in fights (2% a turn), never more than their population room allows
     setv("eGrowth", f"0.2 * (var(enemy_buildings_known_{BARRACKS}) + var(enemy_buildings_known_{REDOUBT}))"),
     *EST_BOUNDS,
     setv("eLocalNow", pow_near('enemy', 'army', 1)),
     # while in melee, enemy power that vanishes from around the army is dead, so the memory of the enemy army drops by it
     setv("eDrop", f"max(0, var(eLocalPrev) - var(eLocalNow)) * var(engaged_prev) * {one('var(bucket_size_army)')}" if P['memory_drop'] else "0"),
     setv("eIdent", "var(enemy_units_visible_11) + 2 * var(enemy_units_visible_12) + 4 * var(enemy_units_visible_13)"),
     setv("eSeenNew", f"min(2 * max(0, var(population_max) - var(eV)), max(var(ePowSeen) * {P['est_decay']} + var(eGrowth) - var(eDrop),"
                      f" var(eIdent) + {P['est_unid_w']} * {EUNID}))"),
     *(setv("eSeenNew", f"min(var(eSeenNew), var(est_cap_{t}))") for t in CAP_MEM),
     setv("ePowSeen", "var(eSeenNew)", True),
     # techs: theirs count once we know of a Blacksmith, ours by what we researched (~25% for attack + armor, hardcoded)
     setv("eTech", f"1 + 0.25 * {one(f'var(enemy_buildings_known_{SMITH})')}"),
     setv("myTech", "1 + 0.125 * (var(tech_researched_2) + var(tech_researched_3))"),
     setv("ePowEst", "var(ePowSeen) * var(eTech)"),
     *(setv("ePowEst", f"min(var(ePowEst), var(est_cap_{t}))") for t in CAP_EST),
     setv("homePow", f"{relevant_pow('_within_1') if P['relevant'] else pow_near('mine', 'home', 1)} * var(myTech)"),
     setv("armyPow", f"{relevant_pow('') if P['relevant'] else pow_near('mine', 'home', 1)} * var(myTech)"),
     setv("ao", f"{eco_odds('assault_odds')} - ({eco_odds('assault_odds')} - {eco_odds('power_odds')}) * {one('(var(armyPow) - 30) / 30')}"),
     # near the population ceiling, units are replaced quickly, so the odds we demand fall linearly to cap_odds
     setv("cap_fill", f"min(1, max(0, (population - {P['cap_ramp']} * var(population_max)) / (max(0.01, 1 - {P['cap_ramp']}) * var(population_max))))"),
     setv("ao", f"min(var(ao), {eco_odds('assault_odds')} - ({eco_odds('assault_odds')} - {eco_odds('cap_odds')}) * var(cap_fill))"),
     # launch: the soldiers gathered at home beat what we have seen, and are a real army
     setv("go", one(f"var(homePow) - {ao} * var(ePowEst) + 1") + " * " + one(f"var(homePow) - {P['assault_min']} + 1")),
     # the army's local fight: soldiers within a space of it, plus defensive buildings in its space
     setv("aPowLocal", f"{pow_near('mine', 'army', 1)} * var(myTech)"),
     setv("aThreat", f"{pow_near('enemy', 'army', 1)} * var(eTech) + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_army) + {vw} * var(enemy_units_visible_1_within_0_of_army)"),
     # before contact it needs the odds to go in; once enemy soldiers share its space, running only gets it cut down,
     # so it stays and fights unless they are actually stronger there
     setv("a_engaged", one(pow_near('enemy', 'army', 0))),
     setv("eLocalPrev", "var(eLocalNow)", True), setv("engaged_prev", "var(a_engaged)", True),
     # an engaged army flees only when the enemy is this many times stronger; with no economy there is nothing to retreat to or rebuild with
     setv("engaged_odds_now", f"{P['engaged_odds']} * var(has_eco) + {P['engaged_odds_noeco'] if P['engaged_odds_noeco'] >= 0 else P['engaged_odds']} * (1 - var(has_eco))"),
     # once launched, the retreat odds sit a margin above the launch odds, so a fluctuating enemy doesn't bounce the army back and forth
     setv("a_odds", f"({ao} - {P['retreat_margin']}) - (({ao} - {P['retreat_margin']}) - var(engaged_odds_now)) * var(a_engaged)"),
     setv("a_outnumbered", f"{one('(var(aThreat) * var(a_odds) - var(aPowLocal)) * 10')} * {one('var(bucket_size_army)')}"),
     # a split army only retreats when the whole army is outnumbered (the local test alone makes it regroup instead)
     setv("a_raw", "var(a_outnumbered)"),
     setv("army_split", one(f"var(bucket_size_army) - var(population_military{BKT}_within_0_of_army)")),
     setv("a_total", f"(1 - var(a_engaged)) * {one('(var(ePowEst) * (' + ao + ' - ' + str(P['retreat_margin']) + ') - var(armyPow)) * 10')} * {one('var(bucket_size_army)')} + var(a_engaged) * var(a_raw)"),
     setv("a_outnumbered", "(1 - var(army_fwd)) * ((1 - var(army_split)) * var(a_raw) + var(army_split) * var(a_total)) + var(army_fwd) * var(a_raw)"),
     *([setv("a_local", "var(a_outnumbered)"), setv("a_outnumbered", f"(1 - var(army_fwd)) * var(a_local) + var(army_fwd) * ((1 - var(a_engaged)) * {one('(var(ePowEst) * (' + ao + ' - ' + str(P['retreat_margin']) + ') - var(armyPow)) * 10')} * {one('var(bucket_size_army)')} + var(a_engaged) * var(a_local))")] if P['forward'] else []),
     # keep attacking until locally outnumbered (then everyone regroups at home) or too few are left
     # a retreat holds while enemy soldiers are within two spaces of the army: it stays one bucket, passive, ordered away, instead of being released into the fight
     *([setv("army_retreat", f"max(var(a_outnumbered), var(army_retreat) * {one(pow_near('enemy', 'army', 2))} * (1 - {one('var(population_military_bucket_army_within_0) - var(bucket_size_army) + 1')} * {one('var(building_count)')}))", True), setv("a_outnumbered", "var(army_retreat)")] if P['retreat_hold'] else []),
     setv("assault", f"min(1, max(var(assault) * (1 - var(a_outnumbered)) *{one('var(bucket_size_army) - 3')}, var(go)))", True),
     {"action": "set_bucket", "bucket": "army", "size": ("max(var(assault), var(army_retreat), 1 - min(1, var(building_count))) * var(mil)" if P['retreat_hold'] else "max(var(assault), 1 - min(1, var(building_count))) * var(mil)"), "task": {"action": "move", "targets": "retreat"}},
     {"action": "fill_bucket", "bucket": "army", "unit_types": ["military"], "eligible": ["move", "gather", "attack_unit", "attack_building", "attack_space", "attack_zone"]})
rule(all_(le("var(assault)", 0), le("var(a_outnumbered)", 0), {"building_count_at_least": {"amount": 1}}), {"action": "empty_bucket", "bucket": "army"})
# the space the army launches from is its home when there are no buildings; recorded on the launch turn
rule(all_(ge("var(assault)", 1), ge("var(bucket_size_army)", 1), le("var(launch_set)", 0)),
     {"action": "select_space", "from": "army", "options": [dict({"targets": "closest_spaces", "from": "army", "where": {"unit_count_at_least": 1}, "score": 0}, **HELD)],
      "var_x": "launch_x", "var_y": "launch_y", "persist": True}, setv("launch_set", 1, True))
rule(le("var(assault)", 0), setv("launch_set", 0, True))
# with no buildings a retreat is over once the whole army is back at the launch space
rule(all_(ge("var(army_retreat)", 1), {"not": HAS_BUILDINGS}, {"population_at_least": {"unit_types": ["military"], "in_bucket": "army", "within": 0, "from": LAUNCH, "amount": "var(bucket_size_army)"}}),
     setv("army_retreat", 0, True))
ARMY = dict(in_bucket="army", eligible=["move", "attack_unit", "attack_building", "attack_space", "attack_zone"], together=True)
# soldiers are defensive by default; only those in a fight turn aggressive (see local fights below)
def army_stance_rule(ab, extra=None):
    return dict({"action": "set_unit_behavior", "in_bucket": "army", "stance": "aggressive" if P['army_aggr'] else "defensive", "attack_back": ab, "interrupt_current": bool(P['army_int']), "unit_when": all_(NOT_IN_FIGHT, *([extra] if extra else []))})
if P['army_ab_entry'] < 0:
    rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0)), army_stance_rule(bool(P['army_ab'])))
# lowest priority first; each later rule that finds a target overrides
# Army target selection: select best space across enemy targets (only when score beats existing saved score)
army_target_options = [
    {"targets": "unidentified_units", "score": f"{P['army_first']} - var(target_distance) - 100 * max(0, {ao} * ({soldiers_near('target', 1)} + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_target)) - var(aPowLocal))", "min_score": 0},
    {"targets": "enemy_units", "target_unit_ids": [VIL], "score": "20 - var(target_distance)"},
    {"targets": "enemy_buildings", "score": "25 - var(target_distance)"},
    {"targets": "enemy_buildings", "target_building_ids": [VC, BARRACKS, REDOUBT, OUTPOST], "score": "30 - var(target_distance)"},
    {"targets": "enemy_units", "target_unit_types": ["military"], "score": "40 - 5 * var(target_distance)", "min_score": 30},
]
ENEMY_SOLDIERS_2 = f"(var(enemy_units_visible_military_within_2_of_army) + var(enemy_units_unidentified_within_2_of_army))"
CR = int(P['contact_radius'])
ENEMY_SOLDIERS_1 = f"(var(enemy_units_visible_military_within_{CR}_of_army) + var(enemy_units_unidentified_within_{CR}_of_army))"  # contact: enemy soldiers this close to the army
SPREAD_ARMY = all_(ge("var(bucket_size_army)", 2), le("var(a_engaged)", 0),
                    le(f"var(population_military{BKT}_within_0_of_army) - var(bucket_size_army)", -1))
# a clear surplus over the retreat odds skips the rest turn, a larger one also the staging; only once some of their soldiers have been identified, since counted-only units are valued low
def skip_flag(margin): return one(f"(var(armyPow) - var(ePowEst) * ({ao} - {P['retreat_margin']} + {margin})) * 1000") + " * " + one("var(ePowEst)") + " * " + one("var(bucket_size_army)") + " * " + one("var(eMil)")
rule(ALWAYS, setv("skip_rest", skip_flag(P['skip_rest_margin'])), setv("skip_stage", skip_flag(P['skip_stage_margin'])))
SKIP_REST, SKIP_STAGE_FLAG = ge("var(skip_rest)", 1), ge("var(skip_stage)", 1)
ARMY_GROUPED = all_(ge("var(bucket_size_army)", 1), ge(f"var(population_military{BKT}_within_0_of_army) - var(bucket_size_army)", 0))
TARGET_AT = {"x": "var(army_target_x)", "y": "var(army_target_y)"}
SKIP_STAGE = SKIP_STAGE_FLAG
NEVER = {"value_at_least": {"value": 0, "amount": 1}}
OVERWHELM_AGAINST = "max(var(aThreat), var(ePowEst))" if P['overwhelm_global'] else "var(aThreat)"
OVERWHELM_TERMS = [ge("var(bucket_size_army)", 1), ge(f"var(aPowLocal) - {P['overwhelm']} * {OVERWHELM_AGAINST}", 0.01)] + ([ge("var(ePowEst)", 1)] if P['overwhelm_known'] else [])
OVERWHELM = all_(*OVERWHELM_TERMS) if P['overwhelm'] > 0 else NEVER
def army_near_target(within, amount=1): return {"population_at_least": dict({"unit_types": ["military"], "within": within, "from": TARGET_AT, "amount": amount}, **ONLY)}
# combat, being attacked, retreat or disbanding wipes the target and staging; contact while marching ends the march
rule(any_(ge("var(a_engaged)", 1), ge("var(a_outnumbered)", 1), le("var(bucket_size_army)", 0), all_(le("var(army_stage)", 0), ge(ENEMY_SOLDIERS_1, 1), {"not": OVERWHELM})),
     setv("army_stage", 0, True), setv("army_rest", 0, True), setv("army_target_score", 0, True))
# staging starts once any member is within two spaces of the saved target
rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), le("var(army_stage)", 0), ge("var(army_target_score)", 1), army_near_target(int(P['stage_dist'])), {"not": OVERWHELM}, {"not": SKIP_STAGE}),
     setv("army_stage", 1, True))
# targets are only selected while marching
rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), le("var(army_stage)", 0), any_(le(ENEMY_SOLDIERS_1, 0), OVERWHELM)),
     {"action": "select_space", "from": "army", "options": army_target_options,
      "var_x": "army_target_x", "var_y": "army_target_y", "var_score": "army_target_score",
      "min_score": "var(army_target_score) + 1", "persist": True})
if int(P['stage_after_select']):  # a target chosen within staging range stages at once, instead of being marched straight at this turn
    rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), le("var(army_stage)", 0), ge("var(army_target_score)", 1), army_near_target(int(P['stage_dist'])), {"not": OVERWHELM}, {"not": SKIP_STAGE}),
         setv("army_stage", 1, True))
ARMY_LIVE = all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), ge("var(army_target_score)", 1))
MARCHING = all_(ARMY_LIVE, le("var(army_stage)", 0), any_(le(ENEMY_SOLDIERS_1, 0), OVERWHELM))
# 1. march toward the target while grouped
rule(all_(MARCHING, {"not": SPREAD_ARMY}), dict({"action": "move", "targets": "coordinate", "coordinate": TARGET_AT}, **ARMY))
# 2. when spread while marching, trailing units catch up to the forward group
rule(all_(MARCHING, SPREAD_ARMY),
     dict({"action": "move", "targets": "closest_spaces", "from": TARGET_AT, "where": {"unit_count_at_least": 1}, **HELD}, **ARMY))
CONTACT = all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), le("var(a_engaged)", 0), le("var(army_stage)", 0), ge(ENEMY_SOLDIERS_1, 1), {"not": OVERWHELM}, *([le("var(army_fwd)", 0)] if P['forward'] else []))
# 3. contact while marching: a spread army regroups in one occupied space, a grouped one attacks the nearest soldiers
rule(all_(CONTACT, SPREAD_ARMY), dict({"action": "move", "targets": "closest_spaces", "from": "army", "where": {"unit_count_at_least": 1}}, **ARMY))
rule(all_(CONTACT, ARMY_GROUPED),
     dict({"action": "attack", "targets": "enemy_units", "target_unit_types": ["military"], "score": "40 - 5 * var(target_distance)", "min_score": 30, "order": "space"}, **ARMY))
# 4. staging: once grouped near the target go on to the approach, else regroup two spaces out at the occupied army space nearest the target
STAGING = all_(ARMY_LIVE, ge("var(army_stage)", 1), le("var(army_stage)", 1))
rule(all_(STAGING, ARMY_GROUPED, army_near_target(int(P['stage_dist']), "var(bucket_size_army)")), setv("army_stage", 2, True))
rule(all_(STAGING, SPREAD_ARMY),
     dict({"action": "move", "targets": "closest_spaces", "from": TARGET_AT, "exclude_reference": True,
           "where": {"all": [{"unit_count_at_least": 1}, {"not": {"owner": "enemy"}}]}, **HELD}, **ARMY))
# 5. approach: move together to a space one away from the target and regroup there
APPROACHING = all_(ARMY_LIVE, ge("var(army_stage)", 2))
REST_DIST = int(P['rest_dist'])
AT_ONE_AWAY = all_(ARMY_GROUPED, army_near_target(REST_DIST, "var(bucket_size_army)"), {"not": army_near_target(REST_DIST - 1)})
if REST_DIST == 1:
    rule(all_(APPROACHING, {"not": AT_ONE_AWAY}),
         dict({"action": "move", "targets": "closest_spaces", "from": TARGET_AT, "exclude_reference": True, **({"face": TARGET_AT} if int(P['edge']) else {}), **({"score": "0 - var(target_distance)"} if int(P['approach_near']) else {})}, **ARMY))
ENTRY_NOW = all_(APPROACHING, AT_ONE_AWAY, any_(ge("var(army_rest)", P['rest_turns']), SKIP_REST))
# rest only counts turns spent regrouped one space away; travel and regrouping turns do not
rule(all_(APPROACHING, AT_ONE_AWAY), setv("army_rest", "var(army_rest) + 1", True))
rule(all_(APPROACHING, {"not": AT_ONE_AWAY}), setv("army_rest", 0, True))
rule(all_(APPROACHING, AT_ONE_AWAY, le("var(army_rest)", P['rest_turns'] - 1), {"not": SKIP_REST}), {"action": "unit_stop", "in_bucket": "army", "interrupt_current": True})
# 6. after one full turn of rest (or none, with a big surplus), attack the saved target together
rule(ENTRY_NOW,
     dict({"action": "move", "targets": "coordinate", "coordinate": TARGET_AT}, **ARMY) if int(P['entry']) in (0, 3) else
     dict({"action": "attack", "targets": "coordinate", "coordinate": TARGET_AT, "order": "space" if int(P['entry']) == 1 else "zone"}, **ARMY))

# attack back is on while the army marches and rests, and off on the turn it goes in so its assigned targets are not dropped
if P['army_ab_entry'] >= 0:
    ENTERING = ENTRY_NOW
    ARMY_ON = all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0))
    rule(all_(ARMY_ON, {"not": ENTERING}), army_stance_rule(bool(P['army_ab'])))
    rule(all_(ARMY_ON, ENTERING), army_stance_rule(bool(P['army_ab_entry'])))
# entry 3: enemy soldiers in full view are shared out over our army before it walks in, nearest targets first
if int(P['entry']) == 3:
    rule(ENTRY_NOW,
         {"action": "attack", "in_bucket": "army", "eligible": ARMY['eligible'], "targets": "enemy_units", "target_unit_types": ["military"],
          "score": 0, "spread": dict(SPREAD, unit_score="0 - var(unit_distance)")})
# going in ends the target: it is dropped the turn the entry orders are made, whether or not anyone is found there
if int(P['entry_clears']):
    FORWARD_MARK = [setv("army_fwd", 1, True), setv("fwd_wait", 0, True), setv("fwd_attacking", 0, True), setv("fb_x", "var(army_target_x)", True), setv("fb_y", "var(army_target_y)", True)] if P['forward'] else []
    rule(ENTRY_NOW, setv("army_stage", 0, True), setv("army_rest", 0, True), setv("army_target_score", 0, True), *FORWARD_MARK)
    # a march with no staging ends the target once a unit is inside it
    rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), ge("var(army_target_score)", 1), le("var(army_stage)", 0), SKIP_STAGE, army_near_target(0)),
         setv("army_target_score", 0, True), *FORWARD_MARK)
if P['forward']:
    FB = {"x": "var(fb_x)", "y": "var(fb_y)"}
    FWD = ge("var(army_fwd)", 1)
    ENEMY_PRESENT = {"any": [{"owner": "enemy"}, {"all": [{"unit_count_at_least": 1}, {"not": {"owner": "own"}}]}]}
    ADJ_ENEMY = {"count_spaces_at_least": {"where": ENEMY_PRESENT, "within": 1, "from": FB, "amount": 1}}
    FB_CLEAR = all_({"enemy_units_visible_at_most": {"within": 0, "from": FB, "amount": 0}}, {"enemy_units_unidentified_at_most": {"within": 0, "from": FB, "amount": 0}},
                    {"enemy_buildings_known_at_most": {"within": 0, "from": FB, "amount": 0}})  # nothing of theirs left in the base space: no soldiers, villagers or buildings
    if int(P['fwd_probe']):  # debug variables: adjacent enemy spaces, nearest enemy spaces, army bucket size
        rule(ALWAYS, setv("probe_adj", {"count_spaces": {"where": ENEMY_PRESENT, "within": 1, "from": FB}}), setv("probe_closest", {"closest_spaces": {"from": FB, "exclude_reference": True, "where": ENEMY_PRESENT}}), setv("probe_bucket", "var(bucket_size_army)"))
    # a retreat, a disbanded army or a newly chosen target ends forward mode
    rule(any_(ge("var(a_outnumbered)", 1), le("var(bucket_size_army)", 0), ge("var(army_target_score)", 1)), setv("army_fwd", 0, True), setv("fwd_wait", 0, True), setv("fwd_attacking", 0, True))
    # no adjacent enemy and no target left: the army goes home
    NOTHING_LEFT = [ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), le("var(army_target_score)", 0), le("var(army_stage)", 0), le(ENEMY_SOLDIERS_1, 0), {"not": all_(FWD, ADJ_ENEMY)}]
    rule(all_(*NOTHING_LEFT, HAS_BUILDINGS), dict({"action": "move", "targets": "retreat", "to": "home"}, **ARMY), setv("army_fwd", 0, True))
    rule(all_(*NOTHING_LEFT, {"not": HAS_BUILDINGS}), dict({"action": "move", "targets": "coordinate", "coordinate": LAUNCH}, **ARMY), setv("army_fwd", 0, True))
    # with the forward base cleared, units take adjacent enemy spaces the whole army can beat, each space getting the power it needs
    NEED_T = f"{ao} * ({pow_near('enemy', 'target', 0)} * var(eTech) + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_target))"
    UNIT_POW = "(1 + min(1, max(0, var(unit_id) - 11)) + 2 * min(1, max(0, var(unit_id) - 12))) * var(myTech)"
    if not int(P['stage_adj']):
        rule(all_(FWD, le("var(a_outnumbered)", 0), le("var(a_local)", 0), FB_CLEAR, ADJ_ENEMY),
             {"action": "attack", "in_bucket": "army", "eligible": ARMY["eligible"], "targets": "closest_spaces", "from": FB, "exclude_reference": True,
              "where": ENEMY_PRESENT, "order": "space", "score": f"0 - 100000 * max(0, {NEED_T} - var(armyPow))", "min_score": -1000,
              "spread": {"group_by": "space", "capacity": f"max(2, {NEED_T})", "weight": UNIT_POW, "overflow": "none", "unit_score": "0 - var(unit_distance)"}})
    DIRS = [(1, 0), (1, -1), (0, -1), (-1, 0), (-1, 1), (0, 1)]  # the six neighbors, in the engine's direction order
    def around_fb(k): return {axis: f"var(fb_{axis}) {'+' if d >= 0 else '-'} {abs(d)}" for axis, d in zip("xy", DIRS[k])}
    def present_at(k): return {"count_spaces_at_least": {"where": ENEMY_PRESENT, "within": 0, "from": around_fb(k), "amount": 1}}
    def beatable(k): return {"value_at_most": {"value": f"var(need_{k}) - var(armyPow)", "amount": 0}}
    def claim(k, scale="adj_scale"): return {"group_by": "space", "capacity": f"max(2, var(need_{k}) * var({scale}))", "weight": UNIT_POW, "overflow": "none"}
    FWD_READY = all_(FWD, le("var(a_outnumbered)", 0), le("var(a_local)", 0), FB_CLEAR)
    if int(P['stage_adj']):  # each neighbor's need, then the attacks (turn 3 of stage, rest, attack), then the wait counter
        NEED_TOTAL = " + ".join(f"var(need_{k})" for k in range(6))
        IDLE_POW = "(" + " + ".join(f"{w} * var(idle_units_{uid}_bucket_army)" for uid, w in ((BLUNT, 1), (PIERCE, 2), (HEAVY, 4))) + ") * var(myTech)"
        for k in range(6):
            rule(all_(FWD, present_at(k)), {"action": "select_space", "from": FB, "options": [{"targets": "closest_spaces", "from": around_fb(k), "where": ENEMY_PRESENT, "score": NEED_T}], "var_score": f"need_{k}"})
        E_T = f"({pow_near('enemy', 'target', 0)} * var(eTech) + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_target))"
        M_T = f"({pow_near('mine', 'target', 0)} * var(myTech))"
        GROUP_OUT_T = one(M_T) + " * " + one(f"({E_T} * var(engaged_odds_now) - {M_T}) * 10")  # our units in that space exist and the enemy there outnumbers them
        for k in range(6):  # a group is the units in one adjacent space; it is outnumbered by the usual calculation applied to that space
            rule(all_(FWD, present_at(k)), {"action": "select_space", "from": FB, "options": [{"targets": "closest_spaces", "from": around_fb(k), "where": ENEMY_PRESENT, "score": GROUP_OUT_T}], "var_score": f"group_out_{k}"})
        rule(FWD, setv("group_out_sum", " + ".join(f"var(group_out_{k})" for k in range(6))))
        rule(FWD, setv("adj_scale", f"max(1, 1.1 * var(armyPow) / max(1, {NEED_TOTAL}))"))  # capacities that add up to the whole army, so no unit is left unassigned
        for k in range(6):
            rule(all_(FWD_READY, present_at(k), beatable(k), ge("var(fwd_wait)", 2)),
                 {"action": "attack", "in_bucket": "army", "eligible": ["move"], "targets": "closest_spaces", "from": around_fb(k), "where": ENEMY_PRESENT, "order": "space", "spread": claim(k)})
        rule(ge("var(fwd_wait)", 2), setv("fwd_wait", 0, True), setv("fwd_attacking", 1, True))
        rule(all_(ge("var(fwd_wait)", 1), le("var(fwd_wait)", 1)), setv("fwd_wait", 2, True))
        rule(FWD, setv("army_all_idle", one("var(idle_units_military_bucket_army) - var(bucket_size_army) + 1")), setv("adj_scale_idle", f"max(1, 1.1 * {IDLE_POW} / max(1, {NEED_TOTAL}))"))
        rule(ge("var(army_all_idle)", 1), setv("fwd_attacking", 0, True))  # the whole army is idle: that attack is over
        for k in range(6):  # units that finished early take the remaining spaces at once, no edge zone, no rest
            rule(all_(FWD_READY, present_at(k), beatable(k), ge("var(fwd_attacking)", 1), le("var(fwd_wait)", 0)),
                 {"action": "attack", "in_bucket": "army", "targets": "closest_spaces", "from": around_fb(k), "where": ENEMY_PRESENT, "order": "space", "spread": claim(k, "adj_scale_idle")})
        for k in range(6):  # a big enough surplus goes straight in, as the units that finished early do
            rule(all_(FWD_READY, present_at(k), beatable(k), le("var(fwd_wait)", 0), ge("var(army_all_idle)", 1), SKIP_STAGE_FLAG),
                 {"action": "attack", "in_bucket": "army", "targets": "closest_spaces", "from": around_fb(k), "where": ENEMY_PRESENT, "order": "space", "spread": claim(k, "adj_scale_idle")}, setv("fwd_attacking", 1, True))
        for k in range(6):  # a fresh cycle, only with the whole army idle: everyone stands on the base's edge zone facing the space they will take
            rule(all_(FWD_READY, present_at(k), beatable(k), le("var(fwd_wait)", 0), ge("var(army_all_idle)", 1), {"not": SKIP_STAGE_FLAG}),
                 {"action": "move", "in_bucket": "army", "targets": "closest_spaces", "from": FB, "face": around_fb(k), "spread": claim(k)}, setv("staged_now", 1))
        rule(ge("var(staged_now)", 1), setv("fwd_wait", 1, True))
        rule(all_(ge("var(staged_now)", 1), SKIP_REST), setv("fwd_wait", 2, True))
# ================= home defence: soldiers not raiding or attacking hold the base and fight only there =================
AT_BASE = {"value_at_least": {"value": "var(base_spaces_within_0_of_unit)", "amount": 1}}
AWAY_FROM_BASE = {"value_at_most": {"value": "var(base_spaces_within_0_of_unit)", "amount": 0}}
DEFENDING = all_(AT_BASE, IN_FIGHT)
DEF = dict(exclude_buckets=True, eligible=["move", "attack_unit", "attack_building", "attack_space", "attack_zone", "auto_attack_unit", "auto_attack_building"], unit_types=["military"], unit_when={"not": DEFENDING})
DEF_ENEMY = f"{pow_near('enemy', 'target', 0)} * var(eTech)"
DEF_NEED = f"max(0, {DEF_ENEMY} - {pow_near('mine', 'target', 0)} * var(myTech))"
DEF_WEIGHT = "(1 + min(1, max(0, var(unit_id) - 11)) + 2 * min(1, max(0, var(unit_id) - 12))) * var(myTech)"
DEF_UNIT_SCORE = f"max(0, {DEF_NEED} - var(target_assigned)) / max(1, {DEF_ENEMY}) - var(unit_distance) / (100 * (1 + var(unit_distance)))"
if int(P['def_idle']):
    # extra soldiers still defend: the fuller a space already is beyond its need, the nearer a soldier must be (def_far zone steps at exactly enough)
    DEF_UNIT_SCORE = f"1 + {DEF_UNIT_SCORE} - var(target_assigned) / max(1, {DEF_NEED}) * var(unit_distance) / {P['def_far']}"
DEF_SPREAD = dict(group_by="space", capacity=DEF_NEED, weight=DEF_WEIGHT, overflow="round_robin" if int(P['def_idle']) else "none", unit_score=DEF_UNIT_SCORE)
# every soldier outside the army fights back when hit, whatever it was doing (walking home, hitting a building)
rule(ALWAYS, {"action": "set_unit_behavior", "unit_types": ["military"], "stance": "defensive", "attack_back": True, "interrupt_current": True, "exclude_buckets": True, "unit_when": NOT_IN_FIGHT})
rule(all_(ge("var(bucket_size_raid)", 1), le("var(raid_danger)", 0)), {"action": "set_unit_behavior", "in_bucket": "raid", "interrupt_current": True, "unit_when": NOT_IN_FIGHT})
rule({"building_count_at_least": {"amount": 1}},
     # soldiers that left the army or the raid come home, whatever they were still doing
     dict({"action": "move", "targets": "home", "together": False}, **DEF),
     dict({"action": "attack", "targets": "enemy_units", "target_unit_types": ["military"], "order": "space",
           "score": "1000 * min(1, var(base_spaces_within_0_of_target)) - 1000", "min_score": 0, "spread": DEF_SPREAD}, **DEF),
     dict({"action": "attack", "in_bucket": "raid", "targets": "enemy_units", "score": "10 - 10 * var(target_distance_home)", "min_score": 0, "spread": SPREAD, "eligible": SPREAD_ELIGIBLE, "unit_when": all_(AT_BASE, NOT_IN_FIGHT)}))

if int(P['est_inspect']):
    # a soldier stands on the edge zone facing a space of unidentified enemy units, which shows what is there next turn
    INSP = {"x": "var(insp_x)", "y": "var(insp_y)"}
    INSP_MOVE = {"action": "move", "targets": "closest_spaces", "from": INSP, "exclude_reference": True, "face": INSP, "score": "0 - var(target_distance)", "max": 1, "unit_types": ["military"]}
    mn, reach = P['est_inspect_min'], P['est_inspect_reach']
    count_here = "var(enemy_units_unidentified_within_0_of_target)"
    SAFE_TERM = " - 100 * var(enemy_units_visible_military_within_1_of_target)" if int(P['est_insp_safe']) else ""
    rule(ALWAYS, {"action": "select_space", "from": "home", "var_x": "insp_x", "var_y": "insp_y", "var_score": "insp_score",
                  "options": [{"targets": "unidentified_units", "score": f"{count_here} - {mn} + 1 - 100 * max(0, var(target_distance_home) - {reach}){SAFE_TERM}", "min_score": 1}]})
    rule(ALWAYS, setv("insp_cool", "max(0, var(insp_cool) - 1)", True))
    rule(all_(ge("var(insp_score)", 1), le("var(insp_cool)", 0)), dict(INSP_MOVE, exclude_buckets=True, eligible=["move"], where={"owner": "own"}, unit_when=NOT_IN_FIGHT),
         *([setv("insp_cool", P['est_insp_cool'], True)] if P['est_insp_cool'] > 0 else []))


# ================= local fights: every soldier sharing a space with enemy soldiers works script-assigned targets =================
# overrides earlier military orders; retreat rules follow
# the enemy soldiers in spaces where we have soldiers are shared out among ours there
SOLDIER_ORDERS = ["move", "attack_unit", "attack_building", "attack_space", "attack_zone"]
# aggressive for this fight only, and never hijacked by attack_back, so the script's spread of targets holds through the turn
rule(ALWAYS, {"action": "set_unit_behavior", "unit_types": ["military"], "stance": "aggressive", "attack_back": False, "interrupt_current": False, "unit_when": IN_FIGHT})
FOCUS = {0: "", 1: " + 4 * (1 - var(target_health) / max(1, var(target_max_health)))", 2: " - var(target_max_health) / 10", 3: " + var(target_max_health) / 10",
         4: " - 4 * (1 - var(target_health) / max(1, var(target_max_health)))"}[int(P['focus'])]
FIGHT_SCORE = "10 * min(1, var(population_military_within_0_of_target)) - var(target_distance) / 10" + FOCUS
SPILL = P['spill_vils'] or P['spill_bldgs'] or P['adj_hold']
FIGHT = dict(unit_types=["military"], eligible=SPREAD_ELIGIBLE, spread=dict(SPREAD, overflow="none" if SPILL else "round_robin"), unit_when=IN_FIGHT)
rule(ALWAYS, dict({"action": "attack", "targets": "enemy_units", "target_unit_types": ["military"], "score": FIGHT_SCORE, "min_score": 5}, **FIGHT))
if P['spill_vils']:
    rule(ALWAYS, dict({"action": "attack", "targets": "enemy_units", "target_unit_ids": [VIL], "score": FIGHT_SCORE, "min_score": 5}, **FIGHT))
if P['spill_bldgs']:
    rule(ALWAYS, dict({"action": "attack", "targets": "enemy_buildings", "score": FIGHT_SCORE, "min_score": 5}, **FIGHT))
if P['adj_hold']:  # leftover units take an adjacent space only when our surplus here, after holding adj_hold x the odds-adjusted need, beats the strength there
    SURPLUS_HERE = f"({pow_near('mine', 'unit', 0)} * var(myTech) - {P['adj_hold']} * {ao} * {pow_near('enemy', 'unit', 0)} * var(eTech))"
    NEED_THERE = f"{ao} * ({pow_near('enemy', 'target', 0)} * var(eTech) + 6 * var(enemy_buildings_known_{DEFENSIVE}_within_0_of_target))"
    rule(le("var(a_outnumbered)", 0), dict({"action": "attack", "targets": "enemy_units", "target_unit_types": ["military"], "order": "space",
         "score": f"12 - var(unit_distance) - 100 * max(0, {NEED_THERE} - {SURPLUS_HERE})", "min_score": 5.5}, **FIGHT))

if P['forward']:  # the army is outnumbered where it fights (the usual local calculation): the whole bucket falls back to the forward base
    AT_FB = {"population_at_least": {"unit_types": ["military"], "in_bucket": "army", "within": 0, "from": FB, "amount": "var(bucket_size_army)"}}
    GROUP_OUT = any_(ge("var(a_local)", 1), ge("var(group_out_sum)", 1)) if int(P['stage_adj']) else ge("var(a_local)", 1)
    rule(all_(FWD, le("var(a_outnumbered)", 0), GROUP_OUT, {"not": AT_FB}),
         {"action": "set_unit_behavior", "in_bucket": "army", "stance": "passive", "attack_back": False, "interrupt_current": False},
         {"action": "move", "in_bucket": "army", "eligible": SOLDIER_ORDERS, "targets": "coordinate", "coordinate": FB})
    # enemies next to the cleared forward base and not one army unit got an assignment: retreat like normal
    rule(all_(FWD, le("var(a_outnumbered)", 0), FB_CLEAR, ADJ_ENEMY, le("var(fwd_wait)", 0), ge("var(bucket_size_army)", 1), ge("var(idle_units_military_bucket_army) - var(bucket_size_army)", 0)),
         setv("a_outnumbered", 1), setv("assault", 0, True), setv("army_fwd", 0, True))
ARMY_PASSIVE = {"action": "set_unit_behavior", "in_bucket": "army", "stance": "passive", "attack_back": False, "interrupt_current": False}
rule(all_(ge("var(bucket_size_army)", 1), ge("var(a_outnumbered)", 1), HAS_BUILDINGS), ARMY_PASSIVE,
     {"action": "move", "in_bucket": "army", "eligible": SOLDIER_ORDERS, "targets": "retreat"})
rule(all_(ge("var(bucket_size_army)", 1), ge("var(a_outnumbered)", 1), {"not": HAS_BUILDINGS}), ARMY_PASSIVE,
     {"action": "move", "in_bucket": "army", "eligible": SOLDIER_ORDERS, "targets": "coordinate", "coordinate": LAUNCH})
# only the local test fired on a split army: units not in a fight gather at the army's own space, still fighting back
rule(all_(ge("var(bucket_size_army)", 1), le("var(a_outnumbered)", 0), ge("var(a_raw)", 1), ge("var(army_split)", 1), le("var(army_fwd)", 0)),
     dict({"action": "move", "targets": "closest_spaces", "from": "army", "where": {"unit_count_at_least": 1}, "unit_when": NOT_IN_FIGHT, **HELD}, **ARMY))
rule(all_(ge("var(bucket_size_raid)", 1), ge("var(raid_danger)", 1)),
     {"action": "set_unit_behavior", "in_bucket": "raid", "stance": "passive", "attack_back": False, "interrupt_current": False, "unit_when": AWAY_FROM_BASE},
     {"action": "move", "in_bucket": "raid", "eligible": SOLDIER_ORDERS, "targets": "retreat", "unit_when": AWAY_FROM_BASE})
rule(all_(ge("var(bucket_size_raid)", 1), ge("var(raid_danger)", 1)),
     {"action": "set_unit_behavior", "in_bucket": "raid", "stance": "defensive", "attack_back": True, "interrupt_current": False, "unit_when": all_(AT_BASE, NOT_IN_FIGHT)})

# ================= villagers under attack: every villager sharing a space with enemy soldiers fights or runs =================
# last, so it overrides whatever they were told this turn. They fight together only when they crush what is there:
# villagers at their worth (hardcoded from unit stats: ~0.6 of a blunt, ~1 with Farming) plus our soldiers there,
# against the enemy's soldiers within a space; otherwise all of them take cover, however few the enemy are.
VIL_ORDERS = ["gather", "build", "move", "repair", "renew", "attack_unit", "attack_space"]
# soldiers a space away reach them within the turn, so they count; attacking their space also takes on the enemy villagers
# there, who fight back (counted at a full villager's worth)
rule(ALWAYS, *(setv(f"{n}_with_pre", foundations_with_builders(b)) for n, (b, _) in TRACKED.items()))
rule(ALWAYS,
     {"action": "attack", "unit_ids": [VIL], "eligible": VIL_ORDERS, "targets": "enemy_units", "target_unit_types": ["military"],
      "score": "0 - var(target_distance)", "min_score": 0, "order": "space", "resume": True,
      "unit_when": all_(VIL_SOLDIERS_HERE, {"value_at_least": {"value": crush_margin, "amount": 0}})},
     {"action": "garrison", "unit_ids": [VIL], "eligible": VIL_ORDERS,
      "unit_when": all_(VIL_SOLDIERS_NEAR, {"value_at_most": {"value": crush_margin, "amount": -0.01}})},
     # out again once no enemy soldier is within a space of them
     {"action": "ungarrison", "unit_ids": [VIL], "unit_when": {"value_at_most": {"value": "var(enemy_units_visible_military_within_1_of_unit)", "amount": 0}}})

# ================= failed build attempts: builders the flee rule pulled off count; at the limit, cancel and rebuild away from the site
def failed_site(b, axis): return (f"({one(f'var(foundation_count_{b}_without_builders)')} * var(foundation_space_{axis}_{b}_without_builders)"
                                  f" + (1 - {one(f'var(foundation_count_{b}_without_builders)')}) * var(foundation_space_{axis}_{b}))")
rule(ALWAYS, *(a for n, (b, limit) in TRACKED.items() for a in (
     setv(f"{n}_with_post", foundations_with_builders(b), True),
     setv(f"{n}_tries", f"var({n}_tries) + max(0, var({n}_with_pre) - var({n}_with_post)) * {one(f'var(foundation_count_{b}) - var({n}_abandoned)')}", True),
     setv(f"{n}_giveup", one(f"var({n}_tries) - {limit} + 1")),
     setv(f"{n}_bad_x", f"var({n}_giveup) * {failed_site(b, 'x')} + (1 - var({n}_giveup)) * var({n}_bad_x)", True),
     setv(f"{n}_bad_y", f"var({n}_giveup) * {failed_site(b, 'y')} + (1 - var({n}_giveup)) * var({n}_bad_y)", True),
     setv(f"{n}_ban", f"max(var({n}_ban), var({n}_giveup))", True),
     # a foundation that cannot be cancelled is abandoned in place, so a replacement can still be ordered
     setv(f"{n}_abandoned", f"var({n}_abandoned) + var({n}_giveup)", True),
     setv(f"{n}_tries", f"var({n}_tries) * (1 - var({n}_giveup))", True))))
for n, (b, _) in TRACKED.items():
    fires = [f for f in BUILD_FIRES if f.rstrip("0123456789") == n]
    # a foundation ordered this very turn doesn't exist yet, and cancelling it would void the whole batch
    rule(all_(ge(f"var({n}_giveup)", 1), *(le(f"var(fire_{f})", 0) for f in fires)), {"action": "cancel_foundations", "building_ids": [b]})

with open(OUT, 'w') as f:
    f.write('{\n  "rules": [\n' + ',\n'.join('    ' + json.dumps(r) for r in rules) + '\n  ]\n}\n')
print(len(rules), 'rules ->', OUT)
