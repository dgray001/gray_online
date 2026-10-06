#!/usr/bin/env python3
"""Searches a sim debug.log and decodes what it finds: AI orders, script variables, turn snapshots, engine order batches."""
from __future__ import annotations

import argparse
import json
import math
import re
from collections import Counter
from pathlib import Path

CONFIG = Path(__file__).resolve().parents[2] / "config"
ORDER_TYPES = ["None", "MoveSpace", "MoveZone", "Gather", "Build", "Repair", "Renew", "AttackSpace", "AttackZone",
               "AttackUnit", "AttackBuilding", "AutoAttackUnit", "AutoAttackBuilding", "Garrison", "Ungarrison", "DeleteUnit",
               "BuildingCreate", "BuildingResearch", "BuildingDelete", "BuildingAttackUnit", "BuildingAttackBuilding",
               "BuildingAutoAttackUnit", "BuildingAutoAttackBuilding", "CancelOrder", "CancelFoundation", "BuyMercenary"]


def nat_to_int(n: int) -> int:
    return n // 2 if n % 2 == 0 else -((n + 1) // 2)


def unpair(z: int) -> tuple[int, int]:
    """Inverts backend util.Pair (Cantor pairing extended to negatives)."""
    w = int((math.sqrt(8 * z + 1) - 1) / 2)
    y = z - (w * w + w) // 2
    return nat_to_int(w - y), nat_to_int(y)


def describe_zone(key: int) -> str:
    space_key, zone_key = unpair(key)
    return f"{unpair(space_key)}/{unpair(zone_key)}"


def load_names(file: str, id_key: str) -> dict[int, str]:
    return {entry[id_key]: entry["display_name"] for entry in json.loads((CONFIG / file).read_text())}


NAMES = {"building": load_names("buildings.json", "building_id"), "unit": load_names("units.json", "unit_id"),
         "tech": load_names("techs.json", "tech_id")}


def name_of(kind: str, id: int) -> str:
    return f"{NAMES[kind].get(id, '?')}#{id}"


ZONE_TARGETS = {2, 3, 8, 24}
INTERNAL_ID_TARGETS = {5, 6, 9, 10, 13}


def describe_target(order_type: int, target: int) -> str:
    if order_type in ZONE_TARGETS:
        return describe_zone(target)
    if order_type in INTERNAL_ID_TARGETS:
        return f"id {target}"
    if order_type in (1, 7):
        return str(unpair(target))
    if order_type in (4, 25):
        item, zone = unpair(target)
        return f"{name_of('building' if order_type == 4 else 'unit', item)} at {describe_zone(zone)}"
    return {16: lambda: name_of("unit", target), 17: lambda: name_of("tech", target), 23: lambda: f"order {target}"}.get(order_type, lambda: "")()


AI_ORDER = re.compile(r"\{Subjects:\[([\d ]*)\] OrderType:(\d+) TargetID:(-?\d+) ClearPreviousOrders:(\w+)\}")
ENGINE_ORDER = re.compile(r"\{(\d+) \[([\d ]*)\] (\d+) (-?\d+) (\w+)\}")


def describe_order(subjects: str, order_type: int, target: int, clear: str) -> str:
    who = ",".join(subjects.split()) or "player"
    return f"{who}: {ORDER_TYPES[order_type]} {describe_target(order_type, target)}{' (clear)' if clear == 'true' else ''}".rstrip()


def decode_orders(payload: str, engine: bool) -> list[str]:
    if engine:
        return [describe_order(m[1], int(m[2]), int(m[3]), m[4]) for m in ENGINE_ORDER.findall(payload)]
    return [describe_order(m[0], int(m[1]), int(m[2]), m[3]) for m in AI_ORDER.findall(payload)]


def at(location: dict) -> str:
    space, zone = location["Space"], location["Zone"]
    return f"({space['X']}, {space['Y']})/({zone['X']}, {zone['Y']})"


def describe_foundation(f: dict) -> str:
    state = "planned" if f["Planned"] else "building"
    return f"{name_of('building', f['BuildingID'])} at {at(f['Location'])} {state} builders={f['Builders']}"


