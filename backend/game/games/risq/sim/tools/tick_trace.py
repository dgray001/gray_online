"""Per-tick account of one turn in a debug log: space-changing moves, hits and deaths by player and unit type.
usage: python3 tools/tick_trace.py outputs/<name>/debug.log <turn> [<turn> ...]"""
import re
import sys
from collections import Counter, defaultdict
from pathlib import Path

MOVE = re.compile(r"Moving unit (\d+) (\w+) Infantry to zone\(.*?\) in space \((-?\d+), (-?\d+)\) tick=(\d+) turn=(\d+) .*from=\((-?\d+), (-?\d+)\)")
HIT = re.compile(r"combat turn=(\d+) tick=(\d+): (\d+) [ub](\d+) \(player (\d)[^)]*\) hits (\d+) [ub](\d+) \(player (\d)\) for ([\d.]+)")
HEALTH = re.compile(r"health turn=(\d+) tick=(\d+): (\d+) u(\d+) before=([\d.-]+) delta=([\d.-]+)")
TYPE_NAMES = {"1": "Villager", "11": "Blunt", "12": "Piercer", "13": "Heavy"}


def player_of_units(lines: list[str]) -> dict[str, str]:
    owners: dict[str, str] = {}
    for line in lines:
        hit = HIT.search(line)
        if hit:
            owners[hit.group(3)], owners[hit.group(6)] = hit.group(5), hit.group(8)
    return owners


def trace(lines: list[str], owners: dict[str, str], turn: int) -> None:
    moves: dict[int, Counter[str]] = defaultdict(Counter)
    hits: dict[int, Counter[str]] = defaultdict(Counter)
    damage: dict[int, Counter[str]] = defaultdict(Counter)
    deaths: dict[int, list[str]] = defaultdict(list)
    wasted: dict[int, Counter[str]] = defaultdict(Counter)
    for line in lines:
        move = MOVE.search(line)
        if move and int(move.group(6)) == turn and move.group(3, 4) != move.group(7, 8):
            tick = int(move.group(5))
            moves[tick][f"p{owners.get(move.group(1), '?')} {move.group(2)} ({move.group(7)},{move.group(8)})->({move.group(3)},{move.group(4)})"] += 1
        hit = HIT.search(line)
        if hit and int(hit.group(1)) == turn:
            tick, key = int(hit.group(2)), f"p{hit.group(5)}->p{hit.group(8)}"
            hits[tick][key] += 1
            damage[tick][key] += float(hit.group(9))
        health = HEALTH.search(line)
        if health and int(health.group(1)) == turn and float(health.group(5)) + float(health.group(6)) <= 0:
            tick, victim = int(health.group(2)), owners.get(health.group(3), "?")
            deaths[tick].append(f"p{victim} {TYPE_NAMES.get(health.group(4), health.group(4))}")
            wasted[tick][f"p{1 - int(victim)}->p{victim}"] += -float(health.group(6)) - float(health.group(5))
    print(f"== turn {turn}")
    for tick in sorted(set(moves) | set(hits) | set(deaths)):
        parts = [f"{n}x {k}" for k, n in sorted(moves[tick].items())]
        parts += [f"{k}: {n} hits {damage[tick][k]:.0f} dmg, overkill {100 * wasted[tick][k] / max(damage[tick][k], 1e-9):.0f}%" for k, n in sorted(hits[tick].items())]
        if deaths[tick]:
            parts.append("DEAD " + ", ".join(f"{n}x {k}" for k, n in sorted(Counter(deaths[tick]).items())))
        print(f"  tick {tick:2d}: " + " | ".join(parts))


def main() -> None:
    lines = Path(sys.argv[1]).read_text().splitlines()
    owners = player_of_units(lines)
    for turn in sys.argv[2:]:
        trace(lines, owners, int(turn))


if __name__ == "__main__":
    main()
