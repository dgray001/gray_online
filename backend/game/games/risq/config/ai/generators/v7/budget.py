from __future__ import annotations

from common import RESOURCES, Definitions, Record, aggregate, cost, selection, variable


def spending_rates(definitions: Definitions) -> list[Record]:
    house, farm = definitions.buildings[2], definitions.buildings[3]["gatherable"]
    actions: list[Record] = []
    for resource in RESOURCES:
        flow = " + ".join(f"var(flow_{unit}) * {cost('unit', unit, resource)}" for unit in (1, 101, 102, 103, 401))
        flow += f" + var(population_flow) * {cost('building', 2, resource)} / {house['population_support']}"
        flow += f" + min(var(food_workers_last), var(active_farms) * {farm['gather_capacity']}) * var(worker_income) * {farm['renew_cost'].get(resource, 0)} / {farm['starting_resources']}"
        research = f"(1 - var(farming)) * {cost('tech', 1, resource)}"
        research += " + (min(1, var(complete_11) + var(want_11))) * (" + " + ".join(
            f"(1 - researched({tech})) * {cost('tech', tech, resource)}" for tech in (2, 3, 5)) + ")"
        actions.extend([variable(f"rate_{resource}", flow), variable(f"research_budget_{resource}", research)])
    average_gold = aggregate("sum", selection("spaces"), "var(item.gold_income)")
    actions.extend([variable("map_space_gold", average_gold),
                    variable("want_21", "(var(complete_22) > 0) * max(0, (var(rate_gold) + var(research_budget_gold) / var(tune_planning_horizon) - var(gold_income)) / max(0.001, var(map_space_gold) / max(1, var(board_spaces))))")])
    return actions


def resource_budgets(definitions: Definitions) -> list[Record]:
    renewing = aggregate("count", dict(selection("orders", {"in": dict(order_kind=["renew"])}), distinct="target_uid"))
    actions = [variable("renewing_farms", renewing)]
    for resource in RESOURCES:
        planned = " + ".join(f"var(want_{building}) * {cost('building', building, resource)}" for building in (1, 2, 3, 11, 21, 22, 23))
        renewal = definitions.buildings[3]["gatherable"]["renew_cost"].get(resource, 0)
        actions.append(variable(f"operating_reserve_{resource}", f"min(max(0, var(depleted_farms) - var(renewing_farms)), max(1, round(var(food_workers_required) / var(tune_renew_workers)))) * {renewal}"))
        saving = f"var(planned_{resource}) + var(depleted_farms) * {renewal}"
        saving += f" + var(want_2) * {cost('building', 2, resource)} * (var(tune_planning_horizon) - 1) * var(house_urgency)"
        saving += f" + var(want_11) * {cost('building', 11, resource)} * (var(tune_smith_saving_max) - 1) * var(smith_urgency)"
        saving += f" + var(want_23) * {cost('building', 23, resource)} * (var(tune_redoubt_saving_max) - 1) * var(redoubt_urgency)"
        buffer = f"max({cost('unit', 1, resource)}, {cost('unit', 101, resource)}, {cost('unit', 102, resource)}, {cost('unit', 103, resource)} * min(1, var(complete_23))) + {cost('building', 2, resource)}"
        passive = "var(gold_income)" if resource == "gold" else "0"
        need = f"max(0, var(rate_{resource}) + (var(saving_{resource}) + var(research_budget_{resource}) + ({buffer}) - gameVar({resource})) / var(tune_planning_horizon) - {passive})"
        actions.extend([variable(f"planned_{resource}", planned), variable(f"saving_{resource}", saving),
                        variable(f"need_{resource}", f"({need}) * (var(capacity_{resource}) > 0)"),
                        variable(f"protected_{resource}", f"min(var(need_{resource}), max(0, var(rate_{resource}) + max(0, var(planned_{resource}) - gameVar({resource})) / var(tune_construction_horizon) - {passive})) / var(worker_income)"),
                        variable(f"investment_{resource}", f"max(0, var(need_{resource}) / var(worker_income) - var(protected_{resource}))")])
    return actions


def worker_targets() -> list[Record]:
    protected = " + ".join(f"var(protected_{resource})" for resource in RESOURCES)
    investment = " + ".join(f"var(investment_{resource})" for resource in RESOURCES)
    actions = [variable("economy_maturity", "min(1, var(own_1) / max(1, var(worker_goal)))"),
               variable("investment_workers", f"min({investment}, var(n_available_builders) * var(tune_investment_share) * var(economy_maturity) * var(economy_maturity))"),
               variable("protected_scale", f"min(1, max(0, var(n_available_builders) - var(investment_workers)) / max(0.001, {protected}))"),
               variable("investment_scale", f"min(1, min(var(investment_workers), max(0, var(n_available_builders) - var(protected_scale) * ({protected}))) / max(0.001, {investment}))")]
    for resource in RESOURCES:
        actions.append(variable(f"workers_{resource}", f"var(protected_{resource}) * var(protected_scale) + var(investment_{resource}) * var(investment_scale)"))
    total = " + ".join(f"var(workers_{resource})" for resource in RESOURCES)
    actions.extend([variable("spare_workers", f"max(0, var(n_available_builders) - ({total}))"),
                    variable("spare_food_share", "(var(need_food) + 1) / (var(need_food) + var(need_wood) + 2)"),
                    variable("workers_food", "var(workers_food) + var(spare_workers) * var(spare_food_share)"),
                    variable("workers_wood", "var(workers_wood) + var(spare_workers) * (1 - var(spare_food_share))"),
                    variable("food_slots", "var(farming) * (2 * var(active_farms) + var(wild_food_workers)) + (1 - var(farming)) * 1000"),
                    variable("farm_wood_short", f"max(0, var(want_3) * {cost('building', 3, 'wood')} + var(depleted_farms) * 60 - gameVar(wood))"),
                    variable("food_over", "min(max(0, var(workers_food) - var(food_slots)), var(farm_wood_short) / (var(worker_income) * 2))"),
                    variable("workers_food", "var(workers_food) - var(food_over)"),
                    variable("workers_wood", "var(workers_wood) + var(food_over)")])
    for resource in RESOURCES:
        offset = "0" if resource == "food" else "0.499999"
        actions.append(variable(f"workers_{resource}", f"max(0, round(min(var(workers_{resource}), var(capacity_{resource})) - {offset}))"))
    return actions
