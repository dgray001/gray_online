from __future__ import annotations

from collections.abc import Iterator
from typing import Any
from common import Record, aggregate, rule, selection


def objects(value: Any) -> Iterator[Record]:
    if isinstance(value, dict):
        yield value
        for child in list(value.values()):
            yield from objects(child)
    elif isinstance(value, list):
        for child in list(value):
            yield from objects(child)


def condition(selector: Record, expression: str) -> None:
    where = selector.setdefault("where", {})
    previous = where.get("if", [])
    where["if"] = (previous if isinstance(previous, list) else [previous]) + [expression]


def insert_before(model: Record, name: str, actions: list[Record]) -> None:
    index = next(i for i, phase in enumerate(model["rules"])
                 if any(item.get("action") == "set_var" and item.get("name") == name for item in objects(phase)))
    model["rules"].insert(index, rule(actions))


def unidentified(role: str) -> Record:
    return aggregate("sum", selection("spaces", dict(in_set="unidentified_spaces", **{
        "if": f"distance({role}, item) <= var(tune_policy_radius)"})), "var(item.unit_count)")
