from __future__ import annotations

import argparse
import importlib.util
import json
import statistics
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
from typing import Any

from tool_search import read_ai_records
from trajectory import read_reports


def game_details(game: dict[str, Any], folder: Path) -> dict[str, Any]:
    result = game["Result"]
    seat = game["Seats"].index(0)
    path = folder / f"debug_{result['Seed']}.log"
    reports = read_reports(path) if path.exists() else {}
    variables = read_ai_records(path, "vars", {30, 40, 50, 60}) if path.exists() else {}
    players = result["Game"]["Players"]
    own, enemy = players[seat], players[1 - seat]
    turns = result["Game"]["Metrics"]["Turns"]
    row = dict(seed=result["Seed"], map=game["Map"], outcome=result["Outcomes"][seat],
               kills=enemy["UnitsLost"] - enemy["Economy"]["VillagersLost"],
               lost=own["UnitsLost"] - own["Economy"]["VillagersLost"],
               score_margin=own["Score"] - enemy["Score"], land=own["Land"], techs=own["TechsResearched"])
    for index, label in ((seat, "own"), (1 - seat, "enemy")):
        for tech, name in ((61, "attack"), (62, "armor")):
            completed = [turn for turn, ps in reports.items() if tech in ps[index]["production"]["techs_researched"]]
            row[f"{label}_{name}"] = min(completed) if completed else None
    for bid, name in ((11, "smith"), (22, "rax")):
        completed = [t["Turn"] for t in turns if any(e["BuildingId"] == bid and e["Count"] > 0
                     for e in t["Players"][seat].get("Production") or [])]
        row[name] = min(completed) if completed else None
    for end in (20, 30, 40):
        records = [t["Players"][seat] for t in turns if 0 < t["Turn"] <= end]
        row[f"vils{end}"] = records[-1]["Villagers"]
        granted = sum(p["VillageCenterStamina"]["Granted"] for p in records)
        wasted = sum(p["VillageCenterStamina"]["Wasted"] for p in records)
        row[f"vc_idle{end}"] = 100 * wasted / granted if granted else 0
        granted = sum(p["VillagerStamina"]["Granted"] for p in records)
        wasted = sum(p["VillagerStamina"]["Wasted"] for p in records)
        row[f"vil_idle{end}"] = 100 * wasted / granted if granted else 0
    for end in (20, 30, 40):
        events = [e for t, ps in reports.items() if 0 < t <= end for e in ps[seat]["production"]["units_created"]]
        row[f"military_produced{end}"] = sum(e["count"] for e in events if e["unit_id"] != 1)
        row[f"blunts_produced{end}"] = sum(e["count"] for e in events if e["unit_id"] == 101)
    for index, resource in enumerate(("food", "wood", "stone", "gold")):
        row[f"income_{resource}40"] = sum(t["Players"][seat]["Income"][index] for t in turns if 0 < t["Turn"] <= 40)
        row[f"spent_{resource}40"] = sum(t["Players"][seat]["Spent"][index] for t in turns if 0 < t["Turn"] <= 40)
    for turn in (30, 40):
        values = variables.get((f"ai-{seat}", turn), {})
        row[f"protected{turn}"] = sum(values.get(f"protected_{r}", 0) for r in ("food", "wood", "stone", "gold")) * values.get("protected_scale", 1)
        producers = sum(values.get(f"protected_producers_{bid}", values.get(f"complete_{bid}", 0)) for bid in (22, 23))
        row[f"protected_producers{turn}"] = producers * (values.get("n_military", 0) < values.get("tune_protected_military_count", 1000000))
        row[f"protected_vil_flow{turn}"] = values.get("flow_1", 0) * (values.get("own_1", 0) < values.get("tune_protected_villager_share", 1) * values.get("goal_1", 0))
        for key in ("investment_workers", "workers_stone", "n_available_builders", "complete_22", "n_raiding_party", "n_scout_units", "army_size", "n_military"):
            row[f"{key}{turn}"] = values.get(key)
    row["both40"] = all(row[f"own_{name}"] is not None and row[f"own_{name}"] <= 40 for name in ("attack", "armor"))
    for turn in (30, 40, 50, 60):
        own_vars = variables.get((f"ai-{seat}", turn), {})
        enemy_vars = variables.get((f"ai-{1 - seat}", turn), {})
        row[f"worker_goal{turn}"] = own_vars.get("worker_goal")
        row[f"workers{turn}"] = own_vars.get("own_1")
        row[f"enemy_worker_goal{turn}"] = enemy_vars.get("vil_goal")
        row[f"enemy_workers{turn}"] = enemy_vars.get("vils")
    return row


