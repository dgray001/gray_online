"""Runs AI configs on the five sim_army seeds where tmp_cC (v0.6 + spill + adjacent attacks) was crushed, against v0.5.
usage: python3 tools/army5.py tmp_a tmp_b ...   (prints per config: the AI's outcome per seed, plus kills and losses)"""
import json
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

SIM_DIR = Path(__file__).resolve().parents[1]
SEEDS = {106014: 0, 106016: 0, 106018: 0, 106021: 1, 106029: 1}  # seed -> seat the AI under test takes
OUTCOME_ORDER = {name: rank for rank, name in enumerate(["defeated", "crushed", "losing", "behind", "ahead", "winning", "crush", "defeat"])}


def run_game(ai_config: str, seed: int, seat: int) -> tuple[str, int, int]:
    name = f"a5_{ai_config}_{seed}"
    players = ["v0.5", "v0.5"]
    players[seat] = ai_config
    scenario = {"seed": seed, "iterations": 1, "map": "script:sim_army", "max_turns": 100, "players": [{"ai_config": p} for p in players]}
    (SIM_DIR / "inputs" / f"{name}.json").write_text(json.dumps(scenario))
    subprocess.run(["go", "run", ".", "-quiet", name], cwd=SIM_DIR, capture_output=True, check=True)
    result = json.loads((SIM_DIR / "outputs" / name / "results.json").read_text())["Sims"][0]["Games"][0]["Result"]
    game = result["Game"]["Players"][seat]
    return result["Outcomes"][seat], game["Kills"], game["UnitsLost"]


def main() -> None:
    subprocess.run(["go", "build", "./..."], cwd=SIM_DIR, check=True)
    for ai_config in sys.argv[1:]:
        with ThreadPoolExecutor(max_workers=5) as pool:
            games = list(pool.map(lambda item: run_game(ai_config, *item), SEEDS.items()))
        outcomes = " ".join(g[0] for g in games)
        score = sum(OUTCOME_ORDER[g[0]] for g in games)
        print(f"{ai_config:28s} {outcomes}  score {score:2d}  kills {sum(g[1] for g in games):3d}  lost {sum(g[2] for g in games):3d}   per game kills/lost: " + " ".join(f"{g[1]}/{g[2]}" for g in games))


if __name__ == "__main__":
    main()
