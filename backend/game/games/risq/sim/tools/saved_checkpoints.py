from __future__ import annotations

import argparse
import json
import statistics
from pathlib import Path
from typing import Any

from metrics import economy, percentage, resources
from run_games import load_run


def checkpoint_rows(run: dict[str, Any], checkpoints: list[int]) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for sim in run["Sims"]:
        for game in sim["Games"]:
            result = game["Result"]
            for seat, index in enumerate(game["Seats"]):
                name = sim["Stats"]["Players"][index]["Name"]
                player = result["Game"]["Players"][seat]
                enemy = result["Game"]["Players"][1 - seat] if len(game["Seats"]) == 2 else None
                for end in checkpoints:
                    records = [{"Turn": t["Turn"], **t["Players"][seat]} for t in result["Game"]["Metrics"]["Turns"] if 0 < t["Turn"] <= end]
                    standing = next((t for t in reversed(result["Trajectory"]) if t["Turn"] <= end), None)
                    total = result["Game"]["Players"][seat]["Units"] if result["Game"]["TurnNumber"] <= end else standing["Players"][seat]["Units"]
                    production = [p for record in records for p in record.get("Production") or [] if p["BuildingId"] == 22]
                    row = dict(bucket=sim["Name"], name=name, seed=result["Seed"], turn=end,
                               villagers=records[-1]["Villagers"], military_alive=total - records[-1]["Villagers"],
                               resources=resources(records), villager_idle=economy(records, "VillagerStamina"),
                               vc_idle=economy(records, "VillageCenterStamina"), rax_idle=economy(production, "Stamina"))
                    row["final_military_lost"] = player["UnitsLost"] - player["Economy"]["VillagersLost"]
                    row["final_military_kills"] = enemy["UnitsLost"] - enemy["Economy"]["VillagersLost"] if enemy else None
                    for building, label in ((22, "first_rax"), (11, "first_smith")):
                        turns = [t["Turn"] for t in result["Game"]["Metrics"]["Turns"] if any(p["BuildingId"] == building and p["Count"] > 0
                                 for p in t["Players"][seat].get("Production") or [])]
                        row[label] = min(turns) if turns else None
                        row[f"{label}_ready"] = bool(turns and min(turns) <= end)
                    rows.append(row)
    return rows


def summarize(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    keys = {(bucket, r["name"], r["turn"]) for r in rows for bucket in (None, r["bucket"])}
    results: list[dict[str, Any]] = []
    for bucket, name, end in sorted(keys, key=lambda k: (k[0] or "", k[1], k[2])):
        group = [r for r in rows if r["name"] == name and r["turn"] == end and (bucket is None or r["bucket"] == bucket)]
        row: dict[str, Any] = dict(bucket=bucket, name=name, turn=end, games=len(group))
        for field in ("villagers", "military_alive", "final_military_lost", "final_military_kills"):
            values = [r[field] for r in group if r[field] is not None]
            row[field] = statistics.mean(values) if values else None
        for field in ("villager_idle", "vc_idle", "rax_idle"):
            granted, wasted = (sum(r[field][key] for r in group) for key in ("granted", "wasted"))
            row[field] = percentage(wasted, granted)
        for field in ("income", "spent"):
            row[field] = {resource: statistics.mean(r["resources"][field][resource] for r in group)
                          for resource in ("food", "wood", "stone", "gold")}
        row["bank"] = {resource: statistics.mean(r["resources"]["stockpile"][resource]["final"] for r in group)
                       for resource in ("food", "wood", "stone", "gold")}
        for field in ("first_rax", "first_smith"):
            values = [r[field] for r in group if r[field] is not None]
            row[field] = statistics.mean(values) if values else None
            row[f"{field}_completed"] = len(values)
            row[f"{field}_ready"] = sum(r[f"{field}_ready"] for r in group)
        results.append(row)
    return results


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("run", type=Path)
    parser.add_argument("--buckets", nargs="+", required=True)
    parser.add_argument("--turns", default="10,20,30,40")
    args = parser.parse_args()
    run = load_run(args.run, set(args.buckets), include_fights=False)
    rows = checkpoint_rows(run, [int(t) for t in args.turns.split(",")])
    report = dict(source=str(args.run.resolve()), summary=summarize(rows), games=rows,
                  notes="Idle is cumulative wasted/granted stamina; army counts use completed-turn standings.")
    (args.run / "early_economy.json").write_text(json.dumps(report, indent=2) + "\n")
    print(f"Saved {len(rows)} player checkpoints across {len(run['Sims'])} buckets")


if __name__ == "__main__":
    main()