def summarize_rows(rows: list[dict[str, Any]], limit: int) -> dict[str, Any]:
    summary: dict[str, Any] = dict(games=len(rows), favorable=sum(r["outcome"] in ("ahead", "winning", "crush", "defeat") for r in rows),
                                   eliminations=sum(r["outcome"] == "defeat" for r in rows), both40=sum(r["both40"] for r in rows))
    for field in ("smith", "rax", "own_attack", "own_armor", "enemy_attack", "enemy_armor"):
        summary[field] = statistics.mean(r[field] if r[field] is not None else limit + 1 for r in rows)
        summary[f"{field}_completed"] = sum(r[field] is not None for r in rows)
    for field in ("kills", "lost", "score_margin", "land", "techs", "vils20", "vils30", "vils40", "vc_idle20", "vc_idle40", "protected30", "protected40", "investment_workers30", "investment_workers40", "workers_stone30", "workers_stone40"):
        values = [r[field] for r in rows if r[field] is not None]
        summary[field] = statistics.mean(values) if values else None
    for field in ("vil_idle20", "vil_idle40", "military_produced40", "income_food40", "income_wood40", "income_stone40", "spent_food40", "spent_wood40", "spent_stone40", "spent_gold40"):
        summary[field] = statistics.mean(r[field] for r in rows)
    for field in ("military_produced20", "military_produced30", "blunts_produced20", "blunts_produced30", "blunts_produced40"):
        summary[field] = statistics.mean(r[field] for r in rows)
    for field in ("protected_producers30", "protected_producers40", "protected_vil_flow30", "protected_vil_flow40"):
        summary[field] = statistics.mean(r[field] for r in rows)
    for turn in (30, 40):
        for prefix in ("n_raiding_party", "n_scout_units", "army_size", "n_military"):
            values = [r[f"{prefix}{turn}"] for r in rows if r[f"{prefix}{turn}"] is not None]
            summary[f"{prefix}{turn}"] = statistics.mean(values) if values else None
    for turn in (30, 40, 50, 60):
        for prefix in ("worker_goal", "workers", "enemy_worker_goal", "enemy_workers"):
            values = [r[f"{prefix}{turn}"] for r in rows if r[f"{prefix}{turn}"] is not None]
            summary[f"{prefix}{turn}"] = statistics.mean(values) if values else None
    return summary


def run_trial(trial: dict[str, Any], manifest: dict[str, Any], sim: Path) -> dict[str, Any]:
    config = Path(manifest["config"])
    if trial.get("generate", True):
        generator = Path(manifest["generator"])
        command = ["python3", str(generator), "--config", str(config), "--out", str(config / "ai" / f"{trial['ai']}.json")]
        command.extend(f"{key}={value}" for key, value in trial["parameters"].items())
        subprocess.run(command, check=True, capture_output=True, text=True)
    folder = sim / "outputs" / trial["name"]
    if not (folder / "results.json").exists():
        command = [manifest["binary"], "-config", str(config), "-parallel", str(trial.get("parallel", manifest.get("games_parallel", 1)))]
        if manifest.get("debug", False):
            command.append("-debug")
        subprocess.run(command + [trial["name"]], cwd=sim, check=True, capture_output=True, text=True)
    result = json.loads((folder / "results.json").read_text())
    errors = result["Overall"]["Players"][0]["Errors"]
    if errors:
        raise ValueError(f"{trial['name']}: {errors} game errors")
    rows = [game_details(json.loads(path.read_text()), folder) for path in sorted((folder / "games").glob("*.json"))]
    summary = summarize_rows(rows, result["MaxTurns"])
    maps = {name: summarize_rows([r for r in rows if r["map"] == name], result["MaxTurns"])
            for name in sorted({r["map"] for r in rows})}
    return dict(trial=trial, summary=summary, maps=maps, games=rows)


