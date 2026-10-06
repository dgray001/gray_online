"""Compares two sim eval runs game by game (same seeds and seats) from their results.json files.
usage: python3 tools/compare.py outputs/eval outputs/eval_variant [--idle-until 40]
The subject is scenario player 0 in both runs."""
import argparse
import json
from collections import Counter
from pathlib import Path
from typing import Any

OUTCOMES = ("defeat", "crush", "winning", "ahead", "behind", "losing", "crushed", "defeated", "error")
RANK = {name: len(OUTCOMES) - 2 - index for index, name in enumerate(OUTCOMES)} | {"error": -1}


def load(path: Path) -> dict[str, Any]:
    return json.loads((path / "results.json" if path.is_dir() else path).read_text())


def subject_seat(game: dict[str, Any]) -> int:
    return game["Seats"].index(0)


def outcome(game: dict[str, Any]) -> str:
    result, seat = game["Result"], subject_seat(game)
    if result["Error"]:
        return "error"
    return result["Outcomes"][seat] if result.get("Outcomes") else "error"


def idle_through(game: dict[str, Any], turn: int) -> int:
    seat = subject_seat(game)
    turns = game["Result"]["Game"]["Metrics"]["Turns"]
    return sum(p["VillagerStamina"]["Wasted"] + p["VillageCenterStamina"]["Wasted"]
               for t in turns if t["Turn"] <= turn for p in [t["Players"][seat]])


def early_stats(game: dict[str, Any], turn: int) -> tuple[int, float, int]:
    """Villagers, total income and military units of the subject at the given turn (military from the nearest 5-turn snapshot)"""
    seat = subject_seat(game)
    result = game["Result"]
    records = [t["Players"][seat] for t in result["Game"]["Metrics"]["Turns"] if t["Turn"] <= turn]
    snapshot = min(result["Timeline"], key=lambda s: abs(s[0]["Turn"] - turn))[seat]
    military = sum(count for unit, count in snapshot["Units"].items() if unit != "1")
    return records[-1]["Villagers"], sum(sum(p["Income"]) for p in records), military


def seed_of(game: dict[str, Any]) -> int:
    return game["Result"]["Seed"]


def totals_line(label: str, counts: Counter[str]) -> str:
    return f"{label:14s} " + " ".join(f"{name}={counts[name]:3d}" for name in OUTCOMES)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("base", type=Path)
    parser.add_argument("variant", type=Path)
    parser.add_argument("--idle-until", type=int, help="Also flag games whose villager/VC idle was 0 through this turn in base but not in variant")
    parser.add_argument("--list-outcome", choices=list(OUTCOMES), help="Print the variant's games with this outcome (bucket, seed, map, seat order)")
    parser.add_argument("--count-unit", type=int, help="Print the subject's count of this unit id at the final snapshot, base and variant, per game that differs")
    parser.add_argument("--early-until", type=int, help="Also flag games where villagers, total income or military at this turn is lower than in base")
    args = parser.parse_args()
    base, variant = load(args.base), load(args.variant)
    changes, idle_flags, early_flags, unit_rows, listed = [], [], [], [], []
    all_base, all_variant = Counter(), Counter()
    for base_sim, variant_sim in zip(base["Sims"], variant["Sims"]):
        base_counts, variant_counts = Counter(), Counter()
        for index, (a, b) in enumerate(zip(base_sim["Games"], variant_sim["Games"])):
            before, after = outcome(a), outcome(b)
            base_counts[before] += 1
            variant_counts[after] += 1
            seed = a["Result"]["Seed"]
            if before != after:
                changes.append(("WORSE" if RANK[after] < RANK[before] else "BETTER", base_sim["Name"], index, seed, before, after))
            if args.idle_until and not a["Result"]["Error"] and not b["Result"]["Error"]:
                if idle_through(a, args.idle_until) == 0 and idle_through(b, args.idle_until) > 0:
                    idle_flags.append((base_sim["Name"], index, seed, idle_through(b, args.idle_until)))
            if args.list_outcome and after == args.list_outcome:
                listed.append((base_sim["Name"], variant_sim["Map"], index, seed_of(b), b["Seats"]))
            if args.count_unit and a["Result"]["Timeline"] and b["Result"]["Timeline"]:
                count = lambda g: g["Result"]["Timeline"][-1][subject_seat(g)]["Units"].get(str(args.count_unit), 0)
                if count(a) != count(b):
                    unit_rows.append((base_sim["Name"], index, a["Result"]["Seed"], count(a), count(b)))
            if args.early_until and not a["Result"]["Error"] and not b["Result"]["Error"] and a["Result"]["Timeline"] and b["Result"]["Timeline"]:
                before_stats, after_stats = early_stats(a, args.early_until), early_stats(b, args.early_until)
                if any(y < x for x, y in zip(before_stats, after_stats)):
                    early_flags.append((base_sim["Name"], index, seed, before_stats, after_stats))
        print(f"{base_sim['Name']}\n  {totals_line('base', base_counts)}\n  {totals_line('variant', variant_counts)}")
        all_base.update(base_counts)
        all_variant.update(variant_counts)
    print(f"\n{totals_line('base overall', all_base)}\n{totals_line('variant overall', all_variant)}")
    for direction, bucket, index, seed, before, after in sorted(changes):
        print(f"  {direction:6s} {bucket} game {index} seed {seed}: {before} -> {after}")
    print(f"{sum(c[0] == 'WORSE' for c in changes)} worse, {sum(c[0] == 'BETTER' for c in changes)} better")
    for bucket, index, seed, wasted in idle_flags:
        print(f"  IDLE   {bucket} game {index} seed {seed}: 0 -> {wasted} stamina wasted through turn {args.idle_until}")
    if args.idle_until:
        print(f"{len(idle_flags)} games lost a zero-idle early economy")
    for bucket, index, seed, before_stats, after_stats in early_flags:
        print(f"  EARLY  {bucket} game {index} seed {seed}: (villagers, income, military) {before_stats} -> {after_stats}")
    for bucket, game_map, index, seed, seats in listed:
        print(f"  LISTED {args.list_outcome}: bucket {bucket} map {game_map} game {index} seed {seed} seats {seats}")
    for bucket, index, seed, before_count, after_count in unit_rows:
        print(f"  UNIT {args.count_unit}  {bucket} game {index} seed {seed}: {before_count} -> {after_count}")
    if args.early_until:
        print(f"{len(early_flags)} games have a lower villager count, income or military at turn {args.early_until}")


if __name__ == "__main__":
    main()
