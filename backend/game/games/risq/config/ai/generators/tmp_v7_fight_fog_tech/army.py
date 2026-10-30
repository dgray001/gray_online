from __future__ import annotations

from typing import Any
from common import Record, aggregate, branch, power, selection, store, variable, whole
from locations import space_key


def assault_odds_rules() -> list[Record]:
    """Calculates dynamic assault odds and launch readiness."""
    eco_assault = "var(tune_assault_odds) * (var(has_economy) > 0) + var(tune_assault_odds_noeco) * (var(has_economy) == 0)"
    eco_power = "var(tune_power_odds) * (var(has_economy) > 0) + var(tune_power_odds_noeco) * (var(has_economy) == 0)"
    eco_cap = "var(tune_cap_odds) * (var(has_economy) > 0) + var(tune_cap_odds_noeco) * (var(has_economy) == 0)"

    ramp = "min(1, max(0, (var(own_power) - 30) / 30))"
    cap_fill = "min(1, max(0, (gameVar(population) - var(tune_cap_ramp) * gameVar(population_limit)) / max(0.01, (1 - var(tune_cap_ramp)) * gameVar(population_limit))))"

    home_mil = aggregate("sum", selection("units", dict(
        owner="own", unit_types=["military"],
        near=dict(distance=1, of=selection("spaces", dict(in_set="home")))
    )), power())

    enemy_within_2 = aggregate("count", selection("units", dict(
        owner="enemy", unit_types=["military"],
        near=dict(distance=2, of=selection("spaces", dict(in_set="army_space")))
    )))
    unid_within_2 = aggregate("sum", selection("spaces", dict(in_set="poor_vision",
        near=dict(distance=2, of=selection("spaces", dict(in_set="army_space")))
    )), "var(item.unit_count)")
    skip_gate = "(var(enemy_strength) > 0) * (var(army_size) > 0) * (var(remembered_military) > 0)"

    return [
        variable("home_military_power", home_mil),
        variable("ao_base", f"({eco_assault}) - (({eco_assault}) - ({eco_power})) * ({ramp})"),
        variable("ao", f"min(var(ao_base), ({eco_assault}) - (({eco_assault}) - ({eco_cap})) * ({cap_fill}))"),
        variable("go", "(var(home_military_power) >= var(ao) * var(enemy_strength)) * (var(home_military_power) >= var(tune_assault_min))"),
        variable("army_enemies_within_2", enemy_within_2),
        variable("army_unid_within_2", unid_within_2),
        variable("army_enemies_within_2", "var(army_enemies_within_2) + var(army_unid_within_2)"),
        variable("skip_rest", f"(var(own_strategic_power) >= var(enemy_strength) * (var(ao) - var(tune_retreat_margin) + var(tune_skip_rest_margin))) * {skip_gate}"),
        variable("skip_stage", f"(var(own_strategic_power) >= var(enemy_strength) * (var(ao) - var(tune_retreat_margin) + var(tune_skip_stage_margin))) * {skip_gate}"),
    ]


def army_formation_rules() -> list[Record]:
    """Assigns all non-raiding, non-scout military units to the main army bucket."""
    assign_army = dict(
        action="select",
        pick=selection("units", dict(
            owner="own", unit_types=["military"],
            not_in_set="raiding_party",
            **{"if": "var(target.is_scout) == 0"}
        ), let=dict(is_scout="inSet(scout_units, target)"), rank=["0 - var(target.uid)"]),
        into=dict(bucket="army", mode="replace", var="army_size")
    )
    return [assign_army]


