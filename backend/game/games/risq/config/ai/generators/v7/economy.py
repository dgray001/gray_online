from __future__ import annotations

from common import RESOURCES, Definitions, Record, aggregate, cost, selection, total_cost, variable


def production_capacity(definitions: Definitions) -> list[Record]:
    actions: list[Record] = []
    for building, units in ((1, (1,)), (22, (101, 102)), (23, (103, 401))):
        planned = aggregate("count", selection("foundations", dict(building_ids=[building], **{"if": "var(item.is_planned)"})))
        actions.append(variable(f"producers_{building}", planned))
        actions.append(variable(f"producers_{building}", f"var(producers_{building}) + var(buildings_{building})"))
        for unit in units:
            actions.append(variable(f"production_rate_{unit}", definitions.production_rate(building, unit)))
    actions.extend([variable("gold_income", aggregate("sum", selection("spaces", dict(owner="own")), "var(item.gold_income)")),
                    variable("enemy_near_base", aggregate("count", selection("units", dict(owner="enemy", unit_types=["military"],
                        near=dict(distance="var(tune_emergency_radius)", of=selection("spaces", dict(in_set="base")))))))])
    actions.extend([variable("region_gold", aggregate("sum", selection("regions", {"if": "var(item.owner) == var(player_id)"}), "var(item.gold_bonus)")),
                    variable("gold_income", "var(gold_income) + var(region_gold)")])
    return actions


def workforce(definitions: Definitions) -> list[Record]:
    barracks = f"max({total_cost('unit', 101, RESOURCES[:3])} * var(production_rate_101), {total_cost('unit', 102, RESOURCES[:3])} * var(production_rate_102))"
    redoubt = f"(1 - var(siege_share)) * {total_cost('unit', 103, RESOURCES[:3])} * var(production_rate_103) + var(siege_share) * {total_cost('unit', 401, RESOURCES[:3])} * var(production_rate_401)"
    actions = [variable("barracks_running_cost", barracks), variable("redoubt_running_cost", redoubt),
               variable("bank_rate", f"max(0, gameVar(food) + gameVar(wood) + gameVar(stone) - {cost('unit', 1, 'food')} - {cost('building', 2, 'wood')}) / var(tune_investment_horizon)"),
               variable("committed_rate", f"var(producers_22) * var(barracks_running_cost) + var(producers_23) * var(redoubt_running_cost) + var(complete_1) * min(1, max(0, var(worker_want_last))) * {total_cost('unit', 1, RESOURCES[:3])} * var(production_rate_1) * (1 - var(est_press))"),
               variable("sustainable_surplus", "var(available_gatherers) * var(worker_income) + max(0, var(bank_rate) - var(tune_bank_surplus_reserve)) - var(committed_rate)"),
               variable("army_target", "gameVar(population_max) * (1 - var(tune_worker_share))"),
               variable("forecast_barracks", "max(1, min(var(army_target) / var(tune_production_turnover), var(producers_22) + var(tune_production_lookahead) * max(1, var(barracks_want_last))))")]
    for resource in RESOURCES[:3]:
        flow = (f"var(forecast_barracks) * max({cost('unit', 101, resource)} * var(production_rate_101), {cost('unit', 102, resource)} * var(production_rate_102))"
                f" + var(producers_23) * ((1 - var(siege_share)) * {cost('unit', 103, resource)} * var(production_rate_103) + var(siege_share) * {cost('unit', 401, resource)} * var(production_rate_401))"
                f" + var(complete_1) * {cost('unit', 1, resource)} * var(production_rate_1)")
        if resource == "wood":
            house = definitions.buildings[2]
            farm = definitions.buildings[3]["gatherable"]
            flow += f" + (var(complete_1) + var(forecast_barracks) + var(producers_23) * var(production_rate_103)) * {cost('building', 2, 'wood')} / {house['population_support']}"
            flow += f" + var(farming) * var(forecast_food) * {farm['renew_cost'].get('wood', 0)} / {farm['starting_resources']}"
        reserve = f"max({cost('unit', 1, resource)}, {cost('unit', 101, resource)}, {cost('unit', 102, resource)}, {cost('unit', 103, resource)} * min(1, var(producers_23))) + {cost('building', 2, resource)}"
        actions.extend([variable(f"forecast_{resource}", flow),
                        variable(f"worker_demand_{resource}", f"max(0, var(forecast_{resource}) + (({reserve}) - gameVar({resource})) / var(tune_planning_horizon))")])
    funding = " + ".join(f"var(worker_demand_{resource})" for resource in RESOURCES[:3])
    actions.append(variable("worker_goal", f"min(max(var(tune_worker_floor), gameVar(population_max) * var(tune_worker_share)), max(var(tune_worker_floor), ({funding}) / var(worker_income) + var(tune_worker_buffer)))"))
    actions.append(variable("worker_want_last", "max(0, var(worker_goal) - var(own_1)) * (var(complete_1) > 0)", True))
    return actions