def write_catalog(manifest: dict[str, Any], sim: Path) -> None:
    spec = importlib.util.spec_from_file_location("sweep_parameters", Path(manifest["generator"]).with_name("parameters.py"))
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    baseline = module.parameters({})
    baseline_key = json.dumps(baseline, sort_keys=True)
    prefix = "_".join(manifest["name"].split("_")[:3])
    runs: dict[str, dict[str, Any]] = {}
    groups: dict[str, dict[str, Any]] = {}
    for path in sorted((sim / "outputs").glob(f"{prefix}_*/summary.json")):
        phase = json.loads(path.read_text())
        limit = phase["manifest"]["design"]["max_turns"]
        for entry in phase["trials"]:
            trial = entry["trial"]
            values = module.parameters(trial["parameters"])
            key = json.dumps(values, sort_keys=True)
            group = groups.setdefault(key, dict(parameters=values, runs=[], full_game=False, baseline=key == baseline_key))
            name = trial["name"]
            if name not in group["runs"]:
                group["runs"].append(name)
            group["full_game"] = group["full_game"] or (limit >= 100 and "summary" in entry)
            runs[name] = dict(phase=path.parent.name, turn_limit=limit, **entry)
    configurations = [groups[key] for key in sorted(groups)]
    report = dict(experimental_configurations=sum(not g["baseline"] for g in configurations),
                  full_game_experimental_configurations=sum(not g["baseline"] and g["full_game"] for g in configurations),
                  recorded_game_runs=sum(r.get("summary", {}).get("games", 0) for r in runs.values()),
                  failed_trials=sum("error" in r for r in runs.values()), configurations=configurations,
                  trials=[runs[name] for name in sorted(runs)])
    folder = sim / "outputs" / prefix
    folder.mkdir(parents=True, exist_ok=True)
    (folder / "catalog.json").write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({key: value for key, value in report.items() if key not in ("configurations", "trials")}))


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--parallel", type=int, default=28)
    parser.add_argument("--catalog", action="store_true")
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text())
    sim = Path(__file__).resolve().parents[1]
    if args.catalog:
        write_catalog(manifest, sim)
        return
    output = sim / "outputs" / manifest["name"]
    output.mkdir(parents=True, exist_ok=True)
    destination = output / "summary.json"
    completed: list[dict[str, Any]] = []
    with ThreadPoolExecutor(max_workers=args.parallel) as executor:
        futures = {executor.submit(run_trial, trial, manifest, sim): trial for trial in manifest["trials"]}
        for future in as_completed(futures):
            trial = futures[future]
            try:
                entry = future.result()
            except Exception as error:
                entry = dict(trial=trial, error=str(error))
            completed.append(entry)
            entries = sorted(completed, key=lambda row: row["trial"]["name"])
            destination.write_text(json.dumps(dict(manifest=manifest, trials=entries), indent=2) + "\n")
            print(json.dumps(dict(name=trial["name"], completed=len(completed), total=len(futures),
                                  summary=entry.get("summary"), error=entry.get("error"))), flush=True)
    if any("error" in entry for entry in completed):
        raise SystemExit("Sweep contains failed trials; inspect summary.json")


if __name__ == "__main__":
    main()
