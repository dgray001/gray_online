from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CONFIG = Path('/tmp/v07_ablation_config_20261017')
WORKERS = 'v0.7_ablation_workers'
RETREAT = 'v0.7_ablation_retreat'
MATCHUPS = [(WORKERS, 'v0.6'), (RETREAT, 'v0.6'),
            (WORKERS, 'v0.7'), (RETREAT, 'v0.7'),
            (WORKERS, RETREAT), (RETREAT, WORKERS), ('v0.7', 'v0.7')]


def prepare() -> list[str]:
    baseline = json.loads((ROOT / 'inputs/eval.json').read_text())
    cases: list[dict[str, object]] = []
    names: list[str] = []
    for bucket_index, bucket in enumerate(baseline['buckets']):
        for sample_index in range(3):
            game_index = sample_index + bucket_index % 2
            cases.append({**bucket, 'name': f"{bucket['name']}__{game_index:02d}",
                          'seed': baseline['seed'] + bucket_index * 1000 + game_index,
                          'swap': game_index % 2 == 1})
    assert sum(bool(case['swap']) for case in cases) == 15
    for index, (first, second) in enumerate(MATCHUPS):
        name = f'v07_ablation_{index + 1}'
        scenario = {**baseline, 'games_per_bucket': 1, 'buckets': cases,
                    'players': [{'ai_config': first}, {'ai_config': second}]}
        (ROOT / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
        names.append(name)
    return names


def prepare_replays() -> list[str]:
    names: list[str] = []
    for label, model, opponent, seed, map_name, swapped in [
            ('workers', WORKERS, 'v0.6', 101002, 'hexagon', False),
            ('retreat', RETREAT, 'v0.6', 100001, 'ring', True),
            ('retreat_duel', RETREAT, 'v0.7', 101003, 'hexagon', True)]:
        for control in (False, True):
            name = f"v07_ablation_{label}_{'control' if control else 'replay'}"
            players = [{'ai_config': 'v0.7' if control else model}, {'ai_config': opponent}]
            scenario = {'seed': seed, 'fixed_seed': True, 'iterations': 1,
                        'map': f'script:{map_name}', 'max_turns': 100,
                        'players': players[::-1] if swapped else players,
                        'metrics': ['production', 'fights']}
            (ROOT / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
            names.append(name)
    return names


def main() -> None:
    for model in {model for matchup in MATCHUPS for model in matchup}:
        assert (CONFIG / 'ai' / f'{model}.json').read_bytes() == (
            ROOT.parent / 'config' / 'ai' / f'{model}.json').read_bytes()
    for name, matchup in zip(prepare(), MATCHUPS):
        print(f'Starting {name}: {matchup}', flush=True)
        subprocess.run(['/tmp/v07_recovery_sim', '-config', str(CONFIG), '-quiet',
                        '-parallel', '12', name], cwd=ROOT,
                       env={**os.environ, 'GOMAXPROCS': '12'}, check=True)
        rows = [json.loads(line) for line in
                (ROOT / f'outputs/{name}/results.json').read_text().splitlines()]
        assert len(rows) == 30 and all(not row['Error'] for row in rows)
        print(f'Completed {name}: {len(rows)} games', flush=True)


if __name__ == '__main__':
    main()
