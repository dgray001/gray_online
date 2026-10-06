"""Turn-by-turn story of one AI in a debug log: our units and the enemy per space, army variables, grouped orders.
usage: python3 tools/army_story.py outputs/<name>/debug.log ai-<seat> <first turn> <last turn> [var,var,...]"""
from __future__ import annotations

import json
import re
import sys
from collections import Counter
from pathlib import Path

from tool_search import AI_LINE, decode_orders

DEFAULT_VARS = ["army_fwd", "fb_x", "fb_y", "army_stage", "army_target_score", "army_target_x", "army_target_y", "a_outnumbered", "assault"]


def spaces_of(units: list[dict]) -> dict[tuple[int, int], int]:
    counts = Counter((u["Location"]["Space"]["X"], u["Location"]["Space"]["Y"]) for u in units)
    return dict(sorted(counts.items()))


def snapshot_line(data: dict) -> str:
    ours = spaces_of(data["units"])
    seen = spaces_of(data.get("enemy_units") or [])
    counts = {(s["Space"]["X"], s["Space"]["Y"]): s["UnitCount"] for s in data["spaces"] if s["UnitCount"] and (s["Space"]["X"], s["Space"]["Y"]) not in seen and (s["Space"]["X"], s["Space"]["Y"]) not in ours}
    buildings = spaces_of(data.get("enemy_buildings") or [])
    return f"ours {ours} | enemy seen {seen} counted-only {counts} bldgs {buildings}"


def order_lines(payload: str) -> list[str]:
    grouped: Counter[str] = Counter()
    for line in decode_orders(payload, engine=False):
        who, _, what = line.partition(": ")
        grouped[what] += len(who.split(","))
    return [f"{n}x {what}" for what, n in grouped.most_common(40) if not what.startswith("AttackUnit") or n > 12]


def main() -> None:
    log, ai, first, last = Path(sys.argv[1]), sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
    names = sys.argv[5].split(",") if len(sys.argv) > 5 else DEFAULT_VARS
    turn = 0
    for line in log.read_text().splitlines():
        match = AI_LINE.search(line)
        if not match or match[1] != ai:
            continue
        kind, payload = match[2], match[3]
        if kind == "new turn data":
            turn += 1
            if first <= turn <= last:
                print(f"== t{turn}: " + snapshot_line(json.loads(payload)))
        elif not first <= turn <= last:
            continue
        elif kind == "vars":
            data = json.loads(payload)
            print("   vars: " + ", ".join(f"{n}={round(data[n], 2) if n in data else '-'}" for n in names))
        else:
            print("   orders: " + "; ".join(order_lines(payload)))


if __name__ == "__main__":
    main()
