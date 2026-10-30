from __future__ import annotations

import json
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

from fight_opportunities import audit_opportunities
from tick_trace import HEALTH, HIT
from tool_search import read_ai_records, read_order_batches


def combat_events(path: Path, frames: dict[tuple[str, int], dict]) -> tuple[list[dict], list[dict]]:
    owners = {unit["InternalID"]: unit["PlayerID"] for data in frames.values() for unit in data["units"]}
    deaths: list[dict] = []
    hits: list[dict] = []
    seen: set[int] = set()
    with path.open() as source:
        for line in source:
            if hit := HIT.search(line):
                attacker_unit = f": {hit[3]} u{hit[4]} (player " in line
                target_unit = f"hits {hit[6]} u{hit[7]} (player " in line
                if attacker_unit:
                    owners[int(hit[3])] = int(hit[5])
                if target_unit:
                    owners[int(hit[6])] = int(hit[8])
                hits.append(dict(turn=int(hit[1]), tick=int(hit[2]), attacker=int(hit[3]),
                                 attacker_kind=int(hit[4]), owner=int(hit[5]), target=int(hit[6]),
                                 target_kind=int(hit[7]), damage=float(hit[9]),
                                 attacker_unit=attacker_unit, target_unit=target_unit))
            elif health := HEALTH.search(line):
                uid, before, delta = int(health[3]), float(health[5]), float(health[6])
                if before > 0 and before + delta <= 0 and uid not in seen:
                    seen.add(uid)
                    deaths.append(dict(turn=int(health[1]), tick=int(health[2]), uid=uid,
                                       unit_id=int(health[4]), owner=owners[uid]))
    return deaths, hits


def fight_indexes(game: dict[str, Any]) -> tuple[dict, dict]:
    inside: dict[tuple[int, int, int], int] = {}
    engaging: dict[tuple[int, int, int], set[int]] = defaultdict(set)
    for fight in game["Result"]["Game"]["Metrics"]["Fights"] or []:
        for frame in fight["Ticks"]:
            for player in frame["Players"]:
                for uid in player["Units"] or []:
                    key = frame["Turn"], frame["Tick"], uid
                    if key in inside and inside[key] != fight["FightId"]:
                        raise ValueError(f"Unit belongs to two fights: {key}")
                    inside[key] = fight["FightId"]
                for uid in player["Engaging"] or []:
                    engaging[(frame["Turn"], frame["Tick"], uid)].add(fight["FightId"])
    return inside, engaging


def audit_fights(game: dict[str, Any], report: dict[str, Any], deaths: list[dict],
                 hits: list[dict], units: dict[str, Any]) -> tuple[list[dict], dict]:
    inside, engaging = fight_indexes(game)
    rows = {fight["fight_id"]: fight for fight in report["fights"]}
    unmatched: Counter[str] = Counter()
    military = {int(uid) for uid, unit in units.items() if unit["unit_type"] != "economic"}
    for row in rows.values():
        row["army_vs_army"] = False
        for player in row["players"]:
            player.update(military_losses=0, loss_cost=0.0, loss_types={}, military_damage=0.0,
                          damage_to_military=0.0, support_damage=0.0)
    for death in deaths:
        if death["unit_id"] not in military:
            continue
        fid = inside.get((death["turn"], death["tick"], death["uid"]))
        if fid not in rows:
            unmatched[f"military_losses_player_{death['owner']}"] += 1
            continue
        player = next(p for p in rows[fid]["players"] if p["player_id"] == death["owner"])
        player["military_losses"] += 1
        kind = str(death["unit_id"])
        player["loss_types"][kind] = player["loss_types"].get(kind, 0) + 1
        player["loss_cost"] += sum(units[kind]["cost"].values())
    for hit in hits:
        key = hit["turn"], hit["tick"], hit["target"]
        fid = inside.get(key) if hit["target_unit"] else None
        if fid is None and hit["attacker_unit"]:
            key = hit["turn"], hit["tick"], hit["attacker"]
            fid = inside.get(key)
            options = engaging.get(key, set())
            if fid is None and len(options) == 1:
                fid = next(iter(options))
        if fid not in rows:
            unmatched["unassigned_hit_damage"] += hit["damage"]
            continue
        row = rows[fid]
        player = next(p for p in row["players"] if p["player_id"] == hit["owner"])
        army_hit = hit["attacker_unit"] and hit["attacker_kind"] in military
        target_military = hit["target_unit"] and hit["target_kind"] in military
        player["military_damage" if army_hit else "support_damage"] += hit["damage"]
        if army_hit and target_military:
            player["damage_to_military"] += hit["damage"]
            row["army_vs_army"] = True
    return list(rows.values()), dict(unmatched)


def add_turn_labels(game: dict[str, Any], fights: list[dict], units: dict[str, Any], ratio: float) -> None:
    from metrics import assess
    rows = {fight["fight_id"]: fight for fight in fights}
    for fight in game["Result"]["Game"]["Metrics"]["Fights"] or []:
        seen: set[tuple[int, int]] = set()
        players = {p["player_id"]: p for p in rows[fight["FightId"]]["players"]}
        for frame in fight["Ticks"]:
            for player in frame["Players"]:
                key = frame["Turn"], player["PlayerId"]
                if player["Engaging"] and key not in seen:
                    seen.add(key)
                    assessment = assess(frame, player, units, ratio)
                    players[player["PlayerId"]].setdefault("turn_labels", []).append(
                        {"turn": frame["Turn"], "tick": frame["Tick"], **assessment})


