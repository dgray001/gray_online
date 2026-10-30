"""Loads a sim run: results.json, with each sim's games read from the per-game files the sim writes to games/ unless -quiet."""
from __future__ import annotations

import json
from pathlib import Path
from typing import Any


def load_run(path: Path, bucket_names: set[str] | None = None, include_fights: bool = True) -> dict[str, Any]:
    folder = path if path.is_dir() else path.parent
    if (folder / "results.json").exists():
        run = json.loads((folder / "results.json").read_text())
    else:
        scenario = json.loads((folder.parent.parent / "inputs" / f"{folder.name}.json").read_text())
        players = [{"Name": player["ai_config"]} for player in scenario["players"]]
        buckets = scenario.get("buckets") or [{"name": folder.name, "map": scenario["map"]}]
        run = {"Sims": [{"Name": bucket["name"], "Map": bucket["map"],
                         "Stats": {"Players": players}} for bucket in buckets]}
    if bucket_names is not None:
        run["Sims"] = [sim for sim in run["Sims"] if sim["Name"] in bucket_names]
    wanted = {sim["Name"] for sim in run["Sims"]}
    games_dir = folder / "games"
    if not games_dir.is_dir():
        raise SystemExit(f"{folder} has no games/ folder; per-game results are only written by a sim run without -quiet")
    by_bucket: dict[str, list[tuple[int, dict[str, Any]]]] = {}
    for file in games_dir.glob("*.json"):
        if bucket_names is not None and file.stem.rsplit("_", 2)[0] not in wanted:
            continue
        game = json.loads(file.read_text())
        if not include_fights:
            metrics = (game["Result"].get("Game") or {}).get("Metrics") or {}
            metrics.pop("Fights", None)
        index = int(file.stem.rsplit("_", 2)[-2])
        by_bucket.setdefault(game["Bucket"], []).append((index, game))
    for sim in run["Sims"]:
        games = sorted(by_bucket.get(sim["Name"], []), key=lambda entry: entry[0])
        sim["Games"] = [game for _, game in games]
    return run
