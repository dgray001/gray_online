from __future__ import annotations

from collections import Counter
from typing import Any

from tool_search import DecodedOrder, unpair

POWER_FIELDS = {101: "blunt", 102: "pierce", 103: "heavy", 401: "siege"}
MIN_COMMIT_SHARE = 0.5


def position(item: dict[str, Any]) -> tuple[int, int]:
    space = item["Location"]["Space"]
    return space["X"], space["Y"]


def distance(a: tuple[int, int], b: tuple[int, int]) -> int:
    dx, dy = a[0] - b[0], a[1] - b[1]
    return max(abs(dx), abs(dy), abs(dx + dy))


def power(unit: dict[str, Any], variables: dict[str, float]) -> float:
    field = POWER_FIELDS.get(unit["UnitID"])
    return variables["tune_power_" + field] if field else 0.0


def destination(kind: int, target: int, data: dict[str, Any]) -> tuple[int, int] | None:
    if kind in (1, 7):
        return unpair(target)
    if kind in (2, 8):
        return unpair(unpair(target)[0])
    if kind in (9, 10):
        items = data["enemy_units"] if kind == 9 else data["enemy_buildings"]
        return next((position(item) for item in items if item["InternalID"] == target), None)
    return None


def reason(variables: dict[str, float], turn: int) -> str:
    if variables.get("army_retreat", 0):
        return "withdrawing"
    if turn <= variables.get("retreat_defend_until", 0):
        return "retreat recovery"
    if not variables.get("assault", 0):
        return "global launch gate"
    if not variables.get("army_core_can_fight", 0) and variables.get("army_in_center", 0) < variables.get("tune_group_share", 1) * variables["army_size"]:
        return "waiting for regroup"
    if variables.get("army_stage", 0):
        return "staging"
    if variables.get("army_rest", 0) <= variables.get("tune_rest_turns", 0) and variables.get("army_dist_to_target", 0) <= variables.get("tune_rest_dist", 0):
        return "resting before attack"
    return "other order/target priority"


def ongoing_destination(unit: dict[str, Any]) -> tuple[int, int] | None:
    order = unit.get("CurrentOrder") or {}
    if order.get("TargetSpace"):
        return order["TargetSpace"]["X"], order["TargetSpace"]["Y"]
    if order.get("TargetZone"):
        space = order["TargetZone"]["Space"]
        return space["X"], space["Y"]
    for field in ("TargetUnit", "TargetBuilding"):
        if order.get(field):
            return position(order[field])
    return None


def approved_targets(data: dict[str, Any], variables: dict[str, float],
                     ratio: float) -> list[dict[str, Any]]:
    center = int(variables["army_center_x"]), int(variables["army_center_y"])
    targets = {position(unit) for unit in data["enemy_units"] if unit["UnitID"] in POWER_FIELDS
               and distance(center, position(unit)) == 1}
    result: list[dict[str, Any]] = []
    margin = variables["tune_assault_odds"] if variables["has_economy"] else variables["tune_assault_odds_noeco"]
    for target in sorted(targets):
        visible = sum(power(unit, variables) for unit in data["enemy_units"]
                      if distance(target, position(unit)) <= 1)
        unknown = sum(max(0, space["UnitCount"] or 0) for space in data["spaces"]
                      if space["Vision"] == 2 and distance(target, (space["Space"]["X"], space["Space"]["Y"])) <= 1)
        towers = sum(building["BuildingID"] in (1, 21, 23) and position(building) == target
                     for building in data["enemy_buildings"])
        threat = visible * variables["enemy_tech_multiplier"] + unknown * variables["unknown_unit_power"] + 3 * towers
        own = variables["army_tactical_power"]
        if threat > 0 and own >= max(ratio, margin) * threat:
            result.append({"target": list(target), "own_power": own, "enemy_power": threat,
                           "power_ratio": own / threat})
    return result


def opportunity_turn(data: dict[str, Any], variables: dict[str, float], batch: list[DecodedOrder],
                     turn: int, ratio: float) -> dict[str, Any] | None:
    if variables.get("army_size", 0) < variables.get("tune_assault_min", 10) or variables.get("army_engaged", 0):
        return None
    targets = approved_targets(data, variables, ratio)
    if not targets:
        return None
    center = int(variables["army_center_x"]), int(variables["army_center_y"])
    nearby = {unit["InternalID"] for unit in data["units"]
              if unit["UnitID"] in POWER_FIELDS and distance(center, position(unit)) <= 1}
    effective = {unit["InternalID"]: ongoing_destination(unit) for unit in data["units"] if unit["InternalID"] in nearby}
    current_orders = {unit["ActiveOrders"][0]["ID"]: unit["InternalID"] for unit in data["units"]
                      if unit["InternalID"] in nearby and unit.get("ActiveOrders")}
    for subjects, kind, target, clear in batch:
        if kind == 23 and target in current_orders:
            effective[current_orders[target]] = None
            continue
        to = destination(kind, target, data)
        for uid in set(subjects) & nearby:
            if clear or effective.get(uid) is None:
                effective[uid] = to
    committed: Counter[tuple[int, int]] = Counter(to for to in effective.values() if to is not None)
    minimum = max(1, variables["army_size"] * MIN_COMMIT_SHARE)
    targets = [target for target in targets if committed[tuple(target["target"])] < minimum]
    if not targets:
        return None
    chosen = int(variables.get("army_target_x", 0)), int(variables.get("army_target_y", 0))
    if chosen != center and committed[chosen] >= minimum:
        return None
    target = max(targets, key=lambda item: (item["enemy_power"], item["power_ratio"]))
    return {**target, "turn": turn, "reason": reason(variables, turn),
            "army_size": variables["army_size"], "army_in_center": variables.get("army_in_center", 0),
            "go": variables.get("go", 0), "enemy_strength": variables.get("enemy_strength", 0),
            "home_power": variables.get("home_military_power", 0)}


def audit_opportunities(frames: dict[tuple[str, int], dict], variables: dict[tuple[str, int], dict],
                        batches: dict[tuple[str, int], list[DecodedOrder]], seat: int,
                        ratio: float) -> list[dict[str, Any]]:
    ai = f"ai-{seat}"
    windows: list[dict[str, Any]] = []
    for (who, turn), data in sorted(frames.items()):
        if who != ai or (who, turn) not in batches:
            continue
        candidate = opportunity_turn(data, variables[(who, turn)], batches[(who, turn)], turn, ratio)
        if candidate is None:
            continue
        if windows and windows[-1]["target"] == candidate["target"] and windows[-1]["reason"] == candidate["reason"] and windows[-1]["last_turn"] == turn - 1:
            windows[-1]["last_turn"] = turn
            windows[-1]["turns"] += 1
        else:
            windows.append({**candidate, "first_turn": turn, "last_turn": turn, "turns": 1})
    return [window for window in windows if window["turns"] >= 2]
