#!/usr/bin/env python3
"""Aggregates per-rule select and loop step counts from a sim debug log; the sim only writes them with -debug -log-steps."""
from __future__ import annotations

import argparse
import re
from collections import defaultdict
from pathlib import Path

LINE = re.compile(r"ai (\S+): rule (\d+) steps select=(\d+) loop=(\d+)")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log_file", type=Path)
    parser.add_argument("--ai", default="", help="only this AI, e.g. ai-0")
    parser.add_argument("--top", type=int, default=20, help="rules to show per AI; zero means all")
    args = parser.parse_args()
    stats: dict[str, dict[int, list[int]]] = defaultdict(lambda: defaultdict(lambda: [0, 0, 0]))
    for line in args.log_file.open():
        match = LINE.search(line)
        if match is None or (args.ai and match.group(1) != args.ai):
            continue
        rule, select, loop = int(match.group(2)), int(match.group(3)), int(match.group(4))
        entry = stats[match.group(1)][rule]
        entry[0] += select
        entry[1] += loop
        entry[2] += 1
    if not stats:
        print(f"no rule step lines in {args.log_file}; rerun the sim with -debug -log-steps")
    for ai, rules in sorted(stats.items()):
        print(f"{ai}: {sum(entry[2] for entry in rules.values())} evaluations")
        print(f"{'rule':>4} {'select':>10} {'loop':>8} {'evals':>6} {'select/eval':>12}")
        ranked = sorted(rules.items(), key=lambda item: item[1][0] + item[1][1], reverse=True)
        for rule, (select, loop, evals) in ranked[: args.top or None]:
            print(f"{rule:4d} {select:10d} {loop:8d} {evals:6d} {select / evals:12.1f}")


if __name__ == "__main__":
    main()
