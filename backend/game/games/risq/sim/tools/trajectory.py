"""Per-turn outcome buckets of one seat in a debug game, read from the turn reports in debug.log.
usage: python3 tools/trajectory.py outputs/<name>/debug.log <seat>
Uses the sim's outcome rules: defeat/defeated, crush/crushed (10x units), winning/losing (2x units, score and land), ahead/behind (score)."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path
from typing import NamedTuple

TURN_REPORT = re.compile(r"turn report turn=(\d+) player=(\d+) metrics=.* report=(\{.*\})$")


class Standing(NamedTuple):
    player: int
    units: int
    score: int
    land: int
    eliminated: bool


def outnumbers(a: Standing, b: Standing, factor: int) -> bool:
    return a.units > 0 and a.units >= factor * b.units


def winning(a: Standing, b: Standing) -> bool:
    return outnumbers(a, b, 2) and a.score > b.score and a.land > b.land


def ahead(a: Standing, b: Standing) -> bool:
    return a.score > b.score or (a.score == b.score and a.player < b.player)


def classify(me: Standing, other: Standing) -> str:
    if other.eliminated != me.eliminated:
        return "defeat" if other.eliminated else "defeated"
    for factor_check, win, lose in ((lambda a, b: outnumbers(a, b, 10), "crush", "crushed"), (winning, "winning", "losing")):
        if factor_check(me, other):
            return win
        if factor_check(other, me):
            return lose
    return "ahead" if ahead(me, other) else "behind"


def read(log: Path) -> dict[int, dict[int, Standing]]:
    """turn -> player -> standing at the end of that turn"""
    reports: dict[int, dict[int, dict]] = {}
    for line in log.read_text().splitlines():
        match = TURN_REPORT.search(line)
        if match:
            reports.setdefault(int(match[1]), {})[int(match[2])] = json.loads(match[3])
    turns: dict[int, dict[int, Standing]] = {}
    for turn, players in reports.items():
        scores = {line["player_id"]: line["now"] for line in next(iter(players.values()))["scores"]}
        turns[turn] = {p: Standing(p, r["population"]["end"], scores[p], r["land"]["held_end"], r["eliminated"]) for p, r in players.items()}
    return turns


def main() -> None:
    seat = int(sys.argv[2])
    turns = read(Path(sys.argv[1]))
    series = [(t, classify(s[seat], s[1 - seat])) for t, s in sorted(turns.items()) if len(s) == 2]
    runs: list[list] = []
    for turn, bucket in series:
        if runs and runs[-1][0] == bucket:
            runs[-1][2] = turn
        else:
            runs.append([bucket, turn, turn])
    print(" -> ".join(f"{b} t{a}" + (f"-{z}" if z != a else "") for b, a, z in runs))
    first = {b: next(t for t, x in series if x == b) for b in dict.fromkeys(b for _, b in series)}
    print("first turn in each bucket:", first)
    flips = [(t0, b0, b1) for (t0, b0), (_, b1) in zip(series, series[1:]) if {b0, b1} <= {"winning", "losing", "crush", "crushed", "defeat", "defeated"} and b0 != b1 and (b0 in {"winning", "crush", "defeat"}) != (b1 in {"winning", "crush", "defeat"})]
    print("flips between the winning side and the losing side:", flips or "none")


if __name__ == "__main__":
    main()
