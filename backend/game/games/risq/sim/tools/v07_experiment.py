import json
import os
import subprocess
import sys
from pathlib import Path

name, games, *tuning = sys.argv[1:]
assert name.startswith('v07_')
root = Path(__file__).resolve().parents[1]
config = 'v0.7_' + name.removeprefix('v07_')
subprocess.run([sys.executable, str(root / 'tools/gen_v07.py'), f'out={root.parent}/config/ai/{config}.json', *tuning], check=True)
scenario = json.loads((root / 'inputs/eval.json').read_text())
scenario['players'][0]['ai_config'] = config
scenario['games_per_bucket'] = int(games)
scenario['buckets'] = scenario['buckets'][:int(os.environ.get('V07_MAPS', '10'))]
if 'V07_BUCKET' in os.environ:
    bucket_index = int(os.environ['V07_BUCKET'])
    scenario['buckets'], scenario['seed'] = [scenario['buckets'][bucket_index]], scenario['seed'] + 1000 * bucket_index
(root / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
subprocess.run(['go', 'run', '.', '-quiet', '-parallel', '4', name], cwd=root, env={**os.environ, 'GOMAXPROCS': '4'}, check=True)
