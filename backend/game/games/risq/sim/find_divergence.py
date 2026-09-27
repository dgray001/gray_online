#!/usr/bin/env python3
"""Finds the first turn/tick where two games in a -debug run diverge, via state_hash lines.

Usage: find_divergence.py <debug.log> <game_index_a> <game_index_b>

Games are 0-indexed in log order (matches results.json). Prints the tick-tagged
events that differ at the first mismatched hash; unrelated same-tick events can
interleave in either order, so events are compared as multisets.
"""
import re
import sys
from collections import Counter

STATE_HASH = re.compile(r"state_hash turn=(\d+) tick=(\w+) hash=(\w+)")
TICK_EVENT = re.compile(r"(combat tick=\d+:|Foundation started:|Construction complete:|Unit created:|Moving unit )")
TIMESTAMP = re.compile(r"^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} ")

Checkpoint = tuple[str, str, str, list[str]]


def parse_games(path: str) -> list[list[Checkpoint]]:
    games: list[list[Checkpoint]] = []
    events: list[str] = []
    with open(path) as f:
        for raw_line in f:
            line = TIMESTAMP.sub("", raw_line).strip()
            if TICK_EVENT.match(line):
                events.append(line)
                continue
            m = STATE_HASH.search(line)
            if not m:
                continue
            turn, tick, digest = m.groups()
            if turn == "0" and tick == "end":
                games.append([])
            games[-1].append((turn, tick, digest, events))
            events = []
    return games


def main() -> None:
    if len(sys.argv) != 4:
        print(__doc__)
        sys.exit(1)
    games = parse_games(sys.argv[1])
    index_a, index_b = int(sys.argv[2]), int(sys.argv[3])
    if max(index_a, index_b) >= len(games):
        print(f"log only has {len(games)} games")
        sys.exit(1)
    game_a, game_b = games[index_a], games[index_b]
    for (turn_a, tick_a, hash_a, events_a), (turn_b, tick_b, hash_b, events_b) in zip(game_a, game_b):
        if (turn_a, tick_a) != (turn_b, tick_b) or hash_a != hash_b:
            print(f"FIRST DIVERGENCE: A at turn={turn_a} tick={tick_a}, B at turn={turn_b} tick={tick_b}")
            count_a, count_b = Counter(events_a), Counter(events_b)
            for label, only in (("A", count_a - count_b), ("B", count_b - count_a)):
                print(f"  only in {label}:")
                for event, n in sorted(only.items()):
                    print(f"    {n}x {event}")
            if count_a == count_b:
                print("  events identical; divergence is in state no tick event logs")
            return
    if len(game_a) != len(game_b):
        print(f"IDENTICAL for {min(len(game_a), len(game_b))} checkpoints, then lengths differ: A={len(game_a)} B={len(game_b)}")
        return
    print(f"IDENTICAL across all {len(game_a)} checkpoints")


if __name__ == "__main__":
    main()
