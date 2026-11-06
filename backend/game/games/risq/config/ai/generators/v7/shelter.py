from __future__ import annotations

from common import Record, aggregate, selection
from policy_common import condition, objects, unidentified


def exposure(role: str) -> Record:
    return aggregate("count", selection("units", dict(in_set="enemy_units", unit_types=["military"], **{
        "if": f"distance({role}, item, 1) <= var(item.attack_range) + var(tune_policy_exposure)"})))


def shelter(model: Record) -> None:
    for action in list(objects(model)):
        if action.get("action") != "select":
            continue
        if action.get("into", {}).get("set") == "threatened_villagers":
            selector, role = action["pick"], "target"
        elif any("garrison" in verb for verb in action.get("issue", [])):
            selector, role = action["actors"], "actor"
        else:
            continue
        selector.setdefault("let", {}).update(policy_exposed=exposure(role), policy_unknown=unidentified(role))
        condition(selector, f"var({role}.policy_exposed) + var({role}.policy_unknown) > 0")
