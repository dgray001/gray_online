from __future__ import annotations

from typing import Any
from common import Record, aggregate, army_order_available, branch, power, selection, store, variable


def threat_home_rules() -> list[Record]:
    """Evaluates military threat and available guard strength at the home base."""
    enemy_near_home = aggregate("count", selection("units", dict(
        owner="enemy", unit_types=["military"],
        near=dict(distance=1, of=selection("spaces", dict(in_set="home")))
    )))
    unidentified_near_home = aggregate("sum", selection("spaces", dict(in_set="poor_vision",
        near=dict(distance=1, of=selection("spaces", dict(in_set="home")))
    )), "var(item.unit_count)")

    friendly_guard = aggregate("count", selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        near=dict(distance=1, of=selection("spaces", dict(in_set="home")))
    )))
    return [
        variable("enemy_near_home", enemy_near_home),
        variable("unidentified_near_home", unidentified_near_home),
        variable("threat_home", "var(enemy_near_home) + var(unidentified_near_home)"),
        variable("guard_home", friendly_guard),
    ]


def civilian_defense_lets() -> dict[str, Any]:
    """Computes spatial combat metrics for an individual villager actor."""
    return dict(
        enemy_space_count=aggregate("count", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(actor, item) == 0"}
        ))),
        enemy_near_power=aggregate("sum", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(actor, item) <= 1"}
        )), power()),
        enemy_near_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{
            "if": "distance(actor, item) <= 1"
        })), "var(item.unit_count)"),
        allies_military_power=aggregate("sum", selection("units", dict(
            owner="own", unit_types=["military"],
            **{"if": "distance(actor, item) == 0"}
        )), power()),
        allies_vils_count=aggregate("count", selection("units", dict(
            owner="own", unit_ids=[1],
            **{"if": "distance(actor, item) == 0"}
        ))),
    )


def civilian_crush_margin() -> str:
    """Calculates villager fight-or-flight margin against hostile forces."""
    allies = "(var(actor.allies_military_power) + var(villager_power) * var(actor.allies_vils_count))"
    threat = "(var(actor.enemy_near_power) + var(tune_vil_unknown) * var(unknown_unit_power) * var(actor.enemy_near_unid))"
    return f"({allies} - var(tune_vil_crush_odds) * {threat})"