def army_target_selection() -> list[Record]:
    """Selects and prioritizes the strategic enemy target space."""
    target_lets = dict(
        enemy_mil_power=aggregate("sum", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(target, item) == 0"}
        )), power()),
        enemy_vils_count=aggregate("count", selection("units", dict(
            owner="enemy", unit_ids=[1],
            **{"if": "distance(target, item) == 0"}
        ))),
        strategic_bldgs=aggregate("count", selection("buildings", dict(
            owner="enemy", building_ids=[1, 21, 22, 23],
            **{"if": "distance(target, item) == 0"}
        ))),
        enemy_bldgs=aggregate("count", selection("buildings", dict(
            owner="enemy",
            **{"if": "distance(target, item) == 0"}
        ))),
        e_mil_near=aggregate("sum", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(target, item) <= 1"}
        )), power()),
        e_unid_near=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{
            "if": "distance(target, item) <= 1"
        })), "var(item.unit_count)"),
        e_towers_here=aggregate("count", selection("buildings", dict(
            owner="enemy", building_ids=[1, 21, 23],
            **{"if": "distance(target, item) == 0"}
        ))),
    )
    base_prio = (
        "max(40 * (var(target.enemy_mil_power) > 0), "
        "35 * (var(target.strategic_bldgs) > 0), "
        "30 * (var(target.enemy_bldgs) > 0), "
        "25 * (var(target.enemy_vils_count) > 0), "
        "20 * (max(0, var(target.unit_count)) > 0))"
    )
    target_threat = "((var(target.e_mil_near) + var(target.e_unid_near) * var(unknown_unit_power)) * var(enemy_tech_multiplier) + 3 * var(target.e_towers_here))"
    target_score = f"({base_prio}) - distance(army_origin, target)"
    has_enemy = "((var(target.enemy_mil_power) > 0) + (var(target.enemy_bldgs) > 0) + (var(target.enemy_vils_count) > 0) + (max(0, var(target.unit_count)) > 0) > 0)"
    target_space_picker = selection("spaces", dict(in_set="enemy_presence", **{
        "if": f"{has_enemy} * (1.2 * {target_threat} <= var(army_tactical_power)) * (({target_score}) >= 1)"
    }), let=target_lets, rank=[target_score, f"0 - {space_key('target')}"], limit=1)
    pick_from_army = dict(action="for_each", select=selection("spaces", dict(in_set="army_space"), limit=1), **{"as": "army_origin"}, do=[
        store("army_target_space", target_space_picker),
    ])
    changed = "(var(army_target_x) != var(prev_target_x)) + (var(army_target_y) != var(prev_target_y)) + (var(army_engaged) > 0) > 0"

    return [
        variable("prev_target_x", "var(army_target_x)"),
        variable("prev_target_y", "var(army_target_y)"),
        store("army_target_space", selection("spaces", {"if": "0"})),
        pick_from_army,
        dict(action="for_each", select=selection("spaces", dict(in_set="army_target_space")), **{"as": "tgt"}, do=[
            variable("army_target_x", "var(tgt.x)", persist=True),
            variable("army_target_y", "var(tgt.y)", persist=True),
            variable("has_army_target", 1, persist=True),
        ]),
        branch("var(n_army_target_space) == 0", [variable("has_army_target", 0, persist=True)]),
        branch(changed, [variable("army_stage", 0, persist=True), variable("army_rest", 0, persist=True)]),
    ]


