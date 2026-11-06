from __future__ import annotations

from typing import Any
from common import Record
from raid_response import bounded_raids
from reserves import reserve_credit, spending
from shelter import shelter


def substitute(value: Any, replacements: dict[str, str]) -> Any:
    if isinstance(value, str):
        for old, new in replacements.items():
            value = value.replace(f"var(tune_{old})", f"var(tune_{new})")
        return value
    if isinstance(value, dict):
        return {key: substitute(child, replacements) for key, child in value.items()}
    if isinstance(value, list):
        return [substitute(child, replacements) for child in value]
    return value


def apply_policies(model: Record) -> None:
    settings = {action["name"].removeprefix("tune_"): action["value"] for action in model["rules"][0]["then"]}
    shelter(model)
    bounded_raids(model, settings)
    spending(model)
    model.update(substitute(model, dict(policy_margin="refine_reserve_margin", policy_batches="refine_reserve_batches")))
    if settings["refine_queued_credit"]:
        reserve_credit(model)
