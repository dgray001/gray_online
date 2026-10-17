from __future__ import annotations

import json
import os
import subprocess
import sys
from collections import Counter
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CONFIG = Path('/tmp/v07_generalist_config_20261017')
MODELS = [f'v0.7_{label}' for label in ('a1', 'a2', 'a3', 'b1', 'b2', 'b3')]


def prepare(mode: str) -> list[tuple[str, str]]:
    baseline = json.loads((ROOT / 'inputs/v07_ablation_1.json').read_text())
    fixtures: list[tuple[str, str]] = []
    for model in MODELS:
        for opponent in (['v0.6', 'v0.7'] if mode == 'eval' else ['v0.6']):
            label = model.removeprefix('v0.7_')
            name = f"v07_generalist_{label}_{opponent.replace('.', '') if mode == 'eval' else 'debug'}"
            players = [{'ai_config': model}, {'ai_config': opponent}]
            scenario = {**baseline, 'players': players}
            if mode == 'smoke':
                economy = label.startswith('a')
                players[1]['ai_config'] = 'v0.6' if economy else 'v0.7'
                scenario = {'seed': 101002 if economy else 101003, 'fixed_seed': True,
                            'iterations': 1, 'map': 'script:hexagon', 'max_turns': 100,
                            'players': players if economy else players[::-1],
                            'metrics': ['production', 'fights']}
            (ROOT / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
            fixtures.append((name, model))
    return fixtures


def run(name: str, model: str, mode: str) -> None:
    print(f'Starting {name}', flush=True)
    subprocess.run(['/tmp/v07_recovery_sim', '-config', str(CONFIG),
                    '-debug' if mode == 'smoke' else '-quiet', '-parallel', '12', name],
                   cwd=ROOT, env={**os.environ, 'GOMAXPROCS': '12'},
                   stdout=subprocess.DEVNULL, check=True)
    output = ROOT / f'outputs/{name}'
    if mode == 'smoke':
        data = json.loads((output / 'results.json').read_text())
        assert all(not game['Result']['Error'] for sim in data['Sims'] for game in sim['Games'])
        counts = {p['Name']: p['Outcomes'] for p in data['Overall']['Players']}
    else:
        rows = list(map(json.loads, (output / 'results.json').read_text().splitlines()))
        assert len(rows) == 30 and all(not row['Error'] for row in rows)
        counts = dict(Counter(row['Outcomes'][row['Ais'].index(model)] for row in rows))
    for log_name in ('sim.log', 'debug.log') if mode == 'smoke' else ('sim.log',):
        with (output / log_name).open() as log:
            assert not any(any(token in line for token in ('ai config:', 'ai expression', 'ai:')) for line in log)
    print(f'Completed {name}: {counts}', flush=True)


def prepare_nomad() -> list[tuple[str, str]]:
    baseline = json.loads((ROOT / 'inputs/v07_promoted_nomad_smoke.json').read_text())
    fixtures: list[tuple[str, str]] = []
    for label in ('a3', 'b3'):
        model, name = f'v0.7_{label}', f'v07_generalist_{label}_nomad'
        scenario = {**baseline, 'seed': 107001,
                    'players': [{'ai_config': 'v0.6'}, {'ai_config': model}]}
        (ROOT / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
        fixtures.append((name, model))
    return fixtures


def main() -> None:
    mode = sys.argv[1]
    assert mode in ('smoke', 'eval')
    for model in MODELS + ['v0.6', 'v0.7']:
        assert (CONFIG / 'ai' / f'{model}.json').read_bytes() == (
            ROOT.parent / 'config' / 'ai' / f'{model}.json').read_bytes()
    for name, model in prepare(mode):
        run(name, model, mode)
    if mode == 'smoke':
        for name, model in prepare_nomad():
            run(name, model, mode)


if __name__ == '__main__':
    main()