def army_local_odds_rules() -> list[Record]:
    """Calculates local combat engagement odds and retreat conditions."""
    local_lets = dict(
        e_mil=aggregate("sum", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(center_unit, item) <= 1"}
        )), power()),
        e_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{
            "if": "distance(center_unit, item) <= 1"
        })), "var(item.unit_count)"),
        e_towers=aggregate("count", selection("buildings", dict(
            owner="enemy", building_ids=[1, 21, 23],
            **{"if": "distance(center_unit, item) == 0"}
        ))),
        own_towers=aggregate("count", selection("buildings", dict(
            owner="own", building_ids=[1, 21, 23],
            **{"if": "distance(center_unit, item) == 0"}
        ))),
        own_vils=aggregate("count", selection("units", dict(
            owner="own", unit_ids=[1],
            **{"if": "distance(center_unit, item) == 0"}
        ))),
        e_engaged=aggregate("count", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": "distance(center_unit, item) == 0"}
        ))),
    )
    e_threat = (
        "max(0, (var(center_unit.e_mil) + "
        "var(center_unit.e_unid) * var(unknown_unit_power)) * var(enemy_tech_multiplier) + "
        "3 * var(center_unit.e_towers) - 3 * var(center_unit.own_towers) - "
        "var(villager_power) * var(center_unit.own_vils))"
    )
    retreat_odds = (
        "(var(center_unit.e_engaged) > 0) * (var(tune_engaged_odds) * (var(has_economy) > 0) + var(tune_engaged_odds_noeco) * (var(has_economy) == 0)) + "
        "(var(center_unit.e_engaged) == 0) * max(0.5, var(ao) - var(tune_retreat_margin))"
    )
    outnumbered = f"(var(e_threat_local) * ({retreat_odds}) > var(army_tactical_power)) * (var(army_size) > 0)"

    escape_lets = dict(
        d_mil=aggregate("sum", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(target, item) <= 1"})), power()),
        d_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{"if": "distance(target, item) <= 1"})), "var(item.unit_count)"),
        d_towers=aggregate("count", selection("buildings", dict(owner="enemy", building_ids=[1, 21, 23], **{"if": "distance(target, item) == 0"}))),
    )
    dest_threat = "((var(target.d_mil) + var(target.d_unid) * var(unknown_unit_power)) * var(enemy_tech_multiplier) + 3 * var(target.d_towers))"
    escape_ok = f"(distance(army_origin, target) > 0) * ((var(target.own_buildings) > 0) + ((var(target.vision) >= 3) * (distance(army_origin, target) <= 2)) > 0) * (({dest_threat} == 0) + ({dest_threat} + 2 < var(e_threat_local)) > 0)"
    escape_picker = selection("spaces", dict(
        passable=True, near=dict(distance=3, of=selection("spaces", dict(in_set="army_space"))), **{"if": escape_ok}
    ), let=escape_lets, rank=["var(target.own_buildings) > 0", f"0 - ({dest_threat})", "0 - distance(army_origin, target)", f"0 - {space_key('target')}"], limit=1)
    pick_escape = dict(action="for_each", select=selection("spaces", dict(in_set="army_space"), limit=1), **{"as": "army_origin"}, do=[
        store("escape_space", escape_picker),
    ])

    return [
        variable("e_threat_local", 0),
        variable("a_outnumbered", 0),
        dict(action="for_each", select=selection("units", dict(in_set="army_center"), let=local_lets, limit=1), **{"as": "center_unit"}, do=[
            variable("e_threat_local", e_threat),
            variable("a_outnumbered", outnumbered),
        ]),
        store("escape_space", selection("spaces", {"if": "0"})),
        pick_escape,
        variable("army_retreat", "(var(n_escape_space) > 0) * max(var(a_outnumbered), var(army_retreat) * (var(army_enemies_within_2) > 0))", persist=True),
        variable("assault", "min(1, max(var(assault) * (1 - var(a_outnumbered)) * (var(army_size) >= 3), var(go))) * (1 - var(army_retreat))", persist=True),
        branch("var(army_retreat) > 0", [variable("army_stage", 0, persist=True), variable("army_rest", 0, persist=True)]),
    ]


