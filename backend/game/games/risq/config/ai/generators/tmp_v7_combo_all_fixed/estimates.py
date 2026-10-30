from __future__ import annotations

from bounds import worker_growth
from common import POWER_TYPES, Record, aggregate, branch, power, selection, store, variable
from locations import enemy_proximity


def composition() -> list[Record]:
    actions: list[Record] = []
    for unit in (1, *POWER_TYPES):
        actions.append(variable(f"memory_{unit}",
            f"max(0, var(memory_{unit}) - var(tune_memory_unit_decay), var(visible_{unit}))", True))
    remembered = " + ".join(f"var(memory_{unit}) * var(tune_power_{name})" for unit, name in POWER_TYPES.items())
    actions.append(store("enemy_unit_spaces", selection("spaces", dict(
        near=dict(distance=0, of=selection("units", dict(in_set="enemy_units")))))))
    actions.append(store("enemy_building_spaces", selection("spaces", dict(building_owner="enemy"))))
    actions.append(store("enemy_presence", selection("spaces", {"if":
        "(inSet(enemy_unit_spaces, target) + inSet(enemy_building_spaces, target) + inSet(poor_vision, target) * (var(target.unit_count) > 0)) > 0"})))
    actions.extend([variable("remembered_power", remembered),
                    variable("remembered_military", " + ".join(f"var(memory_{unit})" for unit in POWER_TYPES)),
                    variable("unknown_unit_power", "max(var(tune_unknown_min_power), var(remembered_power) / max(1, var(remembered_military)))"),
                    variable("unidentified_units", aggregate("sum", selection("spaces", dict(in_set="poor_vision")), "var(item.unit_count)")),
                    variable("visible_power", aggregate("sum", selection("units", dict(in_set="enemy_units")), power()))])
    for unit in (101, 102, 103):
        actions.append(variable(f"enemy_mix_{unit}", f"(var(memory_{unit}) + var(tune_composition_prior)) / max(0.001, var(remembered_military) + 3 * var(tune_composition_prior))"))
    actions.extend([variable("heavies_seen", "var(heavies_seen) + max(0, var(visible_103) - var(previous_heavies))", True),
                    variable("previous_heavies", "var(visible_103)", True)])
    return actions


def unidentified() -> list[Record]:
    actions = enemy_proximity()
    actions.extend([variable("enemy_villager_minimum", f"var(start_villagers) + {worker_growth('reasonable', 'gameVar(turn) - 1')}"),
                    variable("unknown_villager_fraction", "min(var(tune_unknown_villager_share), max(0, var(enemy_villager_minimum) - var(visible_1)) / max(1, var(unknown_near_base)))"),
                    variable("unidentified_power", "var(unidentified_units) * var(unknown_unit_power) - var(unknown_near_base) * var(unknown_villager_fraction) * (var(unknown_unit_power) - var(tune_unknown_villager_power))"),
                    variable("observed_enemy_power", "max(var(remembered_power), var(visible_power) + var(unidentified_power))")])
    return actions


def local_power(owner: str, anchor: str, radius: str, army_only: bool = False) -> Record:
    where = dict(owner=owner, unit_types=["military"], near=dict(
        distance=radius, of=selection("units" if anchor == "army_center" else "spaces", dict(in_set=anchor))))
    if army_only:
        where["bucket"] = "army"
    return aggregate("sum", selection("units", where), power())


def local_observations() -> list[Record]:
    actions: list[Record] = []
    for anchor in ("home", "base", "army_center"):
        for owner in ("own", "enemy"):
            actions.append(variable(f"{anchor}_{owner}_power", local_power(owner, anchor, "var(tune_local_radius)", owner == "own" and anchor == "army_center")))
    for anchor, radius, output in (("home", "var(tune_local_radius)", "home_enemy_power"),
                                  ("base", "var(tune_local_radius)", "base_enemy_power"),
                                  ("army_center", "var(tune_local_radius)", "army_center_enemy_power")):
        kind = "units" if anchor == "army_center" else "spaces"
        unknown = aggregate("sum", selection("spaces", dict(in_set="poor_vision", near=dict(distance=radius,
            of=selection(kind, dict(in_set=anchor))))), "var(item.unit_count)")
        actions.extend([variable(f"{anchor}_unknown", unknown),
                        variable(output, f"var({output}) + var({anchor}_unknown) * var(unknown_unit_power)")])
    actions.append(variable("army_enemy_here", local_power("enemy", "army_center", "0")))
    army_near = selection("units", dict(owner="own", bucket="army", near=dict(
        distance="var(tune_local_radius)", of=selection("units", dict(in_set="army_center")))))
    bonus = "var(tune_tactical_attack_bonus) * researched(61) + var(tune_tactical_armor_bonus) * researched(62)"
    actions.extend([variable("army_unknown_here", 0),
                    dict(action="for_each", select=selection("units", dict(in_set="army_center"), limit=1),
                         **{"as": "center"},
                         do=[variable("army_unknown_here", "var(center.space.unidentified_count)")]),
                    variable("army_enemy_here", "var(army_enemy_here) + var(army_unknown_here) * var(unknown_unit_power)"),
                    variable("army_engaged", "var(army_enemy_here) > 0"),
                    variable("army_tactical_power", aggregate("sum", army_near, f"({power()}) + ({power(infantry_only=True)}) * ({bonus})"))])
    return actions


def strategy() -> list[Record]:
    growth = "var(tune_memory_producer_growth) * (var(enemy_buildings_22) + var(enemy_buildings_23))"
    memory = "max(var(enemy_army_memory) * var(tune_memory_retention) + var(enemy_growth), var(visible_power) + var(tune_unknown_weight) * var(unidentified_power))"
    return [variable("enemy_losses", 0), variable("enemy_growth", growth),
            variable("enemy_army_memory", f"min(var(tune_memory_population_power) * max(0, gameVar(population_max) - var(memory_1)), {memory})", True),
            branch("var(bounds_active)", [variable("enemy_army_memory", "max(var(visible_power), min(var(enemy_army_memory), var(bound_reasonable_cap)))", True)]),
            variable("enemy_tech_multiplier", "1 + var(tune_enemy_smith_bonus) * min(1, var(enemy_buildings_11))"),
            variable("enemy_strength", "var(enemy_army_memory) * var(enemy_tech_multiplier)"),
            branch("var(bounds_active)", [variable("enemy_strength", "max(var(visible_power), min(var(enemy_strength), var(bound_theoretical_cap)))")]),
            variable("own_strategic_multiplier", "1 + var(tune_strategic_tech_bonus) * (researched(61) + researched(62))"),
            variable("own_strategic_power", "var(own_power) * var(own_strategic_multiplier)"),
            variable("recent_enemy_power", "(1 - var(tune_memory_recent_weight)) * var(recent_enemy_power) + var(tune_memory_recent_weight) * var(observed_enemy_power)", True),
            variable("est_close", "(var(n_enemy_buildings) > 0) * max(0, 1 - var(enemy_distance) / var(tune_est_close_ref))"),
            variable("est_behind", "min(1, max(0, (var(recent_enemy_power) - var(own_power)) / max(4, var(recent_enemy_power))))"),
            variable("est_press", "min(1, var(tune_est_prio_close) * var(est_close) + var(tune_est_prio_behind) * var(est_behind))"),
            variable("previous_enemy_local", "var(army_center_enemy_power)", True),
            variable("previous_engaged", "var(army_engaged)", True)]
