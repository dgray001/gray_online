from __future__ import annotations

from common import Record, aggregate, branch, selection, store, variable
from locations import space_key

QUIET = "1 - inSet(enemy_mil_near_spaces, target)"


def nearest(space_set: str) -> Record:
    distance = aggregate("min", selection("spaces", dict(in_set=space_set)), "distance(target, item)")
    distance["min"]["default"] = 999
    return distance


def holders() -> Record:
    return selection("units", dict(owner="own", bucket="holder"))


def release_holders() -> list[Record]:
    release_all = dict(action="select", pick=holders(), into=dict(bucket="holder", mode="remove"))
    near_enemy = dict(action="select", pick=selection("units", dict(owner="own", bucket="holder", at=dict(in_set="enemy_mil_near_spaces"))),
                      into=dict(bucket="holder", mode="remove"))
    return [branch("var(has_economy) > 0", [release_all], otherwise=[near_enemy])]


def holder_targets() -> list[Record]:
    first = "var(tune_hold_near_first) * (0 - var(target.to_home)) + (1 - var(tune_hold_near_first)) * var(target.gold_income)"
    sites = selection("spaces", dict(in_set="hold_pool", owner="unowned", vision_at_least=1,
                                     **{"if": ["var(target.passable) > 0", "var(target.gold_income) > 0", QUIET]}),
                      let=dict(to_home=nearest("home")), rank=[first, "0 - var(target.to_home)", f"0 - {space_key('target')}"], limit="var(tune_hold_sites)")
    keep = selection("spaces", dict(in_set="owned_spaces", not_in_set="own_building_spaces", **{"if": QUIET}))
    targets = selection("spaces", dict(in_set="hold_pool", **{"if": "inSet(hold_keep, target) + inSet(hold_sites, target) > 0"}))
    held_here = "(var(item.x) == var(target.x)) * (var(item.y) == var(target.y))"
    heading_here = "(var(item.order_kind) == 0) * (var(item.order_space_x) == var(target.x)) * (var(item.order_space_y) == var(target.y))"
    covered = aggregate("count", selection("units", dict(owner="own", bucket="holder", **{"if": f"({held_here}) + ({heading_here}) > 0"})))
    uncovered = selection("spaces", dict(in_set="hold_targets", **{"if": "var(target.covered) == 0"}), let=dict(covered=covered))
    return [store("own_building_spaces", selection("spaces", dict(in_set="owned_spaces", building_owner="own"))),
            store("hold_keep", keep), store("hold_sites", sites), store("hold_targets", targets), store("uncovered_targets", uncovered)]


def holder_demand() -> list[Record]:
    first = "var(tune_hold_near_first) * (0 - var(target.to_home)) + (1 - var(tune_hold_near_first)) * var(target.gold_income)"
    priority = ["inSet(hold_keep, target) * inSet(base, target)", "inSet(hold_keep, target)", first, "0 - var(target.to_home)", f"0 - {space_key('target')}"]
    capacity = "floor(var(tune_hold_cap) * var(hold_military)) - var(hold_holders)"
    need = f"min(var(n_uncovered_targets), max(0, {capacity})) * (var(hold_military) >= var(tune_hold_min_army))"
    wanted = selection("spaces", dict(in_set="uncovered_targets"), let=dict(to_home=nearest("home")), rank=priority, limit="var(hold_need)")
    recruits = selection("units", dict(owner="own", unit_types=["military"], not_in_set="raiding_party",
                                       **{"if": ["1 - inBucket(holder, target)", "1 - inSet(scout_units, target)"]}),
                         let=dict(nearest=nearest("hold_wanted")), rank=["0 - var(target.nearest)", "0 - var(target.uid)"], limit="var(hold_need)")
    return [variable("hold_military", aggregate("count", selection("units", dict(owner="own", unit_types=["military"])))),
            variable("hold_holders", aggregate("count", holders())),
            variable("hold_need", need), store("hold_wanted", wanted), store("hold_recruits", recruits)]


def holder_observations() -> list[Record]:
    pool = dict(near=dict(distance=1, of=selection("spaces", dict(in_set="owned_spaces"))))
    claimed = selection("units", dict(owner="own", **{"if": "inBucket(holder, target) + inSet(hold_recruits, target) > 0"}))
    reserved = selection("units", dict(owner="own", **{"if": "inBucket(scout, target) + inBucket(holder, target) + inSet(hold_recruits, target) > 0"}))
    holding = [store("hold_pool", selection("spaces", pool)), *holder_targets(), *holder_demand()]
    return [store("owned_spaces", selection("spaces", dict(owner="own"))), branch("(var(has_economy) == 0) * (var(has_production) > 0)", holding),
            store("claimed_units", claimed),
            store("holder_spaces", selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(in_set="claimed_units")))))),
            store("reserved_units", reserved)]


def holder_orders() -> list[Record]:
    add = dict(action="select", pick=selection("units", dict(in_set="hold_recruits")), into=dict(bucket="holder", mode="add"))
    go = dict(action="select", actors=selection("units", dict(in_set="hold_recruits")), pick=selection("spaces", dict(in_set="hold_wanted"), capacity="1"),
              match=dict(score="0 - distance(actor, target)"), issue=[dict(move={})])
    return [branch("(var(has_economy) == 0) * (var(has_production) > 0)", [add, go])]
