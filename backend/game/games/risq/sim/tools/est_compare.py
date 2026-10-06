"""One AI's estimate of the enemy army (ePowEst) against the enemy's actual power, per turn, from a debug.log.
usage: python3 tools/est_compare.py outputs/<name>/debug.log <seat of the AI under test> [last turn] [step]"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

POWER = {11: 1, 12: 2, 13: 4}
AI_LINE = re.compile(r"ai (ai-\d): (new turn data|vars) (.*)$")


def actual_power(data: dict) -> float:
    return sum(POWER.get(u["UnitID"], 0) for u in data["units"])


def read(log: Path, seat: int) -> tuple[dict[int, float], dict[int, dict[str, float]]]:
    actual: dict[int, float] = {}
    mine: dict[int, dict[str, float]] = {}
    turns = {0: 0, 1: 0}
    for line in log.read_text().splitlines():
        match = AI_LINE.search(line)
        if not match:
            continue
        who = int(match[1][-1])
        data = json.loads(match[3])
        if match[2] == "new turn data":
            turns[who] += 1
            if who != seat:
                actual[turns[who]] = actual_power(data)
        elif who == seat:
            mine[turns[who]] = data
    return actual, mine


def main() -> None:
    log, seat = Path(sys.argv[1]), int(sys.argv[2])
    last = int(sys.argv[3]) if len(sys.argv) > 3 else 60
    step = int(sys.argv[4]) if len(sys.argv) > 4 else 3
    actual, mine = read(log, seat)
    print("turn | enemy actual | est now | est remembered | our homePow | assault")
    over: list[float] = []
    launch = None
    for turn in range(1, last + 1):
        if turn not in mine or turn not in actual:
            continue
        v = mine[turn]
        est = v.get("ePowEst", 0)
        if 8 <= turn <= 50:
            over.append(est - actual[turn])
        if launch is None and v.get("assault", 0) > 0:
            launch = turn
        if turn % step == 0:
            print(f"{turn:4d} | {actual[turn]:5.0f} | {v.get('ePow', 0):6.1f} | {est:6.1f} | {v.get('homePow', 0):5.1f} | {v.get('assault', 0):.0f}")
    print(f"mean (est - actual) turns 8-50: {sum(over) / max(1, len(over)):+.1f}; worst under {min(over, default=0):+.1f}; first assault turn: {launch}")


if __name__ == "__main__":
    main()
