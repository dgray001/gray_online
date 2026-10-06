#!/usr/bin/env python3
"""Finds the first turn/tick where two games in a -debug run diverge, via state_hash lines.

Usage: find_divergence.py <debug.log> [<game_index_a> <game_index_b>]

Games are 0-indexed in log order (matches results.json). Without indexes, every game is
compared against game 0 (for fixed_seed scenarios). Prints the tick-tagged events that
differ at the first mismatched hash; unrelated same-tick events can interleave in either
order, so events are compared as multisets. If the run had RISQ_STATE_DUMP=1, also prints
the differing state entries at that checkpoint.
"""
import re
import sys
from collections import Counter

STATE_HASH = re.compile(r"state_hash turn=(\d+) tick=(\w+) hash=(\w+)")
STATE_DUMP = re.compile(r"state_dump turn=\d+ tick=\w+ (.*)")
TICK_EVENT = re.compile(r"(combat tick=\d+:|Foundation started:|Construction complete:|Unit created:|Moving unit )")
TIMESTAMP = re.compile(r"^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} ")

# turn, tick, hash, events since previous checkpoint, state dump entries
Checkpoint = tuple[str, str, str, list[str], list[str]]


def parse_games(path: str) -> list[list[Checkpoint]]:
    games: list[list[Checkpoint]] = []
    events: list[str] = []
    with open(path) as f:
        for raw_line in f:
            line = TIMESTAMP.sub("", raw_line).strip()
            if TICK_EVENT.match(line):
                events.append(line)
                continue
            dump = STATE_DUMP.search(line)
            if dump and games and games[-1]:
                turn, tick, digest, prior_events, _ = games[-1][-1]
                games[-1][-1] = (turn, tick, digest, prior_events, dump.group(1).split(";"))
                continue
            m = STATE_HASH.search(line)
            if not m:
                continue
            turn, tick, digest = m.groups()
            if turn == "0" and tick == "end":
                games.append([])
            games[-1].append((turn, tick, digest, events, []))
            events = []
    return games


def compare(game_a: list[Checkpoint], game_b: list[Checkpoint]) -> None:
    for (turn_a, tick_a, hash_a, events_a, dump_a), (turn_b, tick_b, hash_b, events_b, dump_b) in zip(game_a, game_b):
        if (turn_a, tick_a) != (turn_b, tick_b) or hash_a != hash_b:
            print(f"FIRST DIVERGENCE: A at turn={turn_a} tick={tick_a}, B at turn={turn_b} tick={tick_b}")
            count_a, count_b = Counter(events_a), Counter(events_b)
            for label, only in (("A", count_a - count_b), ("B", count_b - count_a)):
                print(f"  only in {label}:")
                for event, n in sorted(only.items()):
                    print(f"    {n}x {event}")
            if count_a == count_b:
                print("  events identical; divergence is in state no tick event logs")
            if dump_a or dump_b:
                entries_a, entries_b = Counter(dump_a), Counter(dump_b)
                for label, only in (("A", entries_a - entries_b), ("B", entries_b - entries_a)):
                    print(f"  state only in {label}:")
                    for entry in only:
                        print(f"    {entry}")
            else:
                print("  rerun with RISQ_STATE_DUMP=1 to see differing state entries")
            return
    if len(game_a) != len(game_b):
        print(f"IDENTICAL for {min(len(game_a), len(game_b))} checkpoints, then lengths differ: A={len(game_a)} B={len(game_b)}")
        return
    print(f"IDENTICAL across all {len(game_a)} checkpoints")


def main() -> None:
    if len(sys.argv) not in (2, 4):
        print(__doc__)
        sys.exit(1)
    games = parse_games(sys.argv[1])
    pairs = [(int(sys.argv[2]), int(sys.argv[3]))] if len(sys.argv) == 4 else [(0, i) for i in range(1, len(games))]
    for index_a, index_b in pairs:
        if max(index_a, index_b) >= len(games):
            print(f"log only has {len(games)} games")
            sys.exit(1)
        print(f"game {index_a} vs game {index_b}:")
        compare(games[index_a], games[index_b])


if __name__ == "__main__":
    main()
