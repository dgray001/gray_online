"""Derive economy, military efficiency and fight labels from core RISQ metrics."""

from __future__ import annotations

import argparse
import json
import math
from collections import Counter
from pathlib import Path
from typing import Any

RESOURCES = ("food", "wood", "stone", "gold")
MEMBERSHIPS = ("Units", "Attacking", "NotAttacking", "Moving", "Exhausted", "Garrisoned")
LABELS = ("bad", "neutral", "good")
HEX_NEIGHBORS = ((1, 0), (0, 1), (-1, 1), (-1, 0), (0, -1), (1, -1))


def percentage(numerator: float, denominator: float) -> float | None:
    return 100 * numerator / denominator if denominator else None


def arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("results", type=Path, help="Sim output directory or results.json")
    parser.add_argument("--output", type=Path, help="Default: metrics_analysis.json beside results")
    parser.add_argument("--map", action="append", dest="maps", help="Repeat to select maps")
    parser.add_argument("--ai", action="append", dest="ais", help="Repeat to select AI names")
    parser.add_argument("--from-turn", type=int, default=0, help="First recorded turn, inclusive (default: 0)")
    parser.add_argument("--to-turn", type=int, default=65535, help="Last recorded turn, inclusive")
    parser.add_argument("--seed", action="append", dest="seeds", type=int, help="Repeat to restrict --by-turn to these game seeds")
    parser.add_argument("--at-turn", type=lambda s: [int(n) for n in s.split(",")], help="Print cumulative economy at these turns, e.g. 5,10,15,20,25")
    parser.add_argument("--by-turn", type=int, help="Print a per-game table in blocks of this many turns (production columns need the scenario's \"metrics\": [\"production\"])")
    parser.add_argument("--strength-ratio", type=float, default=1.5, help="Good/bad strength threshold (default: 1.5)")
    args = parser.parse_args()
    if args.from_turn < 0 or args.to_turn < args.from_turn:
        parser.error("Turn range must be nonnegative and ordered")
    if not math.isfinite(args.strength_ratio) or args.strength_ratio <= 1:
        parser.error("Strength ratio must be finite and exceed 1")
    return args


def damage_per_stamina(attacker: dict[str, Any], defender: dict[str, Any]) -> float:
    kinds = attacker.get("attack_type", "none").split("_")
    attack = sum(attacker.get(f"attack_{kind}", 0) for kind in kinds)
    if not attack:
        return 0.0
    defense = sum(defender.get(f"defense_{kind}", 0)
                  * (1 - attacker.get(f"penetration_{kind}", 0) / 100) for kind in kinds)
    reference = attack - defense if defense / attack <= 0.5 else attack * 4 ** (-defense / attack)
    return max(0.1, reference / 10)


def composition(players: list[dict[str, Any]]) -> Counter[str]:
    counts: Counter[str] = Counter()
    for player in players:
        counts.update(player["Military"])
        counts.update(player["Adjacent"])
    return counts


def strength(army: Counter[str], enemy: Counter[str], units: dict[str, Any]) -> float:
    total = 0.0
    for kind, count in army.items():
        unit = units[kind]
        if enemy:
            damage = sum(amount * damage_per_stamina(unit, units[target])
                         for target, amount in enemy.items()) / sum(enemy.values())
        else:
            damage = damage_per_stamina(unit, {})
        total += count * math.sqrt(unit["max_health"] * unit["turn_stamina"] * damage)
    return total


def bases(buildings: list[dict[str, Any]]) -> list[list[tuple[int, int]]]:
    occupied = {tuple(building["Space"]) for building in buildings}
    centers = {tuple(building["Space"]) for building in buildings if building["BuildingId"] == 1}
    components: list[list[tuple[int, int]]] = []
    while occupied:
        first = min(occupied)
        occupied.remove(first)
        connected, pending = {first}, [first]
        for x, y in pending:
            neighbors = {(x + dx, y + dy) for dx, dy in HEX_NEIGHBORS} & occupied
            occupied.difference_update(neighbors)
            connected.update(neighbors)
            pending.extend(sorted(neighbors))
        if not centers or connected & centers:
            components.append(sorted(connected))
    return components


def defending_base(spaces: list[list[int]], components: list[list[tuple[int, int]]]) -> bool:
    fight_spaces = {tuple(space) for space in spaces}
    return any(fight_spaces.intersection(component) for component in components)


