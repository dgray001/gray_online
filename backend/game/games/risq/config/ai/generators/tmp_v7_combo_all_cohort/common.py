from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Union

Record = dict[str, Any]
Value = Union[str, int, float, Record]


def variable(name: str, value: Value, persist: bool = False) -> Record:
    return dict(action="set_var", name=name, value=value, persist=persist)


def selection(kind: str, where: Record | None = None, **options: Any) -> Record:
    return dict(kind=kind, where=where or {}, **options)


def aggregate(operation: str, source: Record, of: str | None = None) -> Record:
    return {operation: dict(source, **({"of": of} if of is not None else {}))}


def rule(actions: list[Record], when: Record | None = None) -> Record:
    return dict(when=when or {"always": {}}, then=actions)


def branch(expression: str, actions: list[Record], otherwise: list[Record] | None = None) -> Record:
    condition = {"value_at_least": dict(value=expression, amount=1)}
    return dict(action="if", when=condition, then=actions, **({"else": otherwise} if otherwise is not None else {}))


def store(name: str, source: Record, persist: bool = False) -> Record:
    into = dict(set=name, var=f"n_{name}")
    if persist:
        into["persist"] = True
    return dict(action="select", pick=source, into=into)


def coordinate(name: str) -> Record:
    return dict(x=f"var({name}_x)", y=f"var({name}_y)")


def coordinates(name: str, kind: str) -> list[Record]:
    source = selection(kind, dict(in_set=name))
    return [variable(f"{name}_{field}", aggregate("max", source, f"var(item.{field})"))
            for field in ("x", "y", "zone_x", "zone_y")]


class Definitions:
    def __init__(self, directory: Path) -> None:
        self.units = self.load(directory, "units", "unit_id")
        self.buildings = self.load(directory, "buildings", "building_id")
        self.techs = self.load(directory, "techs", "tech_id")
        self.bonuses = json.loads((directory / "bonuses.json").read_text())

    @staticmethod
    def load(directory: Path, name: str, key: str) -> dict[int, Record]:
        return {int(item[key]): item for item in json.loads((directory / f"{name}.json").read_text())}

    @staticmethod
    def gathered_cost(item: Record) -> float:
        return sum(float(item.get("cost", {}).get(resource, 0)) for resource in ("food", "wood", "stone"))

    def production_rate(self, building_id: int, unit_id: int) -> float:
        return float(self.buildings[building_id]["turn_stamina"]) / self.units[unit_id]["production_stamina"]


POWER_TYPES = {101: "blunt", 102: "pierce", 103: "heavy", 401: "siege"}


def power(role: str = "item", infantry_only: bool = False) -> str:
    return " + ".join(f"(var({role}.unit_id) == {unit}) * var(tune_power_{name})"
                      for unit, name in POWER_TYPES.items() if not infantry_only or unit != 401)


RESOURCES = ("food", "wood", "stone", "gold")


def cost(kind: str, item: int, resource: str) -> str:
    return f"config({kind}, cost_{resource}, {item})"


def total_cost(kind: str, item: int, resources: tuple[str, ...] = RESOURCES) -> str:
    return "(" + " + ".join(cost(kind, item, resource) for resource in resources) + ")"


def whole(expression: str) -> str:
    return f"max(0, round(({expression}) - 0.499999))"


def army_order_available(role: str) -> str:
    return f"(var(army_retreat) == 0) + (1 - inSet(retreat_cohort, {role})) > 0"
