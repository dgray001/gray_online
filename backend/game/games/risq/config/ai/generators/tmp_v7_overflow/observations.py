from __future__ import annotations

from common import Definitions, POWER_TYPES, Record, aggregate, power, selection, store, variable


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
                    variable("headroom", "gameVar(population_limit) - gameVar(population)"),
                    variable("own_power", aggregate("sum", selection("units", dict(in_set="military")), power()))])
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
    return actions
