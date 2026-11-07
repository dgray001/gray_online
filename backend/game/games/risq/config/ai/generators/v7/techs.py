from __future__ import annotations

from common import RESOURCES, Record, aggregate, branch, cost, selection, variable

TECH_BUILDINGS = {2: 1, 3: 1, 11: 3, 64: 11, 91: 22, 92: 22, 102: 23}
INFANTRY = "(var(own_101) + var(own_102) + var(own_103))"
APPLICABLE = {2: "var(goal_1) - var(own_1) < 1", 3: "var(goal_1) - var(own_1) < 1", 11: "(var(complete_3) >= 1) * (var(own_1) >= 4)", 64: "var(complete_23) >= 1",
              91: f"{INFANTRY} >= 8", 92: f"({INFANTRY} >= 15) * (var(enemy_buildings_24) >= 1)", 102: "var(own_401) >= 1"}


def pending(tech: int) -> str:
    return f"(var(tune_tech_{tech}) > 0) * (1 - researched({tech})) * (var(complete_{TECH_BUILDINGS[tech]}) > 0) * (var(queued_tech_{tech}) == 0) * ({APPLICABLE[tech]})"


def tech_demand() -> list[Record]:
    actions: list[Record] = []
    for tech in TECH_BUILDINGS:
        queued = aggregate("count", selection("orders", {"in": dict(order_kind=["research"], item_tech_id=[tech])}))
        actions.extend([variable(f"queued_tech_{tech}", queued), variable(f"tech_pending_{tech}", pending(tech))])
    for resource in RESOURCES:
        reserve = " + ".join(f"var(tech_pending_{tech}) * var(tune_tech_{tech}) * {cost('tech', tech, resource)}" for tech in TECH_BUILDINGS) if resource == "gold" else "0"
        actions.extend([variable(f"tech_reserve_{resource}", reserve),
                        variable(f"research_budget_{resource}", f"var(research_budget_{resource}) + var(tech_reserve_{resource})")])
    return actions


def buy(tech: int, values: dict[str, float] | None = None) -> Record:
    reserve_mode = int(values.get("issue_tech_reserve", 0)) if values else 0
    if reserve_mode == 1:
        res_deduct = " - var(operating_reserve_{resource})"
    elif reserve_mode == 2:
        res_deduct = " - var(operating_reserve_{resource}) - var(reserve_{resource})"
    else:
        res_deduct = ""
    funds = " * ".join(f"(gameVar({resource}){res_deduct.format(resource=resource)} >= {cost('tech', tech, resource)} / max(0.001, var(tune_tech_{tech})))" for resource in RESOURCES)
    producer = selection("buildings", dict(owner="own", can_research=tech, **{
        "if": "var(actor.queue_length) < var(tune_queue_depth)"}), rank="0 - var(actor.uid)", limit=1)
    action = dict(action="select", actors=producer, issue=[dict(research=dict(tech_id=tech, queue="var(actor.queue_length) + 1"))])
    return branch(f"var(tech_pending_{tech}) * {funds}", [action])


def tech_buy(values: dict[str, float] | None = None) -> list[Record]:
    return [buy(tech, values) for tech in TECH_BUILDINGS]


def tech_hold(values: dict[str, float] | None = None) -> list[Record]:
    holds = [variable(f"reserve_{resource}", f"var(reserve_{resource}) + var(tech_reserve_{resource})") for resource in RESOURCES]
    if values and int(values.get("issue_tech_order", 0)) == 1:
        return holds
    return holds + tech_buy(values)