def format_number(value: float | None) -> str:
    return "-" if value is None else f"{value:.1f}"


def write_ledger(destination: Path, games: list[dict]) -> None:
    lines = ["Fight ledger", "",
             "Labels use the existing first-commitment composition model and a 1.5 strength ratio.",
             "Bad means outgunned away from base; outgunned base defense remains neutral with an explicit reason.",
             "Labels ignore injuries and researched techs. Casualties and overkill come from actual resolution.",
             "Overkill is wasted attack stamina / total attack stamina. Damage includes raw damage before overkill.", ""]
    for game in games:
        lines.extend([f"Seed {game['seed']}", "",
                      "| Fight | Turns | Army combat | V7 label / ratio | V6 label / ratio | Military lost V7/V6 | Overkill % V7/V6 | Military damage V7/V6 |",
                      "|---|---|---|---|---|---|---|---|"])
        for fight in game["fights"]:
            players = {p["player_id"]: p for p in fight["players"]}
            a, b = players[game["v7_seat"]], players[1 - game["v7_seat"]]
            ca, cb = a["classification"], b["classification"]
            labels = [(c["label"] + " / " + format_number(c["strength_ratio"])) if c else "no commitment" for c in (ca, cb)]
            for index, player in enumerate((a, b)):
                bad_turns = [entry["turn"] for entry in player.get("turn_labels", []) if entry["label"] == "bad"]
                if bad_turns:
                    labels[index] += f"; bad at t{bad_turns[0]}"
            lines.append(f"| {fight['fight_id']} | {fight['first_tick'][0]}-{fight['last_tick'][0]} | {fight['army_vs_army']} | "
                         f"{labels[0]} | {labels[1]} | {a['military_losses']}/{b['military_losses']} | "
                         f"{format_number(a['military']['overkill_pct'])}/{format_number(b['military']['overkill_pct'])} | "
                         f"{a['military_damage']:.0f}/{b['military_damage']:.0f} |")
        lines.append("")
    destination.write_text("\n".join(lines) + "\n")


def audit_run(run: dict[str, Any], analysis: dict[str, Any], folder: Path,
              units: dict[str, Any], ratio: float) -> dict[str, Any]:
    reports = {report["seed"]: report for report in analysis["games"]}
    games: list[dict] = []
    for sim in run["Sims"]:
        for game in sim["Games"]:
            seed = game["Result"]["Seed"]
            if seed not in reports:
                continue
            path = folder / f"debug_{seed}.log"
            if not path.exists():
                path = folder / "debug.log"
            frames = read_ai_records(path, "new turn data")
            variables = read_ai_records(path, "vars")
            batches = read_order_batches(path)
            deaths, hits = combat_events(path, frames)
            fights, unmatched = audit_fights(game, reports[seed], deaths, hits, units)
            add_turn_labels(game, fights, units, ratio)
            seat = game["Seats"].index(0)
            for player in game["Result"]["Game"]["Players"]:
                military_deaths = sum(d["owner"] == player["PlayerId"] and units[str(d["unit_id"])]["unit_type"] != "economic" for d in deaths)
                expected = player["UnitsLost"] - player["Economy"]["VillagersLost"]
                if military_deaths != expected:
                    raise ValueError(f"Seed {seed} player {player['PlayerId']}: deaths {military_deaths} != result {expected}")
            opportunities = audit_opportunities(frames, variables, batches, seat, ratio)
            games.append(dict(seed=seed, v7_seat=seat, outcomes=game["Result"]["Outcomes"],
                              final_players=game["Result"]["Game"]["Players"],
                              fights=fights, opportunities=opportunities, unmatched=unmatched))
    output = {"strength_ratio": ratio, "opportunity_rule": "Adjacent visible military target, AI-model power advantage >= ratio, no current army engagement, no coordinated commitment, at least two consecutive turns.",
              "limitations": "Fight labels ignore injuries and techs; opportunity windows are candidates under the AI model, not guaranteed wins.",
              "games": games}
    (folder / "fight_audit.json").write_text(json.dumps(output, indent=2) + "\n")
    write_ledger(folder / "fight_ledger.md", games)
    opportunities = ["Candidate favorable opportunities", "",
                     "Each window is locally favorable under V7's own model, not a guaranteed or exhaustive missed win.", ""]
    for game in games:
        for window in game["opportunities"]:
            opportunities.append(f"- Seed {game['seed']}, turns {window['first_turn']}-{window['last_turn']}, target {window['target']}: "
                                 f"{window['reason']}; local model power {window['own_power']:.1f}/{window['enemy_power']:.1f}, "
                                 f"army gathered {window['army_in_center']:.0f}/{window['army_size']:.0f}.")
    (folder / "opportunities.md").write_text("\n".join(opportunities) + "\n")
    return output
