from __future__ import annotations

from typing import Any
from common import Record, aggregate, power, selection, store, variable
from retreat_policy import micro_available


def combat_micro_rules() -> list[Record]:
    """Tactical combat micro: dynamic stance management, target spreading, focus fire, and spillover."""
    enemy_mil_spaces = selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(owner="enemy", unit_types=["military"])))))
    enemy_vil_spaces = selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(owner="enemy", unit_ids=[1])))))
    enemy_bldg_spaces = selection("spaces", dict(building_owner="enemy"))
    vil_combat_spaces = selection("spaces", {"if": "(inSet(enemy_vil_spaces, target) * (1 - inSet(enemy_mil_spaces, target))) > 0"})
    bldg_combat_spaces = selection("spaces", {"if": "(inSet(enemy_bldg_spaces, target) * (1 - inSet(enemy_mil_spaces, target)) * (1 - inSet(enemy_vil_spaces, target))) > 0"})

    # 1. Own military in melee combat with enemy military (excluding raiders)
    mil_combatants_source = selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        at=dict(in_set="enemy_mil_spaces"), **{"if": micro_available("target")}
    ))

    combat_spaces = selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(in_set="combatants")))))

    mil_combatants = selection("units", dict(in_set="combatants"))
    infantry_combatants = selection("units", dict(in_set="combatants", unit_ids=[101, 102, 103]))

    combat_stance = dict(
        action="select", actors=infantry_combatants,
        issue=[dict(set_behavior=dict(stance="aggressive", attack_back=False, interrupt_current=False))]
    )

    enemy_mil = selection("units", dict(
        owner="enemy", unit_types=["military"],
        at=dict(in_set="combat_spaces")
    ), capacity="max(1, var(target.health) / (var(effective_hit) * var(tune_kill_ticks)))")
    enemy_mil_uncapped = selection("units", dict(
        owner="enemy", unit_types=["military"],
        at=dict(in_set="combat_spaces")
    ))
    assigned_combatants = selection("units", dict(in_set="combatants", **{"in": dict(order_kind=["attack_unit"])}))
    overflow_combatants = selection("units", dict(in_set="combatants", **{"if": "1 - inSet(assigned_combatants, actor)"}))

    focus_score = (
        "10 - distance(actor, target) + "
        "var(tune_focus) * (1 - var(target.health) / max(0.1, var(target.max_health)))"
    )
    engage_mil = dict(
        action="select", actors=mil_combatants, pick=enemy_mil,
        match=dict(where={"if": "distance(actor, target) == 0"}, score=focus_score),
        issue=[dict(attack={})]
    )
    engage_mil_overflow = dict(
        action="select", actors=overflow_combatants, pick=enemy_mil_uncapped,
        match=dict(where={"if": "distance(actor, target) == 0"}, score=focus_score),
        issue=[dict(attack={})]
    )

    # 2. Spillover against enemy villagers in same space
    vil_combatants_source = selection("units", dict(
        owner="own", unit_types=["military"], not_in_set="raiding_party",
        at=dict(in_set="vil_combat_spaces"), **{"if": micro_available("target")}
    ))

    vil_spaces = selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(in_set="vil_combatants")))))

    vil_combatants = selection("units", dict(in_set="vil_combatants"))

    enemy_vils = selection("units", dict(
        owner="enemy", unit_ids=[1],
        at=dict(in_set="vil_spaces")
    ), capacity="max(1, var(target.health) / (var(effective_hit) * var(tune_kill_ticks)))")

    engage_vils = dict(
        action="select",
        actors=vil_combatants,
        pick=enemy_vils,
        match=dict(
            where={"if": "distance(actor, target) == 0"},
            score=focus_score
        ),
        issue=[dict(attack={})]
    )

    # 3. Spillover against enemy buildings in same space
    bldg_combatants_source = selection("units", dict(
        owner="own", unit_ids=[101, 102, 103], not_in_set="raiding_party",
        at=dict(in_set="bldg_combat_spaces"), **{"if": micro_available("target")}
    ))

    bldg_spaces = selection("spaces", dict(near=dict(distance=0, of=selection("units", dict(in_set="bldg_combatants")))))

    bldg_combatants = selection("units", dict(in_set="bldg_combatants"))

    enemy_bldgs = selection("buildings", dict(
        owner="enemy",
        at=dict(in_set="bldg_spaces")
    ))
    strat_bldg = "((var(target.building_id) == 1) + (var(target.building_id) == 21) + (var(target.building_id) == 22) + (var(target.building_id) == 23))"

    engage_bldgs = dict(
        action="select",
        actors=bldg_combatants,
        pick=enemy_bldgs,
        match=dict(
            where={"if": "distance(actor, target) == 0"},
            score=f"3 * {strat_bldg} + {focus_score}"
        ),
        issue=[dict(attack={})]
    )

    catapult_stance = dict(
        action="select", actors=selection("units", dict(owner="own", unit_ids=[401], **{"if": micro_available("actor")})),
        issue=[dict(set_behavior=dict(stance="stand_ground", attack_back=True, interrupt_current=False))]
    )
    catapult_lets = dict(
        enemy_mil_here=aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"], **{"if": "distance(actor, item) == 0"}))),
        enemy_bldgs_here=aggregate("count", selection("buildings", dict(owner="enemy", **{"if": "distance(actor, item) == 0"}))),
        escort_inf=aggregate("count", selection("units", dict(owner="own", unit_ids=[101, 102, 103], **{"if": "distance(actor, item) <= 1"}))),
    )
    ready_catapults = selection("units", dict(
        owner="own", unit_ids=[401],
        **{"if": [micro_available("actor"), "(var(actor.enemy_mil_here) == 0) * ((var(actor.escort_inf) >= 4) + (var(actor.enemy_bldgs_here) > 0) > 0)"]}
    ), let=catapult_lets)
    catapult_bldgs = selection("buildings", dict(owner="enemy", near=dict(distance=1, of=selection("units", dict(owner="own", unit_ids=[401])))))
    engage_catapults = dict(
        action="select", actors=ready_catapults, pick=catapult_bldgs,
        match=dict(where={"if": "distance(actor, target) <= 1"}, score=f"10 * {strat_bldg} + 20 - 60 * distance(actor, target)"),
        issue=[dict(attack={})]
    )

    return [
        variable("effective_hit", "var(tune_hit_damage) + 0.6 * researched(61)"),
        store("enemy_mil_spaces", enemy_mil_spaces),
        store("enemy_vil_spaces", enemy_vil_spaces),
        store("enemy_bldg_spaces", enemy_bldg_spaces),
        store("vil_combat_spaces", vil_combat_spaces),
        store("bldg_combat_spaces", bldg_combat_spaces),
        catapult_stance,
        store("combatants", mil_combatants_source),
        store("combat_spaces", combat_spaces),
        combat_stance,
        engage_mil,
        store("assigned_combatants", assigned_combatants),
        engage_mil_overflow,
        store("vil_combatants", vil_combatants_source),
        store("vil_spaces", vil_spaces),
        engage_vils,
        store("bldg_combatants", bldg_combatants_source),
        store("bldg_spaces", bldg_spaces),
        engage_bldgs,
        engage_catapults,
    ]