def summarize_turn(data: dict) -> list[str]:
    pop, res = data["population"], data["resources"]
    units = Counter(name_of("unit", u["UnitID"]) for u in data["units"])
    buildings = Counter(name_of("building", b["BuildingID"]) + ("*" if b["UnderConstruction"] else "") for b in data["buildings"])
    enemies = Counter(name_of("unit", u["UnitID"]) for u in data.get("enemy_units") or [])
    return [f"pop {pop['current']}/{pop['limit']}  food {res['food']:.0f} wood {res['wood']:.0f} stone {res['stone']:.0f} gold {res['gold']:.0f}",
            f"units {dict(units)}  buildings {dict(buildings)} (* = under construction)",
            f"foundations {[describe_foundation(f) for f in data.get('foundations') or []]}", f"enemy units visible {dict(enemies)}"]


def select_vars(data: dict, names: list[str]) -> list[str]:
    if not names:
        return [f"{len(data)} script variables; pass --vars name,name to show some"]
    return [", ".join(f"{name}={data.get(name, '(unset)')}" for name in names)]


AI_LINE = re.compile(r"ai (ai-\d+): (submitting orders|vars|new turn data) (.*)$")
ENGINE_LINE = re.compile(r"Executing submit orders for: (\d+) (.*)$")
TIMESTAMP = re.compile(r"^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} ")


def decode_line(line: str, var_names: list[str]) -> list[str]:
    ai = AI_LINE.search(line)
    if ai and ai[2] == "submitting orders":
        return decode_orders(ai[3], engine=False)
    if ai:
        data = json.loads(ai[3])
        return summarize_turn(data) if ai[2] == "new turn data" else select_vars(data, var_names)
    engine = ENGINE_LINE.search(line)
    if engine:
        return [f"player {engine[1]} submitted:", *decode_orders(engine[2], engine=True)]
    return [TIMESTAMP.sub("", line)]


def label_lines(lines: list[str]) -> list[tuple[str, int, str]]:
    """(label, turn, ai) per line: AI lines carry that AI's own turn, other lines the latest turn reached."""
    ai_turns: Counter[str] = Counter()
    latest, labels = 0, []
    for line in lines:
        match = AI_LINE.search(line)
        if not match:
            labels.append((f"engine after t{latest}", latest, ""))
            continue
        ai_turns[match[1]] += match[2] == "new turn data"
        latest = max(latest, ai_turns[match[1]])
        labels.append((f"{match[1]} t{ai_turns[match[1]]}", ai_turns[match[1]], match[1]))
    return labels


def parse_turns(spec: str) -> tuple[int, int]:
    low, _, high = spec.partition("-")
    return int(low or 0), int(high or low or 10**9)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log_file", type=Path)
    parser.add_argument("query", help="case-insensitive substring; empty matches every line")
    parser.add_argument("--context", type=int, default=0, help="lines of context around each match")
    parser.add_argument("--ai", default="", help="only this AI's lines, e.g. ai-0 (engine lines always match)")
    parser.add_argument("--turns", default="", help="turn range such as 5-12, or a single turn")
    parser.add_argument("--vars", default="", help="script variables to show on 'vars' lines, comma separated")
    parser.add_argument("--show", default="", help="regex; only decoded lines matching it are printed (e.g. 'Build|Garrison')")
    parser.add_argument("--raw", action="store_true", help="print the original line instead of decoding it")
    args = parser.parse_args()
    lines = args.log_file.read_text().splitlines()
    labels = label_lines(lines)
    low, high = parse_turns(args.turns)
    var_names = [name for name in args.vars.split(",") if name]
    for i, line in enumerate(lines):
        _, turn, ai = labels[i]
        if args.query.lower() not in line.lower() or not low <= turn <= high or (args.ai and ai and ai != args.ai):
            continue
        for j in range(max(0, i - args.context), min(len(lines), i + args.context + 1)):
            texts = [lines[j].strip()] if args.raw else decode_line(lines[j], var_names)
            texts = [text for text in texts if re.search(args.show, text, re.IGNORECASE)]
            if texts:
                print(f"{'>>' if j == i else '  '} [{labels[j][0]}]", *(f"\n      {text}" for text in texts), sep="")
        if args.context:
            print("-" * 40)


if __name__ == "__main__":
    main()