def assess(frame: dict[str, Any], player: dict[str, Any],
           units: dict[str, Any], ratio: float) -> dict[str, Any]:
    if "Buildings" not in player:
        raise ValueError("Fight metrics lack building locations; rerun the sim")
    army = composition([player])
    enemies = composition([p for p in frame["Players"] if p["PlayerId"] != player["PlayerId"]])
    own, enemy = strength(army, enemies, units), strength(enemies, army, units)
    player_bases = bases(player["Buildings"] or [])
    defending = defending_base(frame["Spaces"], player_bases)
    label, reason = "neutral", "comparable_strength"
    if own > 0 and own >= ratio * enemy:
        label, reason = "good", "superior_strength"
    elif enemy > 0 and enemy >= ratio * own:
        label = "bad" if defending is False else "neutral"
        reason = "outgunned_defending_base" if defending else "outgunned_away_from_base"
    return {"label": label, "reason": reason, "turn": frame["Turn"], "tick": frame["Tick"],
            "defending_base": defending, "own_strength": own, "enemy_strength": enemy,
            "strength_ratio": own / enemy if enemy else None, "spaces": frame["Spaces"],
            "own_military": dict(army), "enemy_military": dict(enemies), "bases": player_bases,
            "own_inside": player["Military"], "own_adjacent": player["Adjacent"],
            "snapshot": "before_first_local_attack_commitment"}


def resources(turns: list[dict[str, Any]]) -> dict[str, Any]:
    banks = [turn["Stockpile"] for turn in turns if turn["Turn"] > 0]
    values: dict[str, Any] = {"samples": len(banks)}
    for field in ("Stockpile", "Income", "Spent"):
        values[field.lower()] = {}
        for index, name in enumerate(RESOURCES):
            amounts = [turn[field][index] for turn in turns]
            values[field.lower()][name] = sum(amounts) if field != "Stockpile" else {
                "average": sum(bank[index] for bank in banks) / len(banks) if banks else None,
                "peak": max((bank[index] for bank in banks), default=None),
                "final": amounts[-1] if amounts else None}
    return values


def economy(turns: list[dict[str, Any]], field: str) -> dict[str, Any]:
    granted = sum(turn[field]["Granted"] for turn in turns)
    wasted = sum(turn[field]["Wasted"] for turn in turns)
    return {"granted": granted, "wasted": wasted, "idle_pct": percentage(wasted, granted)}


