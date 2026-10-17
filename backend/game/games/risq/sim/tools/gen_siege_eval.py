import json
import random
from pathlib import Path
from typing import Any

SIM = Path(__file__).resolve().parents[1]
RUN = 'eval_siege_20261017'
DRAW_SEED = 20261017


def sample_cases(baseline: dict[str, Any]) -> list[dict[str, Any]]:
    rng = random.Random(DRAW_SEED)
    cases = []
    for index, bucket in enumerate(baseline['buckets']):
        for game in sorted(rng.sample(range(baseline['games_per_bucket']), 3)):
            seed = baseline['seed'] + index * 1000 + game
            cases.append(dict(bucket, name=f"{bucket['name']}_seed_{seed}", seed=seed, swap=bool(game % 2)))
    return cases


def main() -> None:
    baseline = json.loads((SIM / 'inputs/eval_baseline_v07_v06_20261017.json').read_text())
    cases = sample_cases(baseline)
    configs = sorted((SIM / '../config/ai/tmp').glob('v07_siege_*.json'))
    assert len(cases) == 30 and len(configs) == 9
    names = []
    for config in configs:
        name = f'{RUN}_{config.stem.removeprefix("v07_siege_")}'
        scenario = dict(baseline, games_per_bucket=1, swap_seats=False, buckets=cases,
                        players=[{'ai_config': f'tmp/{config.stem}'}, {'ai_config': 'v0.6'}])
        (SIM / f'inputs/{name}.json').write_text(json.dumps(scenario, indent=2) + '\n')
        names.append(name)
    manifest = {'draw_seed': DRAW_SEED, 'cases': cases, 'scenarios': names}
    (SIM / f'inputs/{RUN}_manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print('\n'.join(names))


if __name__ == '__main__':
    main()
