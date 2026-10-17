"""Replays one evaluator nomad seed with its settings and alternating seats."""
import json
import os
import subprocess
import sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
bucket = json.loads((root / 'inputs/eval.json').read_text())['buckets'][7]
steps = json.loads((root.parent / 'config/maps/scripted/triangle.json').read_text())
steps[1:1] = [{'step': 'define', 'params': {'name': key, 'value': value}} for key, value in bucket['settings'].items()]
map_name = 'v0.7_nomad_replay'
(root.parent / f'config/maps/scripted/{map_name}.json').write_text(json.dumps(steps) + '\n')
config, name = sys.argv[1:3] if len(sys.argv) > 2 else ('v0.7', 'v07_nomad_debug')
seed = int(sys.argv[3]) if len(sys.argv) > 3 else 107001
players = [{'ai_config': config}, {'ai_config': 'v0.6'}]
scenario = {'seed': seed, 'fixed_seed': True, 'iterations': 1, 'max_turns': 100, 'map': f'script:{map_name}', 'players': players[::-1] if seed % 2 else players, 'metrics': ['production', 'fights']}
(root / f'inputs/{name}.json').write_text(json.dumps(scenario) + '\n')
subprocess.run(['go', 'run', '.', '-debug', name], cwd=root, env={**os.environ, 'GOMAXPROCS': '2'}, check=True)