def army_maneuver_rules() -> list[Record]:
    """Executes army movement, grouping, staging, resting, and assault advance."""
    home_space = selection("spaces", dict(in_set="home"), limit=1)
    escape_space = selection("spaces", dict(in_set="escape_space"), limit=1)
    army_at_enemy = selection("units", dict(owner="own", bucket="army", at=dict(in_set="enemy_mil_spaces")))
    army_clear = store("army_clear", selection("units", dict(owner="own", bucket="army", not_in_set="army_at_enemy")))

    # 1. Retreat: Move to escape_space in defensive stance without interrupting attacks
    retreat_actors = selection("units", dict(owner="own", bucket="army", **{"if": "var(army_retreat) > 0"}))
    retreat_behavior = dict(
        action="select", actors=retreat_actors,
        issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=False))]
    )
    retreat_move = dict(action="select", actors=retreat_actors, pick=escape_space, issue=[dict(move={})])

    # 1b. Hold Home when not launching or no target exists
    hold_cond = "(var(army_retreat) == 0) * ((var(assault) == 0) + (var(n_army_target_space) == 0) > 0)"
    hold_actors = selection("units", dict(in_set="army_clear", **{"if": hold_cond}))
    hold_behavior = dict(action="select", actors=hold_actors, issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=False))])
    hold_move = dict(action="select", actors=hold_actors, pick=home_space, issue=[dict(move={})])

    # 2. Advance / Staging / Resting: Regroup when spread, stage & rest near target, advance when ready
    center_space = selection("spaces", dict(in_set="army_space"), limit=1)
    target_space = selection("spaces", dict(in_set="army_target_space"), limit=1)
    stage_zones = selection("zones", dict(
        near=dict(distance=1, of=selection("spaces", dict(in_set="army_target_space"))),
        **{"if": "distance(tgt_space, target) == 1"}
    ), rank=["0 - distance(aspace, target)", "0 - distance(tgt_space, target, 1)", f"0 - {space_key('target')}"], limit=1)
    measure_dist = dict(action="for_each", select=center_space, **{"as": "aspace"}, do=[
        dict(action="for_each", select=target_space, **{"as": "tgt_space"}, do=[
            variable("army_dist_to_target", "distance(aspace, tgt_space)"),
            store("stage_zone", stage_zones),
        ]),
    ])
    army_in_center = aggregate("count", selection("units", dict(
        owner="own", bucket="army", at=dict(in_set="army_space")
    )))
    is_grouped = "(var(army_in_center) >= var(tune_group_share) * var(army_size))"
    active = "(var(army_retreat) == 0) * (var(assault) > 0) * (var(n_army_target_space) > 0) * (var(army_engaged) == 0)"
    in_stage_band = f"({active}) * ({is_grouped}) * (var(army_dist_to_target) <= var(tune_stage_dist)) * (var(army_dist_to_target) > var(tune_rest_dist)) * (1 - var(skip_stage))"
    update_stage = variable("army_stage", f"(var(army_stage) == 1) * 2 + (var(army_stage) != 1) * max(var(army_stage), {in_stage_band})", persist=True)
    at_rest = "(var(army_dist_to_target) <= var(tune_rest_dist)) * (var(army_dist_to_target) > 0)"
    update_rest = variable("army_rest", f"(var(army_rest) + 1) * ({at_rest})", persist=True)
    preparing = f"((var(army_stage) == 1) + (({at_rest}) * (var(army_rest) <= var(tune_rest_turns)) * (1 - var(skip_rest))) > 0)"

    regroup_actors = selection("units", dict(in_set="army_clear", **{"if": f"({active}) * (1 - {is_grouped})"}))
    regroup_move = dict(action="select", actors=regroup_actors, pick=center_space, issue=[dict(move={})])
    prep_actors = selection("units", dict(owner="own", bucket="army", **{"if": f"({active}) * ({is_grouped}) * ({preparing})"}))
    prep_stop = dict(action="select", actors=prep_actors, issue=[dict(stop={}), dict(set_behavior=dict(stance="passive", attack_back=True, interrupt_current=False))])
    stage_actors = selection("units", dict(in_set="army_clear", **{"if": f"({active}) * ({is_grouped}) * (var(army_stage) == 2) * (var(army_dist_to_target) > var(tune_rest_dist))"}))
    stage_move = dict(action="select", actors=stage_actors, pick=selection("zones", dict(in_set="stage_zone"), limit=1), issue=[dict(move=dict(to="zone"))])
    can_advance = f"({active}) * ({is_grouped}) * (1 - {preparing}) * ((var(skip_stage) > 0) + (var(army_stage) == 0) + ({at_rest}) > 0)"
    advance_actors = selection("units", dict(in_set="army_clear", **{"if": can_advance}))
    advance_behavior = dict(action="select", actors=advance_actors, issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=False))])
    advance_move = dict(action="select", actors=advance_actors, pick=target_space, issue=[dict(move={})])

    return [store("army_at_enemy", army_at_enemy), army_clear, retreat_behavior, retreat_move, hold_behavior, hold_move,
            variable("army_dist_to_target", 999), variable("army_in_center", army_in_center), store("stage_zone", selection("zones", {"if": "0"})),
            measure_dist, update_stage, update_rest, regroup_move, prep_stop, stage_move, advance_behavior, advance_move]
