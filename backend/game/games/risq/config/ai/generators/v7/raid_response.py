from __future__ import annotations

from common import Record, aggregate, branch, power, rule, selection, store
from policy_common import insert_before, unidentified


def responders(role: str, anchor: str, limit: str) -> Record:
    available = f"(var({role}.unit_id) != 1) + inSet(economy_workers, {role}) > 0"
    defensive = f"(1 - var(tune_refine_defensive_only)) + (1 - inBucket(army, {role})) + ((var(assault) == 0) * (var(army_retreat) == 0)) > 0"
    return selection("units", dict(in_set="own_units", unit_ids=[1, 101, 102, 103], **{"if": [
        f"1 - var({role}.is_garrisoned)", f"var({role}.hp_pct) >= var(tune_refine_health_floor)", available, defensive,
        f"1 - inSet(evacuees, {role})", f"1 - inSet(raiding_party, {role})", f"1 - inSet(reserved_units, {role})",
        f"1 - inSet(refine_used, {role})", f"distance({role}, {anchor}) <= (var({role}.unit_id) != 1) * var(tune_refine_local_radius)"]}),
        rank=[f"var({role}.unit_id) != 1", f"{power(role)}", f"0 - distance({role}, {anchor})", f"0 - var({role}.uid)"], limit=limit)


def raid_lets(role: str) -> Record:
    enemy = selection("units", dict(in_set="enemy_units", unit_types=["military"], **{"if": f"distance({role}, item) == 0"}))
    limit = f"min(var(tune_policy_responders), max(1, ceil(var({role}.refine_count) * var(tune_refine_response_scale))))"
    allies = responders("item", role, f"var({role}.refine_limit)")
    strength = f"(({power()}) + var(villager_power) * (var(item.unit_id) == 1)) * var(item.hp_pct)"
    return dict(refine_count=aggregate("count", enemy), refine_enemy=aggregate("sum", enemy, power()),
                refine_unknown=unidentified(role), refine_limit=limit, refine_own=aggregate("sum", allies, strength))


def leash(model: Record) -> None:
    nearby = aggregate("count", selection("units", dict(in_set="enemy_units", unit_types=["military"],
        **{"if": "distance(actor, item) == 0"})))
    actors = selection("units", dict(in_set="own_units", unit_ids=[1, 101, 102, 103], **{"if": [
        "var(actor.refine_here) == 0", "1 - inSet(raiding_party, actor)", "1 - inSet(reserved_units, actor)",
        "(var(actor.unit_id) == 1) + ((var(assault) == 0) * (var(army_retreat) == 0)) > 0"]}), let=dict(refine_here=nearby))
    stop = dict(action="select", actors=actors, issue=[dict(stop=dict(orders=["attack_unit"]))])
    insert_before(model, "economy_maturity", [stop])


def bounded_raids(model: Record, settings: dict[str, float]) -> None:
    if settings["refine_worker_leash"]:
        leash(model)
    safe = "(var(target.refine_count) > 0) * (var(target.refine_count) <= var(tune_policy_small_raid)) * (var(target.refine_unknown) == 0) * (var(target.refine_own) >= var(tune_refine_raid_margin) * var(target.refine_enemy))"
    raids = selection("spaces", dict(in_set="enemy_mil_spaces", **{"if": safe}), let=raid_lets("target"))
    observe = [store("refine_used", selection("units", dict(in_set="own_units", **{"if": "0"}))), store("refine_raids", raids)]
    model["rules"].append(rule(observe))
    current = store("refine_current", responders("target", "refine_raid", "var(refine_raid.refine_limit)"))
    actors = selection("units", dict(in_set="refine_current"))
    enemies = selection("units", dict(in_set="enemy_units", unit_types=["military"], **{"if": "distance(target, refine_raid) == 0"}),
        rank=["0 - var(target.health)", "0 - var(target.uid)"], capacity="max(1, var(target.health) / var(effective_hit))")
    attack = dict(action="select", actors=actors, pick=enemies,
        match=dict(where={"if": "var(actor.order_target_uid) != var(target.uid)"}), issue=[dict(attack={})],
        mode="prepend" if settings["refine_preserve_work"] else "replace")
    used = store("refine_used", selection("units", dict(in_set="own_units", **{
        "if": "inSet(refine_used, target) + inSet(refine_current, target) > 0"})))
    sufficient = "var(refine_raid.refine_own) >= var(tune_refine_raid_margin) * var(refine_raid.refine_enemy)"
    source = selection("spaces", dict(in_set="refine_raids"), let=raid_lets("refine_raid"))
    model["rules"].append(rule([dict(action="for_each", select=source, **{"as": "refine_raid"},
        do=[branch(sufficient, [current, attack, used])])]))
