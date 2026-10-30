from __future__ import annotations

from typing import Any
from common import Record, aggregate, power, selection, store, variable, whole


def raiding_state() -> list[Record]:
    """Manages raiding party size, casualty tracking, and cooldown pauses."""
    survivors = aggregate("count", selection("units", dict(in_set="raiding_party")))
    return [
        variable("raid_survivors", survivors),
        variable("raid_lost", "max(0, var(raid_prev) - var(raid_survivors))"),
        variable("raid_wiped", "(var(raid_prev) > 0) * (var(raid_lost) > var(raid_prev) / 2)"),
        variable("raid_fails", "var(raid_fails) + var(raid_wiped)", persist=True),
        variable("raid_pause", "max(0, max(var(raid_pause) - 1, var(raid_wiped) * var(tune_raid_pause) * var(raid_fails)))", persist=True),
        variable("raid_size",
                 "min(var(tune_raid_max), max(var(tune_raid_min), var(own_101) * var(tune_raid_frac))) * "
                 "(var(own_101) >= var(tune_raid_min)) * (var(has_economy) > 0) * "
                 "(var(threat_home) <= var(guard_home)) * (var(raid_pause) == 0)"),
        store("raiding_party", selection("units", dict(
            owner="own", unit_ids=[101], not_in_set="scout_units"
        ), limit=whole("var(raid_size)"), rank=["0 - var(target.uid)"]), persist=True),
        variable("raid_prev", "var(n_raiding_party)", persist=True),
    ]


def raider_danger_lets(role: str = "actor") -> dict[str, Any]:
    """Evaluates military threats near a raiding party member."""
    return dict(
        danger_power=aggregate("sum", selection("units", dict(
            owner="enemy", unit_types=["military"],
            **{"if": f"distance({role}, item) <= 2"}
        )), power()),
        danger_unid=aggregate("sum", selection("spaces", dict(in_set="poor_vision", **{
            "if": f"distance({role}, item) <= 2"
        })), "var(item.unit_count)"),
    )


def raiding_tactics() -> list[Record]:
    """Issues tactical raid orders: threat avoidance, economy harassment, and raze."""
    danger_lets = raider_danger_lets("target")
    danger_condition = "(var(target.danger_power) + var(target.danger_unid) * var(unknown_unit_power) > 0)"
    danger_store = store("danger_raiders", selection("units", dict(
        in_set="raiding_party",
        **{"if": danger_condition}
    ), let=danger_lets))
    enemy_military = selection("units", dict(owner="enemy", unit_types=["military"]))
    raid_sets = [
        store("raid_mil_near2_spaces", selection("spaces", dict(near=dict(distance=2, of=enemy_military)))),
        store("raid_tower_spaces", selection("spaces", dict(near=dict(distance=0, of=selection("buildings", dict(owner="enemy", building_ids=[1, 21, 23])))))),
        store("raid_unid_near2_spaces", selection("spaces", dict(near=dict(distance=2, of=selection("spaces", dict(in_set="unidentified_spaces")))))),
        store("raid_producer_near2_spaces", selection("spaces", dict(near=dict(distance=2, of=selection("buildings", dict(owner="enemy", building_ids=[22, 23])))))),
        store("raid_safe_spaces", selection("spaces", {"if": "(inSet(raid_mil_near2_spaces, target) + inSet(raid_tower_spaces, target) + inSet(raid_unid_near2_spaces, target) + inSet(raid_producer_near2_spaces, target)) == 0"})),
    ]
    safe_target = dict(at=dict(in_set="raid_safe_spaces"))
    vils_here = aggregate("count", selection("units", dict(
        owner="enemy", unit_ids=[1],
        **{"if": "distance(target, item) == 0"}
    )))

    # 1. Threat Retreat: Flee toward home if enemy soldiers close in
    threatened_raiders = selection("units", dict(in_set="danger_raiders"))
    home_space = selection("spaces", dict(in_set="home"), limit=1)

    retreat_behavior = dict(
        action="select", actors=threatened_raiders,
        issue=[dict(set_behavior=dict(stance="passive", attack_back=False, interrupt_current=True))]
    )
    retreat_move = dict(action="select", actors=threatened_raiders, pick=home_space, issue=[dict(move={})])

    # 2. Defensive Stance for active, unthreatened raiders
    active_raiders = selection("units", dict(in_set="raiding_party", not_in_set="danger_raiders"))

    active_behavior = dict(
        action="select", actors=active_raiders,
        issue=[dict(set_behavior=dict(stance="defensive", attack_back=True, interrupt_current=False))]
    )

    # 3. Priority 1: Harass Villagers
    vil_safe = "(var(villager_power) * var(target.vils_here) <= var(n_raiding_party) / var(tune_raid_odds))"
    enemy_vils = selection("units", dict(
        owner="enemy", unit_ids=[1], **safe_target, **{"if": vil_safe}
    ), let=dict(vils_here=vils_here), rank=["0 - var(target.uid)"])
    harass_vils = dict(
        action="select", actors=active_raiders, pick=enemy_vils,
        match=dict(score="20 - distance(actor, target)"),
        issue=[dict(attack={})]
    )

    # 4. Priority 2: Raze undefended enemy buildings (for unassigned raiders)
    unassigned_raiders = selection("units", dict(
        in_set="raiding_party", not_in_set="danger_raiders",
        **{"if": "var(actor.order_kind) == -1"}
    ))

    enemy_bldgs = selection("buildings", dict(
        owner="enemy", **safe_target
    ), rank=["0 - var(target.uid)"])
    raze_bldgs = dict(
        action="select", actors=unassigned_raiders, pick=enemy_bldgs,
        match=dict(score="20 - distance(actor, target)"),
        issue=[dict(attack={})]
    )

    # 5. Priority 3: Scout / Hunt Resource Nodes near enemy base (for unassigned raiders)
    enemy_resources = selection("gatherables", dict(
        near=dict(distance=2, of=selection("buildings", dict(owner="enemy"))),
        **safe_target
    ), rank=["0 - var(target.uid)"])
    hunt_resources = dict(
        action="select", actors=unassigned_raiders, pick=enemy_resources,
        match=dict(score="10 - distance(actor, target)"),
        issue=[dict(move={})]
    )

    return [danger_store, *raid_sets, retreat_behavior, retreat_move, active_behavior, harass_vils, raze_bldgs, hunt_resources]
