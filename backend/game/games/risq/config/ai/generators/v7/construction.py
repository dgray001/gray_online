from __future__ import annotations

from typing import Union

from common import RESOURCES, Definitions, Record, aggregate, branch, cost, selection, store, variable
from foundations import TRACKED, site_clear
from settlement import vc_placement
from sites import build_action


def housing_crew_sizing(definitions: Definitions) -> list[Record]:
    house_build_stamina = definitions.buildings[2]["build_stamina"]
    return [
        variable("villager_build_stamina", "8 + 2 * var(farming)"),
        variable("housing_turns_left", "var(headroom) / max(0.1, var(population_flow))"),
        variable("house_builders",
                 f"min(4, max(1, round({house_build_stamina} / (var(villager_build_stamina) * max(0.5, var(housing_turns_left) - 1)) + 0.49)))"),
    ]


def building_affordable(building_id: int) -> str:
    checks = [f"(gameVar({res}) >= {cost('building', building_id, res)})" for res in RESOURCES]
    return "(" + " * ".join(checks) + ")"


def service_unbuilt_foundations() -> list[Record]:
    actions: list[Record] = []
    for name, (building, _) in TRACKED.items():
        actors = selection("units", dict(
            owner="own", unit_ids=[1],
            **{"in": dict(order_kind=["none", "gather"])},
            **{"if": "1 - var(actor.is_garrisoned)"},
        ), limit=1)
        pick = selection("foundations", dict(building_ids=[building], **{"if": "var(target.builders) == 0"}), limit=1)
        assist = dict(action="select", actors=actors, pick=pick, issue=[dict(build=dict(building_id=building))])
        condition = f"(var({name}_abandoned) == 0) * (var(foundations_{building}) > 0)"
        actions.append(branch(condition, [assist]))
    return actions


def recover_abandoned_foundations() -> list[Record]:
    actions: list[Record] = []
    for name, (building, _) in TRACKED.items():
        actors = selection("units", dict(
            owner="own", unit_ids=[1],
            **{"in": dict(order_kind=["none", "gather"])},
            **{"if": "1 - var(actor.is_garrisoned)"},
        ), limit=1)
        pick = selection("foundations", dict(building_ids=[building], **{"if": "var(target.builders) == 0"}), limit=1)
        resend = dict(action="select", actors=actors, pick=pick, issue=[dict(build=dict(building_id=building))])
        condition = f"(var({name}_abandoned) > 0) * (var({name}_replacement) == 0) * (var({name}_resend_wait) >= var(tune_resend_every)) * {site_clear(name)}"
        recovery = [resend, variable(f"{name}_resend_wait", 0, True)]
        actions.append(branch(condition, recovery))
    return actions


def repairs() -> list[Record]:
    actors = selection("units", dict(
        owner="own", unit_ids=[1],
        **{"in": dict(order_kind=["none", "gather"])},
        **{"if": "1 - var(actor.is_garrisoned)"},
    ), limit=1)
    pick = selection("buildings", dict(owner="own", state="complete", **{"if": "var(target.hp_pct) < 1"}),
                     rank="0 - var(target.hp_pct)", limit=1)
    return [dict(action="select", actors=actors, pick=pick, issue=[dict(repair={})])]


def isolate_builders() -> list[Record]:
    builders = selection("units", dict(
        owner="own", unit_ids=[1],
        **{"in": dict(order_kind=["build", "repair"])},
    ))
    available = selection("units", dict(
        in_set="villagers",
        not_in_set="construction_workers",
        **{"in": dict(order_kind=["none", "gather"]), "if": "1 - var(target.is_garrisoned)"}
    ))
    return [store("construction_workers", builders), store("available_builders", available)]


def urgent_smith() -> list[Record]:
    cond = f"(var(want_11) > 0) * (var(smith_urgency) >= 1) * (var(complete_11) + var(foundations_11) == 0) * {building_affordable(11)}"
    return build_action("smith", 11, builders=2, condition=cond)


def settlement_and_shelter() -> list[Record]:
    actions: list[Record] = []
    vc_cond = f"(var(want_1) > 0) * (var(complete_1) + var(foundations_1) == 0) * {building_affordable(1)}"
    actions.extend(vc_placement(condition=vc_cond, builders=4))

    house_cond = f"(var(want_2) > var(foundations_2)) * {building_affordable(2)}"
    actions.extend(build_action("house", 2, builders="var(house_builders)", condition=house_cond))

    return actions


def farms(max_farms: int = 3) -> list[Record]:
    actions: list[Record] = []
    for i in range(max_farms):
        farm_cond = f"(var(want_3) > var(foundations_3) + {i}) * {building_affordable(3)}"
        actions.extend(build_action("farm", 3, builders=2, condition=farm_cond))

    return actions


def first_barracks() -> list[Record]:
    cond = f"(var(producers_22) == 0) * (var(want_22) > 0) * (var(foundations_22) == 0) * {building_affordable(22)}"
    return build_action("barracks", 22, builders=3, condition=cond)


def expansion_buildings() -> list[Record]:
    actions: list[Record] = []
    rax_cond = f"(var(producers_22) > 0) * (var(want_22) > 0) * (var(foundations_22) == 0) * {building_affordable(22)}"
    actions.extend(build_action("barracks", 22, builders=3, condition=rax_cond))

    smith_cond = f"(var(want_11) > 0) * (var(complete_11) + var(foundations_11) == 0) * {building_affordable(11)}"
    actions.extend(build_action("smith", 11, builders=2, condition=smith_cond))

    redoubt_cond = f"(var(want_23) > 0) * (var(foundations_23) == 0) * {building_affordable(23)}"
    actions.extend(build_action("redoubt", 23, builders=4, condition=redoubt_cond))

    outpost_cond = f"(var(want_21) > 0) * (var(foundations_21) == 0) * {building_affordable(21)}"
    actions.extend(build_action("outpost", 21, builders=1, condition=outpost_cond))

    return actions


def buildable_spaces() -> list[Record]:
    within = selection("spaces", dict(near=dict(distance=1, of=selection("spaces", dict(owner="own"))), **{
        "if": "(var(target.owner) == var(player_id)) + (var(target.owner) < 0) > 0"}))
    unclaimed = selection("spaces", dict(owner="unowned"))
    return [variable("own_space_count", aggregate("count", selection("spaces", dict(owner="own")))),
            branch("var(own_space_count) == 0",
                   [store("buildable_spaces", unclaimed)],
                   [store("buildable_spaces", within)])]


def buildable_zones() -> list[Record]:
    has_foundation = aggregate("count", selection("foundations", {"if": "distance(target, item, 1) == 0"}))
    zones = selection("zones", dict(building=False, resource=False, at=dict(in_set="buildable_spaces"),
                                    **{"if": "var(target.has_foundation) == 0"}),
                      let=dict(has_foundation=has_foundation))
    return [store("buildable_zones", zones)]