def infrastructure(definitions: Definitions) -> list[Record]:
    bootstrap = f"(var(own_1) >= var(tune_worker_floor)) + ((gameVar(food) >= var(tune_bootstrap_batches) * ({cost('unit', 1, 'food')} + {cost('unit', 101, 'food')})) * (gameVar(wood) >= {cost('building', 22, 'wood')} + var(tune_bootstrap_batches) * {cost('unit', 101, 'wood')} + {cost('building', 2, 'wood')}))"
    actions = [variable("barracks_bootstrap", bootstrap),
            variable("want_22", f"(var(own_1) > 0) * (((var(producers_22) == 0) * ((var(barracks_bootstrap) > 0) + (var(enemy_near_base) > 0)) > 0) + ((var(producers_22) > 0) * (var(sustainable_surplus) >= var(tune_additional_producer_margin) * var(barracks_running_cost)) * (var(sustainable_surplus) * var(tune_investment_horizon) >= {total_cost('building', 22)})))"),
            variable("want_23", f"(var(own_1) > 0) * (var(sustainable_surplus) >= var(tune_additional_producer_margin) * var(redoubt_running_cost)) * ((var(gold_income) + gameVar(gold) / var(tune_investment_horizon) >= (var(producers_23) + 1) * {cost('unit', 103, 'gold')} * var(production_rate_103)) + (var(goal_401) > var(own_401)) > 0)"),
            variable("barracks_want_last", "min(1, var(want_22))", True),
            variable("want_1", "(var(own_1) > 0) * (var(producers_1) == 0)"),
            variable("want_2", f"(var(own_1) > 0) * (gameVar(population_limit) < gameVar(population_max)) * max(0, (var(population_flow) * var(tune_housing_horizon) + 1 - var(headroom)) / {definitions.buildings[2]['population_support']})"),
            variable("army_investment", " + ".join(f"var(own_{unit}) * {total_cost('unit', unit)}" for unit in (101, 102, 103))),
            variable("projected_army_investment", "var(army_investment) + (var(producers_22) * var(barracks_running_cost) + var(producers_23) * var(redoubt_running_cost)) * var(tune_investment_horizon)"),
            variable("want_11", f"(var(own_1) > 0) * (var(buildings_11) + var(foundations_11) == 0) * (var(projected_army_investment) * var(tune_tactical_attack_bonus) >= {total_cost('tech', 61)} + {total_cost('building', 11)})")]
    idle_gold = f"(var(producers_23) == 0) * (var(own_1) > 0) * (var(gold_idle) >= var(tune_surplus_hire_batches) * var(tune_hire_price_estimate) * {total_cost('unit', 101)})"
    actions.append(variable("want_23", f"max(var(want_23), {idle_gold})"))
    return actions


def food_supply(definitions: Definitions) -> list[Record]:
    nearby = dict(distance="var(tune_wild_food_radius)", of=selection("spaces", dict(in_set="base")))
    occupied = dict(distance=0, of=selection("units", dict(in_set="villagers")))
    farm = definitions.buildings[3]["gatherable"]
    return [variable("wild_food_nearby", aggregate("sum", selection("resources", dict(category="food", near=nearby)), "var(item.amount_left)")),
            variable("wild_food_worked", aggregate("sum", selection("resources", dict(category="food", near=occupied)), "var(item.amount_left)")),
            variable("wild_food_confidence", "min(1, max(0, (var(own_1) - var(tune_worker_floor)) / var(tune_wild_food_walk_scale)))"),
            variable("wild_food_workers", "(var(wild_food_worked) + max(0, var(wild_food_nearby) - var(wild_food_worked)) * var(wild_food_confidence)) / (var(worker_income) * var(tune_planning_horizon))"),
            variable("active_farms", aggregate("count", selection("gatherables", dict(category="food", **{"if": "var(item.is_building)"})))),
            variable("depleted_farms", "max(0, var(complete_3) - var(active_farms))"),
            variable("food_workers_required", "min(var(available_gatherers), max(var(food_workers_last), var(worker_demand_food) / var(worker_income)))"),
            variable("want_3", f"var(farming) * max(0, (var(food_workers_required) - var(wild_food_workers)) / {farm['gather_capacity']} + 1 - var(active_farms) - var(foundations_3))"),
            variable("smith_age", "max(0, var(smith_age) + 2 * var(want_11) - 1)", True),
            variable("redoubt_age", "max(0, var(redoubt_age) + 2 * var(want_23) - 1)", True),
            variable("smith_urgency", "min(1, max(var(smith_age) / var(tune_smith_urgency_turns), (var(heavies_seen) >= var(tune_pierce_heavies)) * (1 - var(pikes))))"),
            variable("redoubt_urgency", "min(1, var(redoubt_age) / var(tune_redoubt_urgency_turns))"),
            variable("house_urgency", "min(1, max(0, 1 - var(headroom) / max(0.001, var(population_flow) * var(tune_housing_horizon))))")]
