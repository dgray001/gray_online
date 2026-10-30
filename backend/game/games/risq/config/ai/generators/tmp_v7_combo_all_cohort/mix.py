from __future__ import annotations

from common import Definitions, Record, variable


def damage(definitions: Definitions, attacker: Record, target: Record) -> float:
    category = "building" if "building_id" in target else target["unit_type"]
    bonuses = [bonus for bonus in definitions.bonuses if category in bonus["target_types"] and (
        attacker["unit_id"] in bonus.get("affects_unit_ids", []) or attacker["unit_type"] in bonus.get("affects_unit_types", []))]
    attack, defense = 0.0, 0.0
    for kind in attacker.get("attack_type", "none").split("_"):
        attack += attacker.get(f"attack_{kind}", 0) + sum(bonus.get(f"bonus_attack_{kind}", 0) for bonus in bonuses)
        defense += target.get(f"defense_{kind}", 0) * max(0, 1 - attacker.get(f"penetration_{kind}", 0) / 100)
    if attack <= 0:
        return 0
    strength = attack - defense if defense <= attack / 2 else attack * 4 ** (-defense / attack)
    return 3 * max(0.1, strength / 10)


def composition(definitions: Definitions, parameters: dict[str, float]) -> list[Record]:
    actions: list[Record] = []
    for unit in (101, 102, 103):
        terms = [f"var(tune_matchup_{unit}_{enemy}) * var(memory_{enemy})" for enemy in (101, 102, 103)]
        prior = f"var(tune_mix_prior_{unit})"
        if unit == 101:
            terms.append("var(tune_early_blunt_bias) * (var(producers_23) == 0)")
        if unit == 102:
            terms.append("var(tune_pierce_pikes_gain) * var(pikes) * var(memory_103)")
        actions.append(variable(f"value_{unit}", f"({' + '.join(terms)} + {prior}) / max(1, var(memory_101) + var(memory_102) + var(memory_103) + 1)"))
    for unit in (101, 102, 103, 401):
        attacker = definitions.units[unit]
        terms = [f"(var(visible_{enemy}) + var(tune_utility_enemy_prior)) * {damage(definitions, attacker, target)}"
                 for enemy, target in definitions.units.items() if target["unit_type"] != "economic"]
        terms.extend(f"var(tune_utility_building_weight) * var(enemy_buildings_{enemy}) * {damage(definitions, attacker, target)}"
                     for enemy, target in definitions.buildings.items())
        expenses = " + ".join(f"{amount} * (1 + {amount} / max({amount}, gameVar({resource})))" for resource, amount in attacker["cost"].items())
        reach = parameters["utility_range_bonus"] if attacker.get("range", "zone") != "zone" else 1
        factor = attacker["max_health"] ** parameters["utility_survival_exponent"] * reach * attacker["turn_stamina"] / attacker["production_stamina"]
        actions.append(variable(f"utility_{unit}", f"({' + '.join(terms)}) * {factor} / max(1, {expenses})"))
    actions.extend([variable("siege_escorts", "var(own_101) + var(own_102) + var(own_103)"),
                    variable("siege_share", "min(var(tune_siege_share_cap), var(utility_401) / max(0.001, var(utility_101) + var(utility_102) + var(utility_103) + var(utility_401))) * min(1, max(0, var(siege_escorts) / var(tune_siege_escort_floor) - 1)) / (1 + var(enemy_near_base))")])
    return actions


def unit_targets(parameters: dict[str, float]) -> list[Record]:
    blunt = " * ".join(["var(value_101)"] * int(parameters["mix_sharpness"]))
    pierce = " * ".join(["var(value_102)"] * int(parameters["mix_sharpness"]))
    return [variable("blunt_weight", blunt), variable("pierce_weight", pierce),
            variable("blunt_share", "var(blunt_weight) / max(0.001, var(blunt_weight) + var(pierce_weight))"),
            variable("free_army", "max(0, gameVar(population_max) - max(var(worker_goal), var(own_1)) - var(n_military))"),
            variable("goal_1", "var(worker_goal)"),
            variable("goal_101", "var(own_101) + var(free_army) * var(blunt_share)"),
            variable("goal_102", "var(own_102) + var(free_army) * (1 - var(blunt_share))"),
            variable("goal_401", "var(siege_escorts) * var(siege_share) / max(0.001, 1 - var(siege_share))"),
            variable("goal_103", "var(own_103) + max(0, var(free_army) - max(0, var(goal_401) - var(own_401))) * (var(value_103) >= var(tune_heavy_value_gate) * var(value_102))"),
            variable("flow_1", "var(complete_1) * var(production_rate_1) * (var(goal_1) - var(own_1) >= 1)"),
            variable("flow_101", "var(complete_22) * var(blunt_share) * var(production_rate_101) * (var(free_army) > 0)"),
            variable("flow_102", "var(complete_22) * (1 - var(blunt_share)) * var(production_rate_102) * (var(free_army) > 0)"),
            variable("flow_103", "var(complete_23) * (1 - var(siege_share)) * var(production_rate_103) * (var(goal_103) > var(own_103))"),
            variable("flow_401", "var(complete_23) * var(siege_share) * var(production_rate_401) * (var(goal_401) > var(own_401))"),
            variable("population_flow", "var(flow_1) + var(flow_101) + var(flow_102) + var(flow_103) + var(flow_401)")]
