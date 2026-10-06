"""Per-tick combat of one turn in a debug log: each target's attackers, damage, health after and overkill, and a per-player summary.
usage: python3 tools/attack_trace.py outputs/<name>/debug.log <turn> [<turn> ...]"""
from __future__ import annotations

import re
import sys
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path

HIT = re.compile(r"combat turn=(\d+) tick=(\d+): (\d+) [ub](\d+) \(player (\d), stamina (\d+)\) hits (\d+) [ub](\d+) \(player (\d)\) for ([\d.]+)")
HEALTH = re.compile(r"health turn=(\d+) tick=(\d+): (\d+) u\d+ before=([\d.-]+) delta=([\d.-]+)")


@dataclass
class Hit:
    tick: int
    attacker: int
    attacker_kind: int
    attacker_player: int
    stamina: int
    target: int
    target_kind: int
    target_player: int
    damage: float


Health = dict[tuple[int, int], tuple[float, float]]


def parse_turn(lines: list[str], turn: int) -> tuple[list[Hit], Health]:
    hits: list[Hit] = []
    health: Health = {}
    for line in lines:
        if hit := HIT.search(line):
            if int(hit[1]) == turn:
                hits.append(Hit(int(hit[2]), int(hit[3]), int(hit[4]), int(hit[5]), int(hit[6]), int(hit[7]), int(hit[8]), int(hit[9]), float(hit[10])))
        elif change := HEALTH.search(line):
            if int(change[1]) == turn:
                health[(int(change[2]), int(change[3]))] = (float(change[4]), float(change[5]))
    return hits, health


def overkill(tick: int, target: int, health: Health) -> float | None:
    before, delta = health.get((tick, target), (0.0, 0.0))
    return -(before + delta) if (tick, target) in health and before + delta <= 0 else None


def target_line(tick: int, target: int, hits: list[Hit], health: Health) -> str:
    before, delta = health.get((tick, target), (None, None))
    after = "?" if before is None else f"{max(0.0, before + delta):.3f}"
    attackers = ", ".join(f"{h.attacker}(u{h.attacker_kind},st{h.stamina})={h.damage:.2f}" for h in sorted(hits, key=lambda h: h.attacker))
    dead = overkill(tick, target, health)
    ending = "" if dead is None else f"  DEAD overkill={dead:.2f}"
    return f"  target {target} (player {hits[0].target_player}, unit_id {hits[0].target_kind}) hp_before={before} incoming={sum(h.damage for h in hits):.2f} attackers={len(hits)} [{attackers}] -> hp_after={after}{ending}"


def side_line(turn: int, tick: int, player: int, hits: list[Hit], health: Health) -> str:
    by_target: dict[int, list[Hit]] = defaultdict(list)
    for hit in hits:
        by_target[hit.target].append(hit)
    damage = sum(h.damage for h in hits)
    wasted = 0.0
    killed: Counter[int] = Counter()
    for target, group in by_target.items():
        if (excess := overkill(tick, target, health)) is not None:
            wasted += excess
            killed[group[0].target_kind] += 1
    most_hit = sorted(by_target.items(), key=lambda item: (-len(item[1]), item[0]))[:4]
    focus = ", ".join(f"{target}<-{len(group)}" for target, group in most_hit)
    return f"  t{turn} tick {tick} player {player}: {len(hits)} hits on {len(by_target)} targets, dmg {damage:.1f}, overkill {wasted:.1f} ({100 * wasted / max(damage, 1e-9):.0f}%), kills {sum(killed.values())} {dict(killed)}; most hit: {focus}"


def main() -> None:
    lines = Path(sys.argv[1]).read_text().splitlines()
    for turn in map(int, sys.argv[2:]):
        hits, health = parse_turn(lines, turn)
        for tick in sorted({h.tick for h in hits}):
            tick_hits = [h for h in hits if h.tick == tick]
            print(f"\n== turn {turn} tick {tick}")
            for target in sorted({h.target for h in tick_hits}):
                print(target_line(tick, target, [h for h in tick_hits if h.target == target], health))
            for player in sorted({h.attacker_player for h in tick_hits}):
                print(side_line(turn, tick, player, [h for h in tick_hits if h.attacker_player == player], health))


if __name__ == "__main__":
    main()