def civilian_defense() -> list[Record]:
    """Civilian fight-or-flight micro and garrison shelter management."""
    lets = civilian_defense_lets()
    crush = civilian_crush_margin()

    # 1. Fight: If enemy soldiers share our space and local allies crush the hostile force
    fight_actors = selection("units", dict(
        owner="own", unit_ids=[1], not_in_set="evacuees",
        **{"if": f"(1 - var(actor.is_garrisoned)) * (var(actor.enemy_space_count) >= 1) * ({crush} >= 0)"}
    ), let=lets)
    fight_pick = selection("units", dict(
        owner="enemy", unit_types=["military"],
        near=dict(distance=0, of=selection("units", dict(in_set="villagers")))
    ), rank=["0 - var(target.health) / max(0.1, var(target.max_health))"],
       capacity="max(1, var(target.health) / var(villager_power))")
    fight_match = dict(where={"if": "distance(actor, target) == 0"})
    fight_action = dict(action="select", actors=fight_actors, pick=fight_pick, match=fight_match, issue=[dict(attack={})])

    # 2. Garrison: If enemy soldiers are near and crush margin is unfavorable
    threatened = "((var(actor.enemy_space_count) > 0) + (var(actor.enemy_near_power) > 0) + (var(actor.enemy_near_unid) > 0))"
    garrison_actors = selection("units", dict(
        owner="own", unit_ids=[1], not_in_set="evacuees",
        **{"if": f"(1 - var(actor.is_garrisoned)) * ({threatened} > 0) * ({crush} < 0)"}
    ), let=lets)
    pending_garrison = aggregate("count", selection("units", dict(
        owner="own", **{"in": dict(order_kind=["garrison"])},
        **{"if": "var(item.order_target_uid) == var(target.uid)"}
    )))
    garrison_pick = selection("buildings", dict(
        owner="own", state="complete",
        **{"if": "var(target.garrison_count) + var(target.pending) < var(target.garrison_capacity)"}
    ), let=dict(pending=pending_garrison),
       capacity="var(target.garrison_capacity) - var(target.garrison_count) - var(target.pending)")
    garrison_match = dict(score="0 - distance(actor, target)")
    garrison_action = dict(action="select", actors=garrison_actors, pick=garrison_pick, match=garrison_match, issue=[dict(garrison={})])

    # 3. Ungarrison: When no hostile military units remain within 1 space
    safe_lets = dict(
        enemy_near_count=aggregate("count", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(actor, item) <= 1"}
        ))),
        enemy_near_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{
            "if": "distance(actor, item) <= 1"
        })), "var(item.unit_count)"),
    )
    ungarrison_actors = selection("units", dict(
        owner="own", unit_ids=[1],
        **{"if": "var(actor.is_garrisoned) * (var(actor.enemy_near_count) == 0) * (var(actor.enemy_near_unid) == 0)"}
    ), let=safe_lets)
    ungarrison_action = dict(action="select", actors=ungarrison_actors, issue=[dict(ungarrison={})])

    threat_base_enemy = aggregate("count", selection("units", dict(
        owner="enemy", unit_types=["military"],
        near=dict(distance=1, of=selection("spaces", dict(in_set="base")))
    )))
    threat_base_unid = aggregate("sum", selection("spaces", dict(in_set="poor_vision",
        near=dict(distance=1, of=selection("spaces", dict(in_set="base")))
    )), "var(item.unit_count)")
    threat_vil_enemy = aggregate("count", selection("units", dict(
        owner="enemy", unit_types=["military"],
        near=dict(distance=1, of=selection("units", dict(in_set="villagers")))
    )))
    threat_vil_unid = aggregate("sum", selection("spaces", dict(in_set="poor_vision",
        near=dict(distance=1, of=selection("units", dict(in_set="villagers")))
    )), "var(item.unit_count)")
    garrisoned_vils = aggregate("count", selection("units", dict(
        owner="own", unit_ids=[1],
        **{"if": "var(item.is_garrisoned)"}
    )))

    return [
        variable("threat_base_enemy", threat_base_enemy),
        variable("threat_base_unid", threat_base_unid),
        variable("threat_base", "var(threat_base_enemy) + var(threat_base_unid)"),
        variable("threat_vil_enemy", threat_vil_enemy),
        variable("threat_vil_unid", threat_vil_unid),
        variable("threat_civilians", "var(threat_base) + var(threat_vil_enemy) + var(threat_vil_unid)"),
        variable("garrisoned_vils", garrisoned_vils),
        branch("(var(threat_civilians) > 0) + (var(garrisoned_vils) > 0) > 0", [
            fight_action, garrison_action, ungarrison_action
        ]),
    ]


def base_defense() -> list[Record]:
    """Defends base spaces with non-deployed military units."""
    defend_cond = "((var(assault) == 0) + (var(army_retreat) > 0) + (var(n_army_target_space) == 0) > 0) * (var(actor.e_here) == 0)"
    defend_lets = dict(e_here=aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(actor, item) == 0"}))))
    defenders = selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        **{"if": [defend_cond, army_order_available("actor")]}
    ), let=defend_lets)
    threats_at_base = selection("units", dict(
        owner="enemy", unit_types=["military"],
        at=dict(in_set="base")
    ))
    behavior = dict(
        action="select",
        actors=defenders,
        issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=False))]
    )
    defend_action = dict(
        action="select", actors=defenders, pick=threats_at_base,
        match=dict(score="0 - distance(actor, target)"),
        issue=[dict(attack={})]
    )

    idle_guards = selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        **{"if": ["var(actor.idle) * (var(actor.order_kind) == -1) * (var(actor.is_scout) == 0)", army_order_available("actor")]}
    ), let=dict(is_scout="inSet(scout_units, actor)"))
    home_space = selection("spaces", dict(in_set="home"), limit=1)
    hold_base = dict(action="select", actors=idle_guards, pick=home_space, issue=[dict(move={})])

    return [behavior, defend_action, hold_base]


