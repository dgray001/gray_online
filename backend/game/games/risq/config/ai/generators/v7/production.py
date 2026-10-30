from __future__ import annotations

from common import RESOURCES, Record, aggregate, branch, cost, selection, total_cost, variable, whole


def affordable(kind: str, item: int) -> str:
    return "min(" + ", ".join(f"({cost(kind, item, resource)} > 0) * ({whole(f'(gameVar({resource}) - var(operating_reserve_{resource})) / max(0.001, {cost(kind, item, resource)})')}) + ({cost(kind, item, resource)} == 0) * 1000000" for resource in RESOURCES) + ")"


def create(unit: int, condition: str = "1", queue: str = "var(tune_queue_depth)") -> Record:
    queued = aggregate("count", selection("orders", {"in": dict(order_kind=["create"], item_unit_id=[unit])}))
    amount = f"max(0, min(({queue}) - var(actor.queue_length), {affordable('unit', unit)}, gameVar(population_limit) - gameVar(population), {whole(f'var(goal_{unit}) - var(own_{unit}) - var(actor.queued)')}))"
    action = dict(action="select", actors=selection("buildings", dict(owner="own", can_create=unit), **{"let": dict(queued=queued)}, rank="0 - var(actor.uid)"),
                  issue=[dict(create=dict(unit_id=unit, queue=f"var(actor.queue_length) + ({amount})"))])
    return branch(condition, [variable(f"queued_{unit}", queued), action])


def research(tech: int, condition: str) -> Record:
    queued = aggregate("count", selection("orders", {"in": dict(order_kind=["research"], item_tech_id=[tech])}))
    producer = selection("buildings", dict(owner="own", can_research=tech, **{
        "if": "var(actor.queue_length) < var(tune_queue_depth)"}), rank="0 - var(actor.uid)", limit=1)
    action = dict(action="select", actors=producer, issue=[dict(research=dict(
        tech_id=tech, queue=f"var(actor.queue_length) + min(1, {affordable('tech', tech)})"))])
    return branch(f"(1 - researched({tech})) * ({condition})", [
        variable(f"queued_tech_{tech}", queued), branch(f"var(queued_tech_{tech}) == 0", [action])])


def opening() -> list[Record]:
    return [research(1, "var(own_1) >= var(tune_farming_workers)"), create(1)]


def military() -> list[Record]:
    pierce_queue = "max(1, round(var(tune_queue_depth) * (1 - var(blunt_share))))"
    return [create(103, "var(own_103) == 0", "1"), create(401, "var(own_103) > 0", "1"),
            create(103, "var(own_103) > 0", "1"),
            create(102, "(var(pikes) > 0) + (var(memory_103) >= var(tune_pierce_heavies)) + (var(heavies_seen) >= var(tune_pierce_heavies)) > 0", pierce_queue),
            create(101)]


def upgrades() -> list[Record]:
    return [research(63, "(var(heavies_seen) >= var(tune_pierce_heavies)) + (var(goal_102) - var(own_102) >= var(tune_pikes_unit_threshold)) > 0"),
            research(61, f"var(projected_army_investment) * var(tune_tactical_attack_bonus) >= {total_cost('tech', 61)}"),
            research(62, f"var(projected_army_investment) * var(tune_tactical_armor_bonus) >= {total_cost('tech', 62)}")]


def mercenaries() -> list[Record]:
    reserve = "max(0, var(rate_gold) - var(mercenary_rate_gold)) * var(tune_planning_horizon) + var(research_budget_gold)"
    actions = [variable("gold_surplus", f"max(0, gameVar(gold) - ({reserve}))"),
               variable("gold_idle", "(1 - var(tune_gold_smoothing)) * var(gold_idle) + var(tune_gold_smoothing) * var(gold_surplus)", True),
               research(101, "(var(complete_23) > 0) * (var(gold_surplus) > 0)")]
    for unit, priority in ((102, "var(goal_102) - var(own_102) > var(goal_101) - var(own_101)"),
                           (101, "var(goal_101) - var(own_101) >= var(goal_102) - var(own_102)")):
        count = f"min(var(item.can_afford), {whole(f'(gameVar(gold) - ({reserve})) / max(0.001, var(offer.cost_gold))')}, gameVar(population_limit) - gameVar(population), {whole(f'var(goal_{unit}) - var(own_{unit})')})"
        hire = dict(action="select", pick=selection("zones", dict(owner="own", at=dict(in_set="home")),
                                                    rank=["0 - var(target.zone_x)", "0 - var(target.zone_y)"], limit=1),
                    issue=[dict(hire=dict(unit_id=unit, count=count))])
        actions.append(dict(action="for_each", source="mercenaries", **{"as": "offer"}, do=[
            branch(f"(var(offer.id) == {unit}) * ({priority})", [hire])]))
    return actions


def renew_farms() -> list[Record]:
    farms = selection("buildings", dict(owner="own", building_ids=[3], state="complete", **{
        "if": "var(farm.resources_left) <= 0"}))
    funding = " * ".join(f"(gameVar({resource}) >= var(farm.renew_cost_{resource}))" for resource in RESOURCES)
    actors = selection("units", dict(bucket="gather_food", **{"in": dict(order_kind=["none", "gather"])}),
                       rank="0 - var(actor.uid)", limit="var(tune_renew_workers)")
    target = selection("buildings", {"if": "var(target.uid) == var(farm.uid)"})
    renewal = dict(action="select", actors=actors, pick=target, issue=[dict(renew={})])
    return [dict(action="for_each", select=farms, **{"as": "farm"}, do=[branch(funding, [renewal])])]
