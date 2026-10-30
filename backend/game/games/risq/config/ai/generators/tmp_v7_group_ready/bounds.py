from __future__ import annotations

from common import Definitions, Record, branch, variable


def initialize() -> Record:
    values = dict(bank="gameVar(food) + gameVar(wood) + gameVar(stone)", villagers="var(own_1)", power="var(own_power)",
                  barracks="var(complete_22)", redoubts="var(complete_23)", vcs="var(complete_1)",
                  headroom="max(0, gameVar(population_limit) - gameVar(population))", economy="var(has_economy)")
    return branch("var(initialized) == 0", [
        *[variable(f"start_{name}", expression, True) for name, expression in values.items()],
        variable("initialized", 1, True)])


def worker_growth(plan: str, elapsed: str) -> str:
    turns = f"(max(0, {elapsed}) * max(1, var(start_vcs)))"
    if plan == "none":
        return "0"
    if plan == "floor":
        return f"min({turns}, max(0, var(tune_bound_worker_floor) - var(start_villagers)))"
    if plan == "pump":
        return turns
    first = "max(0, var(tune_bound_worker_first) - var(start_villagers))"
    second = "max(0, var(tune_bound_worker_second) - max(var(start_villagers), var(tune_bound_worker_first)))"
    third = "max(0, var(tune_bound_worker_third) - max(var(start_villagers), var(tune_bound_worker_second)))"
    second_time = f"({second} / var(tune_bound_second_rate))"
    return (f"min({turns}, {first})"
            f" + min(max(0, {turns} - {first}), {second_time}) * var(tune_bound_second_rate)"
            f" + min(max(0, {turns} - {first} - {second_time}), {third} / var(tune_bound_third_rate)) * var(tune_bound_third_rate)")


def projection(definitions: Definitions, plan: str, rate: str, extra_turn: str) -> list[Record]:
    house, barracks = definitions.buildings[2], definitions.buildings[22]
    worker_stamina = float(definitions.units[1]["turn_stamina"])
    house_cost = definitions.gathered_cost(house)
    housing = f"({house_cost} + {rate} * {house['build_stamina']} / {worker_stamina}) / {house['population_support']}"
    labor = f"{rate} * {barracks['build_stamina']} / {worker_stamina}"
    producer_cost = f"({definitions.gathered_cost(barracks)} + {labor})"
    first_build_turns = f"max(1, round({barracks['build_stamina']} / ({worker_stamina} * max(1, var(start_villagers))) + 0.499999))"
    blunt_rate = definitions.production_rate(22, 101)
    heavy_rate = definitions.production_rate(23, 103)
    name = f"bound_{plan}"
    capacity = (f"(var(start_barracks) * {blunt_rate} * var(tune_power_blunt)"
                f" + var(start_redoubts) * {heavy_rate} * var(tune_power_heavy)) * max(0, gameVar(turn) - 1)"
                f" + (var(start_barracks) == 0) * max(0, gameVar(turn) - ({first_build_turns})) * {blunt_rate} * var(tune_power_blunt)")
    blunt_cost = f"({definitions.gathered_cost(definitions.units[101])} + ({housing})) / var(tune_power_blunt)"
    heavy_cost = f"({definitions.gathered_cost(definitions.units[103])} + ({housing})) / max(0.001, var(tune_power_heavy))"
    unit_cost = f"(var(start_redoubts) == 0) * ({blunt_cost}) + (var(start_redoubts) > 0) * min({blunt_cost}, {heavy_cost})"
    worker_cost = f"{definitions.gathered_cost(definitions.units[1])} + ({housing})"
    share = "var(tune_bound_military_share)" if plan == "reasonable" else "1"
    pool = (f"{share} * (var(start_bank) + ({housing}) * var(start_headroom)"
            f" - (var(start_barracks) == 0) * {producer_cost} + var({name}_income)"
            f" - ({worker_cost}) * ({worker_growth(plan, 'gameVar(turn)')}))")
    extra = f"max(0, gameVar(turn) - {extra_turn}) * {blunt_rate} * var(tune_power_blunt)"
    second_pool = f"var({name}_pool) - (gameVar(turn) >= {extra_turn}) * {producer_cost}"
    return [variable(f"{name}_income", f"var({name}_income) + {rate} * (var(start_villagers) + ({worker_growth(plan, 'gameVar(turn) - 1')}))", True),
            variable(f"{name}_capacity", capacity), variable(f"{name}_pool", pool),
            variable(name, f"var(start_power) + var(start_economy) * max(min(var({name}_capacity), max(0, var({name}_pool)) / max(0.001, {unit_cost})), min(var({name}_capacity) + {extra}, max(0, {second_pool}) / max(0.001, {unit_cost})))")]


def bounds(definitions: Definitions) -> list[Record]:
    actions: list[Record] = []
    for plan in ("none", "floor", "pump", "reasonable"):
        assumption = "reasonable" if plan == "reasonable" else "theoretical"
        actions.extend(projection(definitions, plan, f"var(tune_bound_{assumption}_income)",
                                  f"var(tune_bound_{assumption}_barracks_turn)"))
    actions.extend([
        variable("bound_uncertainty", "var(bound_uncertainty) + var(tune_bound_uncertainty_rate)", True),
        variable("bounds_active", "var(bound_uncertainty) < var(tune_bound_uncertainty_limit)"),
        variable("bound_fraction", "min(1, var(bound_uncertainty) / var(tune_bound_uncertainty_limit))"),
        variable("bound_theoretical", "max(var(bound_none), var(bound_floor), var(bound_pump))"),
        variable("bound_theoretical_cap", "var(bound_theoretical) * (1 + var(tune_bound_widening) * var(bound_fraction))"),
        variable("bound_reasonable_cap", "var(bound_reasonable) + var(tune_bound_interpolation) * (var(bound_theoretical) - var(bound_reasonable)) * var(bound_fraction)"),
    ])
    return actions
