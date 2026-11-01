from __future__ import annotations

from common import Definitions, POWER_TYPES, Record, aggregate, power, selection, store, variable
from parameters import POWER_LEVELS, POWER_TECHS


def observations(definitions: Definitions) -> list[Record]:
    actions = [dict(action="for_each", source="players", where=dict(owner="own"),
                    **{"as": "me"}, do=[variable("player_id", "var(me.id)")])]
    for name, kind, where in (
        ("own_units", "units", dict(owner="own")),
        ("own_buildings", "buildings", dict(owner="own")),
        ("enemy_units", "units", dict(owner="enemy")),
        ("enemy_buildings", "buildings", dict(owner="enemy")),
        ("villagers", "units", dict(owner="own", unit_ids=[1])),
        ("military", "units", dict(owner="own", unit_types=["military"])),
    ):
        actions.append(store(name, selection(kind, where)))
    actions.append(store("poor_vision", selection("spaces", dict(vision=2))))
    enemy_military = selection("units", dict(owner="enemy", unit_types=["military"]))
    actions.append(store("enemy_mil_spaces", selection("spaces", dict(near=dict(distance=0, of=enemy_military)))))
    actions.append(store("enemy_mil_near_spaces", selection("spaces", dict(near=dict(distance=1, of=enemy_military)))))
    actions.append(store("unidentified_spaces", selection("spaces", dict(in_set="poor_vision", **{"if": "var(target.unit_count) > 0"}))))
    actions.append(store("unidentified_near_spaces", selection("spaces", dict(near=dict(distance=1, of=selection("spaces", dict(in_set="unidentified_spaces")))))))
    actions.append(store("enemy_presence_near_spaces", selection("spaces", {"if": "(inSet(enemy_mil_near_spaces, target) + inSet(unidentified_near_spaces, target)) > 0"})))
    actions.append(store("threatened_villagers", selection("units", dict(owner="own", unit_ids=[1], at=dict(in_set="enemy_presence_near_spaces")))))
    for unit in (1, *POWER_TYPES):
        for prefix, owner in (("own", "own"), ("visible", "enemy")):
            source = selection("units", dict(owner=owner, unit_ids=[unit]))
            actions.append(variable(f"{prefix}_{unit}", aggregate("count", source)))
    for building in (1, 2, 3, 11, 21, 22, 23):
        for prefix, where in (("buildings", {}), ("complete", dict(state="complete"))):
            source = selection("buildings", dict(owner="own", building_ids=[building], **where))
            actions.append(variable(f"{prefix}_{building}", aggregate("count", source)))
        source = selection("foundations", dict(building_ids=[building]))
        actions.append(variable(f"foundations_{building}", aggregate("count", source)))
    economic_ids = [unit for unit, item in definitions.units.items() if item["unit_type"] == "economic"]
    economic = aggregate("count", selection("units", dict(owner="own", unit_ids=economic_ids)))
    producers = aggregate("count", selection("buildings", dict(owner="own", can_create=1, state="complete")))
    actions.extend([variable("economic_units", economic), variable("producer_count", producers),
                    variable("has_economy", "var(economic_units) + var(producer_count) > 0"),
                    variable("mercenaries_on_offer", 0),
                    dict(action="for_each", source="mercenaries", **{"as": "offer"}, do=[variable("mercenaries_on_offer", 1)]),
                    variable("has_production", "var(has_economy) + var(complete_22) + var(complete_23) + var(mercenaries_on_offer) > 0"),
                    variable("headroom", "gameVar(population_limit) - gameVar(population)"),
                    variable("population_ceiling", "max(gameVar(population_max), gameVar(population_limit))")])
    actions.extend([
        dict(action="if", when={"value_at_most": dict(value="gameVar(turn)", amount=1)},
             then=[variable("board_spaces", aggregate("count", selection("spaces")), True)]),
        variable("explored_spaces", aggregate("count", selection("spaces", dict(vision_at_least=1)))),
        variable("farming", "researched(1)"), variable("pikes", "researched(63)"),
        variable("worker_income", "var(tune_worker_income) + var(tune_worker_farming_income) * var(farming)"),
        variable("villager_power", "var(tune_power_villager) + var(tune_power_farming_bonus) * var(farming)"),
        variable("had_city", "max(var(had_city), min(1, var(complete_1)))", True),
        variable("available_gatherers", aggregate("count", selection("units", dict(
            owner="own", unit_ids=economic_ids, **{"in": dict(order_kind=["none", "gather"])},
            **{"if": "1 - var(item.is_garrisoned)"})))),
    ])
    for resource in ("food", "wood", "stone", "gold"):
        source = selection("gatherables", dict(category=resource))
        actions.extend([variable(f"remaining_{resource}", aggregate("sum", source, "var(item.amount_left)")),
                        variable(f"capacity_{resource}", aggregate("sum", source, "var(item.gather_capacity)"))])
    for building in definitions.buildings:
        source = selection("buildings", dict(owner="enemy", building_ids=[building]))
        actions.append(variable(f"enemy_buildings_{building}", aggregate("count", source)))
    actions.extend(strength_observations())
    actions.append(variable("own_power", aggregate("sum", selection("units", dict(in_set="military")), power())))
    return actions


def strength_observations() -> list[Record]:
    siege = selection("units", dict(in_set="enemy_units", unit_ids=[401]))
    actions = [variable("enemy_siege_reach", aggregate("max", siege, "var(item.attack_reach)"))]
    for unit, name in POWER_TYPES.items():
        own = " + ".join(f"{1 << bit} * researched({tech})" for bit, tech in enumerate(POWER_TECHS[unit]))
        if unit == 401:
            enemy = "(var(enemy_siege_reach) > 0) + 2 * min(1, var(enemy_buildings_23))"
        else:
            enemy = f"{3 if unit == 101 else 7} * min(1, var(enemy_buildings_11))"
        for side, mask in (("own", own), ("enemy", enemy)):
            values = []
            for state in range(len(POWER_LEVELS[unit])):
                parameter = f"power_{name}" + (f"_state_{state}" if state else "")
                values.append(f"(({mask}) == {state}) * var(tune_{parameter})")
            actions.append(variable(f"{side}_power_{name}", " + ".join(values)))
    return actions