def civilian_evacuation() -> list[Record]:
    """Identifies evacuating villagers, selects a safe refuge resource, and dispatches evacuees + 1 escort."""
    evac_lets = dict(
        e_pow=aggregate("sum", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(target, item) <= 1"})), power()),
        a_mil=aggregate("sum", selection("units", dict(owner="own", unit_types=["military"], **{"if": "distance(target, item) <= 1"})), power()),
        a_vils=aggregate("count", selection("units", dict(owner="own", unit_ids=[1], **{"if": "distance(target, item) == 0"}))),
    )
    support = "(var(villager_power) * var(target.a_vils) + var(target.a_mil))"
    is_evac = f"(1 - var(target.is_garrisoned)) * (var(target.e_pow) >= var(tune_evac_threat_min)) * (var(target.e_pow) > var(tune_evac_threat_ratio) * {support})"
    evacuees = selection("units", dict(owner="own", unit_ids=[1], **{"if": is_evac}), let=evac_lets)
    refuge_lets = dict(
        e_mil=aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(target, item) <= 2"}))),
        e_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{"if": "distance(target, item) <= 1"})), "var(item.unit_count)"),
        e_bldg=aggregate("count", selection("buildings", dict(owner="enemy", **{"if": "distance(target, item) <= 1"}))),
    )
    refuge_ok = "(var(target.category) <= 1) * (var(target.amount_left) > 0) * (var(target.e_mil) == 0) * (var(target.e_unid) == 0) * (var(target.e_bldg) == 0) * (25 - distance(home_origin, target) >= 1)"
    refuge = selection("resources", {"if": refuge_ok}, let=refuge_lets, rank=["25 - distance(home_origin, target)", "0 - var(target.uid)"], limit=1)
    pick_refuge = dict(action="for_each", select=selection("spaces", dict(in_set="home"), limit=1), **{"as": "home_origin"}, do=[
        store("refuge", refuge, persist=True),
        variable("has_refuge", "max(var(has_refuge), var(n_refuge) > 0)", persist=True),
    ])
    refuge_pick = selection("resources", dict(in_set="refuge"), limit=1)
    non_evac_vils = selection("units", dict(owner="own", unit_ids=[1], not_in_set="evacuees"))
    non_evac_behavior = dict(action="select", actors=non_evac_vils, issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=True))])
    evac_actors = selection("units", dict(in_set="evacuees"))
    evac_behavior = dict(action="select", actors=evac_actors, issue=[dict(set_behavior=dict(stance="passive", attack_back=False, interrupt_current=False))])
    evac_move = dict(action="select", actors=evac_actors, pick=refuge_pick, issue=[dict(move=dict(to="space"))])
    escort_actors = selection("units", dict(
        owner="own", unit_types=["military"], **{"if": "var(actor.e_here) == 0"}
    ), let=dict(e_here=aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(actor, item) == 0"})))), rank=["0 - var(actor.uid)"], limit=1)
    escort_move = dict(action="select", actors=escort_actors, pick=refuge_pick, issue=[dict(move=dict(to="space"))])
    return [
        store("evacuees", evacuees),
        non_evac_behavior,
        branch("var(n_evacuees) >= 1", [pick_refuge]),
        branch("(var(has_refuge) >= 1) * (var(n_evacuees) >= 1)", [evac_behavior, evac_move]),
        branch("(var(has_refuge) >= 1) * ((var(n_evacuees) >= 1) + ((var(complete_1) == 0) * (var(had_city) >= 1)) > 0)", [escort_move]),
    ]
