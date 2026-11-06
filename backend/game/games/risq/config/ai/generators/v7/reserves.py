from __future__ import annotations

from common import RESOURCES, Record, aggregate, selection, variable
from policy_common import condition, insert_before, objects


def observations(model: Record) -> None:
    unit = "101 + 2 * (var(item.building_id) == 23)"
    shortage = " + ".join(f"(gameVar({resource}) < var(tune_policy_batches) * config(unit, cost_{resource}, {unit}))" for resource in RESOURCES)
    idle = selection("buildings", dict(owner="own", building_ids=[22, 23], state="complete", **{
        "if": ["var(item.queue_length) == 0", f"({shortage}) > 0"]}))
    actions = [variable("policy_starved", aggregate("count", idle)),
               variable("policy_pressure", "(var(base_enemy_power) > 0) * (var(base_own_power) < var(tune_policy_margin) * var(base_enemy_power))")]
    insert_before(model, "home_military_power", actions)


def spending(model: Record) -> None:
    observations(model)
    for action in list(objects(model)):
        if action.get("action") != "select" or "actors" not in action:
            continue
        for verb in action.get("issue", []):
            kind, item, essential = "", 0, "0"
            if "research" in verb and verb["research"]["tech_id"] != 1:
                kind, item = "tech", verb["research"]["tech_id"]
            elif "build" in verb and verb["build"]["building_id"] not in (1, 3):
                kind, item = "building", verb["build"]["building_id"]
                essential = "var(headroom) <= 0" if item == 2 else "var(producers_22) == 0" if item == 22 else "0"
            if not kind:
                continue
            funds = " * ".join(f"(gameVar({resource}) - config({kind}, cost_{resource}, {item}) >= var(tune_policy_batches) * config(unit, cost_{resource}, 101))" for resource in RESOURCES)
            gate = f"(var(policy_pressure) == 0) + ({funds}) > 0"
            condition(action["actors"], f"({essential}) + ({gate}) > 0")


def reserve_credit(model: Record) -> None:
    queued = aggregate("count", selection("orders", {"in": dict(order_kind=["create"], item_unit_id=[101, 102, 103])}))
    for action in list(objects(model)):
        if action.get("action") != "select" or "actors" not in action:
            continue
        where = action["actors"].get("where", {})
        clauses = where.get("if", [])
        clauses = clauses if isinstance(clauses, list) else [clauses]
        if not any(isinstance(clause, str) and "var(policy_pressure)" in clause for clause in clauses):
            continue
        action["actors"].setdefault("let", {})["refine_queued"] = queued
        reserve = "max(0, var(tune_refine_reserve_batches) - var(actor.refine_queued))"
        where["if"] = [clause.replace("var(tune_refine_reserve_batches)", reserve) if isinstance(clause, str) else clause for clause in clauses]