def military(samples: list[dict[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {"tick_samples": len(samples)}
    for field in MEMBERSHIPS:
        sets = [set(sample[field] or []) for sample in samples]
        result[field] = {"unique": len(set().union(*sets)), "observations": sum(map(len, sets)),
                         "peak": max(map(len, sets), default=0)}
    for field in ("AttackStamina", "OverkillStamina", "MoveStamina", "IdleStamina", "OtherStamina"):
        result[field] = sum(sample[field] for sample in samples)
    result["overkill_pct"] = percentage(result["OverkillStamina"], result["AttackStamina"])
    adjacent = [sum(sample["Adjacent"].values()) for sample in samples]
    result["adjacent"] = {"peak": max(adjacent, default=0),
                          "mean": sum(adjacent) / len(adjacent) if adjacent else None}
    result["ready_stationary_not_attacking"] = sum(len(set(sample["NotAttacking"] or [])
        - set(sample["Moving"] or []) - set(sample["Exhausted"] or [])) for sample in samples)
    result["peak_combatants"] = max((sample["Combatants"] for sample in samples), default=0)
    result["peak_health"] = max((sample["Health"] for sample in samples), default=0)
    return result


def engagement(frames: list[dict[str, Any]], seat: int,
               units: dict[str, Any], ratio: float) -> dict[str, Any] | None:
    for frame in frames:
        player = next(p for p in frame["Players"] if p["PlayerId"] == seat)
        if "Engaging" not in player:
            raise ValueError("Fight metrics lack engagement snapshots; rerun the sim")
        if player["Engaging"]:
            assessment = assess(frame, player, units, ratio)
            return {**assessment, "engaging_units": player["Engaging"]}
    return None


def fight_player(frames: list[dict[str, Any]], seat: int,
                 units: dict[str, Any], args: argparse.Namespace,
                 all_frames: list[dict[str, Any]]) -> dict[str, Any]:
    pairs = [(frame, next(p for p in frame["Players"] if p["PlayerId"] == seat)) for frame in frames]
    samples = [player for _, player in pairs]
    witness = engagement(all_frames, seat, units, args.strength_ratio)
    return {"player_id": seat, "involved": any(p["Combatants"] or p["Engaging"] for p in samples),
            "classification": witness,
            "engagement_in_window": witness is not None and in_window(witness["turn"], args),
            "military": military(samples)}


def fight_report(fight: dict[str, Any], frames: list[dict[str, Any]], seats: dict[int, str],
                 units: dict[str, Any], args: argparse.Namespace) -> dict[str, Any]:
    players = []
    for seat, name in seats.items():
        player = fight_player(frames, seat, units, args, fight["Ticks"])
        players.append({"name": name, **player})
    return {"fight_id": fight["FightId"], "first_tick": [frames[0]["Turn"], frames[0]["Tick"]],
            "last_tick": [frames[-1]["Turn"], frames[-1]["Tick"]], "tick_samples": len(frames),
            "peak_spaces": max(len(frame["Spaces"]) for frame in frames),
            "spaces": sorted({tuple(space) for frame in frames for space in frame["Spaces"]}),
            "players": players}


def in_window(turn: int, args: argparse.Namespace) -> bool:
    return args.from_turn <= turn <= args.to_turn


def analyze_game(sim: dict[str, Any], suite: dict[str, Any], units: dict[str, Any],
                 args: argparse.Namespace) -> dict[str, Any]:
    result = suite["Result"]
    report: dict[str, Any] = {"sim": sim["Name"], "map": sim["Map"], "seed": result["Seed"]}
    if result["Error"]:
        return {**report, "error": result["Error"]}
    game = result["Game"]
    metrics = game.get("Metrics")
    if not metrics or not metrics["Turns"]:
        raise ValueError(f"{sim['Name']} seed {result['Seed']}: no core metrics; rerun the sim")
    seats = {seat: sim["Stats"]["Players"][index]["Name"] for seat, index in enumerate(suite["Seats"])}
    seats = {seat: name for seat, name in seats.items() if not args.ais or name in args.ais}
    report.update({"turn_number": game["TurnNumber"], "players": [], "fights": []})
    turns = [{**turn, "Players": [p for p in turn["Players"] if p["PlayerId"] in seats]}
             for turn in metrics["Turns"] if in_window(turn["Turn"], args)]
    report["turns"] = turns
    for fight in metrics["Fights"] or []:
        frames = [frame for frame in fight["Ticks"] if in_window(frame["Turn"], args)]
        if frames:
            report["fights"].append(fight_report(fight, frames, seats, units, args))
    for seat, name in seats.items():
        player_turns = [{"Turn": turn["Turn"], **p} for turn in turns
                        for p in turn["Players"] if p["PlayerId"] == seat]
        samples = [p for fight in metrics["Fights"] or [] for frame in fight["Ticks"]
                   if in_window(frame["Turn"], args) for p in frame["Players"] if p["PlayerId"] == seat]
        labels = Counter(p["classification"]["label"] for fight in report["fights"]
                         for p in fight["players"] if p["player_id"] == seat and p["engagement_in_window"])
        fight_count = sum(p["involved"] for fight in report["fights"]
                          for p in fight["players"] if p["player_id"] == seat)
        report["players"].append({"player_id": seat, "name": name, "resources": resources(player_turns),
            "villagers": economy(player_turns, "VillagerStamina"), "village_centers": economy(player_turns, "VillageCenterStamina"),
            "military": military(samples), "fight_count": fight_count, "engagement_count": sum(labels.values()),
            "fight_labels": {label: labels[label] for label in LABELS},
            "outcome": next(p for p in game["Players"] if p["PlayerId"] == seat)})
    return report


def add_groups(groups: dict[tuple[str | None, str], dict[str, Any]], report: dict[str, Any],
               suite: dict[str, Any], args: argparse.Namespace) -> None:
    for player in report["players"]:
        seat = player["player_id"]
        turns = [{"Turn": turn["Turn"], **p} for turn in report["turns"]
                 for p in turn["Players"] if p["PlayerId"] == seat]
        samples = [p for fight in suite["Result"]["Game"]["Metrics"]["Fights"] or [] for frame in fight["Ticks"]
                   if in_window(frame["Turn"], args) for p in frame["Players"] if p["PlayerId"] == seat]
        for scope in (None, report["map"]):
            group = groups.setdefault((scope, player["name"]), {"map": scope, "name": player["name"],
                "games": 0, "fight_count": 0, "turns": [], "samples": [], "labels": Counter(), "unique": Counter()})
            group["games"] += 1
            group["fight_count"] += player["fight_count"]
            group["turns"].extend(turns)
            group["samples"].extend(samples)
            group["labels"].update(player["fight_labels"])
            group["unique"].update({field: player["military"][field]["unique"] for field in MEMBERSHIPS})


def group_report(group: dict[str, Any]) -> dict[str, Any]:
    army = military(group["samples"])
    for field in MEMBERSHIPS:
        army[field]["unique"] = group["unique"][field]
    bank = resources(group["turns"])
    for resource in bank["stockpile"].values():
        resource.pop("final")
    return {"map": group["map"], "name": group["name"], "games": group["games"],
            "resources": bank, "villagers": economy(group["turns"], "VillagerStamina"),
            "village_centers": economy(group["turns"], "VillageCenterStamina"),
            "military": army, "fight_count": group["fight_count"], "engagement_count": sum(group["labels"].values()),
            "fight_labels": {label: group["labels"][label] for label in LABELS}}


def display_percent(value: float | None) -> str:
    return f"{value:.2f}%" if value is not None else "n/a"


def print_summary(groups: list[dict[str, Any]]) -> None:
    print("map | AI | games | villager idle | VC idle | attack | overkill | move | idle | military unique/peak | adjacent peak | bad/neutral/good")
    for group in groups:
        army, labels = group["military"], group["fight_labels"]
        print(f"{group['map'] or 'overall'} | {group['name']} | {group['games']} | "
              f"{display_percent(group['villagers']['idle_pct'])} | {display_percent(group['village_centers']['idle_pct'])} | "
              f"{army['AttackStamina']:g} | {display_percent(army['overkill_pct'])} | "
              f"{army['MoveStamina']:g} | {army['IdleStamina']:g} | "
              f"{army['Units']['unique']}/{army['Units']['peak']} | {army['adjacent']['peak']} | "
              f"{labels.get('bad', 0)}/{labels.get('neutral', 0)}/{labels.get('good', 0)}")


def idle_pct(granted: int, wasted: int) -> str:
    pct = percentage(wasted, granted)
    return f"{pct:3.0f}%" if pct is not None else "  - "


def completion_turns(records: list[tuple[int, dict[str, Any]]]) -> dict[int, list[int]]:
    built: dict[int, list[int]] = {}
    seen: dict[int, int] = {}
    for turn, player in records:
        for entry in player.get("Production") or []:
            building = entry["BuildingId"]
            built.setdefault(building, []).extend([turn] * (entry["Count"] - seen.get(building, 0)))
            seen[building] = entry["Count"]
    return built


def print_by_turn(sim: dict[str, Any], suite: dict[str, Any], step: int, last: int) -> None:
    result = suite["Result"]
    turns = result["Game"]["Metrics"]["Turns"]
    snapshots = {snapshot[0]["Turn"]: snapshot for snapshot in result["Timeline"]}
    for seat, index in enumerate(suite["Seats"]):
        name = sim["Stats"]["Players"][index]["Name"]
        records = [(t["Turn"], t["Players"][seat]) for t in turns if t["Turn"] <= last]
        print(f"\n{name} seat {seat} seed {result['Seed']}")
        for building, built_turns in sorted(completion_turns(records).items()):
            print(f"  building {building} completed at turns {built_turns}")
        print("  turn | vil idle | vc idle | production idle (id:count idle) | food wood stone gold | income f/w/s/g | spent f/w/s/g | units mil")
        for end in range(step, last + 1, step):
            block = [p for t, p in records if end - step < t <= end]
            if not block:
                continue
            vil = [sum(p[k][f] for p in block) for k in ("VillagerStamina", "VillageCenterStamina") for f in ("Granted", "Wasted")]
            production = {}
            for p in block:
                for e in p.get("Production") or []:
                    total = production.setdefault(e["BuildingId"], [0, 0, 0])
                    total[0], total[1], total[2] = e["Count"], total[1] + e["Stamina"]["Granted"], total[2] + e["Stamina"]["Wasted"]
            prod_text = " ".join(f"{b}:{c} {idle_pct(g, w).strip()}" for b, (c, g, w) in sorted(production.items()))
            stock = " ".join(f"{v:5.0f}" for v in block[-1]["Stockpile"])
            income = "/".join(f"{sum(p['Income'][i] for p in block):.0f}" for i in range(4))
            spent = "/".join(f"{sum(p['Spent'][i] for p in block):.0f}" for i in range(4))
            units = snapshots.get(end, [{}] * (seat + 1))[seat].get("Units", {})
            total_units, military = sum(units.values()), sum(n for u, n in units.items() if u != "1")
            print(f"  {end:4d} | {idle_pct(vil[0], vil[1])}    | {idle_pct(vil[2], vil[3])}   | {prod_text:32s} | {stock} | {income} | {spent} | {total_units} {military}")


def print_at_turns(sim: dict[str, Any], suite: dict[str, Any], checkpoints: list[int]) -> None:
    result = suite["Result"]
    turns = result["Game"]["Metrics"]["Turns"]
    for seat, index in enumerate(suite["Seats"]):
        print(f"\n{sim['Stats']['Players'][index]['Name']} seat {seat} seed {result['Seed']}")
        print("  turn | vils | vil idle | vc idle | gathered f/w/s/g (cumulative) | spent (cumulative)")
        for checkpoint in checkpoints:
            records = [t["Players"][seat] for t in turns if 0 < t["Turn"] <= checkpoint]
            if not records:
                continue
            idle = lambda key, field: sum(p[key][field] for p in records)
            gathered = [sum(p["Income"][i] for p in records) for i in range(4)]
            spent = sum(sum(p["Spent"]) for p in records)
            print(f"  {checkpoint:4d} | {records[-1]['Villagers']:4d} | {idle_pct(idle('VillagerStamina', 'Granted'), idle('VillagerStamina', 'Wasted'))}    | "
                  f"{idle_pct(idle('VillageCenterStamina', 'Granted'), idle('VillageCenterStamina', 'Wasted'))}   | "
                  f"{'/'.join(f'{c:.0f}' for c in gathered):30s} | {spent:.0f}")


def main() -> None:
    args = arguments()
    source = args.results / "results.json" if args.results.is_dir() else args.results
    destination = args.output or source.with_name("metrics_analysis.json")
    if source.resolve() == destination.resolve():
        raise ValueError("Analysis output must differ from input results")
    run = json.loads(source.read_text())
    if args.at_turn:
        for sim in run["Sims"]:
            for suite in sim["Games"]:
                if not args.seeds or suite["Result"]["Seed"] in args.seeds:
                    print_at_turns(sim, suite, args.at_turn)
        return
    if args.by_turn:
        for sim in run["Sims"]:
            for suite in sim["Games"]:
                if args.seeds and suite["Result"]["Seed"] not in args.seeds:
                    continue
                print_by_turn(sim, suite, args.by_turn, min(args.to_turn, suite["Result"]["Game"]["TurnNumber"]))
        return
    definitions = json.loads((Path(__file__).resolve().parents[2] / "config/units.json").read_text())
    units = {str(unit["unit_id"]): unit for unit in definitions}
    reports: list[dict[str, Any]] = []
    groups: dict[tuple[str | None, str], dict[str, Any]] = {}
    for sim in run["Sims"]:
        if args.maps and sim["Map"] not in args.maps:
            continue
        for suite in sim["Games"]:
            report = analyze_game(sim, suite, units, args)
            if report.get("error"):
                reports.append(report)
            elif report["players"] and report["turns"]:
                reports.append(report)
                add_groups(groups, report, suite, args)
    if not reports:
        raise ValueError("No games match the requested AI, map and turn filters")
    summaries = [group_report(group) for group in groups.values()]
    rules = {"turn_window": [args.from_turn, args.to_turn], "strength_ratio": args.strength_ratio,
        "strength": "sum(count * sqrt(max_health * turn_stamina * damage_per_stamina_vs_enemy_mix))",
        "fight_label": "First local military attack-order commitment per player; frozen pre-combat snapshot; no engagement means no label",
        "bad": "Enemy strength meets threshold and fight is away from base",
        "good": "Own strength meets threshold; all other engagements are neutral",
        "base": "Connected owned building spaces; keep VC-containing components when any VC exists, otherwise keep all; defense requires fight-space overlap",
        "limitations": "Composition estimate ignores injuries, techs, civilian and building combat power",
        "stockpiles": "End-of-turn samples; turn zero retained but excluded from mean and peak",
        "income": "Recorded Income includes refunds", "idle": "Actual refresh loss credited to preceding turn; final turn has no refresh",
        "memberships": "Observations repeat units across ticks; opening strength includes fight spaces and immediate adjacent military once",
        "label_window": "Count engagements committed in the selected window; retain original labels on continuing fights"}
    output = {"source": str(source.resolve()), "rules": rules, "unit_definitions": definitions,
              "summary": summaries, "games": reports}
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_text(json.dumps(output, indent=2, allow_nan=False) + "\n")
    print_summary(summaries)
    errors = sum(bool(report.get("error")) for report in reports)
    print(f"{len(reports)} games, {errors} errors; full analysis: {destination}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError) as error:
        raise SystemExit(f"metrics analysis failed: {error}") from error
