"""Space-changing moves of units in an id range in a debug log, grouped by unit (in sim_army ids 1-30 are player 0, 31-60 player 1).
usage: python3 tools/unit_paths.py outputs/<name>/debug.log <min id> <max id> [<first turn> <last turn>]"""
import re
import sys
from collections import defaultdict
from pathlib import Path

MOVE = re.compile(r"Moving unit (\d+) (\w+) Infantry to zone\(.*?\) in space \((-?\d+), (-?\d+)\) tick=(\d+) turn=(\d+) .*from=\((-?\d+), (-?\d+)\)")


def main() -> None:
    lines = Path(sys.argv[1]).read_text().splitlines()
    low, high = int(sys.argv[2]), int(sys.argv[3])
    first, last = (int(sys.argv[4]), int(sys.argv[5])) if len(sys.argv) > 5 else (0, 10**9)
    paths: dict[int, list[str]] = defaultdict(list)
    types: dict[int, str] = {}
    for line in lines:
        move = MOVE.search(line)
        if not move or move.group(3, 4) == move.group(7, 8) or not first <= int(move.group(6)) <= last or not low <= int(move.group(1)) <= high:
            continue
        unit = int(move.group(1))
        types[unit] = move.group(2)
        paths[unit].append(f"t{move.group(6)}.{move.group(5)} ({move.group(7)},{move.group(8)})->({move.group(3)},{move.group(4)})")
    for unit in sorted(paths):
        print(f"unit {unit} {types[unit]}: " + "  ".join(paths[unit][:12]) + (f"  ... ({len(paths[unit])} moves)" if len(paths[unit]) > 12 else ""))
    print(f"{len(paths)} units with ids {low}-{high} changed space in turns {first}-{last}")


if __name__ == "__main__":
    main()
