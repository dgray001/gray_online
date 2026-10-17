import json
from collections import Counter
from pathlib import Path
from typing import Any

SIM = Path(__file__).resolve().parents[1]
RUN = 'eval_siege_20261017'
ORDER = ('defeated', 'crushed', 'losing', 'behind', 'ahead', 'winning', 'crush', 'defeat')
Game = dict[str, Any]


def load_games(name: str) -> list[Game]:
    document = json.loads((SIM / f'outputs/{name}/results.json').read_text())
    games = [game for bucket in document['Sims'] for game in bucket['Games']]
    assert all(not g['Result']['Error'] and len(g['Result']['Outcomes']) == 2 for g in games), name
    return games


def outcomes(games: list[Game]) -> dict[int, str]:
    return {g['Result']['Seed']: g['Result']['Outcomes'][g['Seats'].index(0)] for g in games}


def summarize(name: str, games: list[Game], baseline: dict[int, str] | None = None) -> Game:
    scores = outcomes(games)
    counts = Counter(scores.values())
    peaks = [max((frame[g['Seats'].index(0)]['Units'].get('401', 0) for frame in g['Result']['Timeline']), default=0) for g in games]
    players = [g['Result']['Game']['Players'][g['Seats'].index(0)] for g in games]
    row = {'name': name, 'games': len(games), 'outcomes': dict(counts),
           'ahead_or_better': sum(counts[o] for o in ORDER[4:]),
           'enemy_eliminated': counts['defeat'], 'self_eliminated': counts['defeated'],
           'severe_losses': counts['defeated'] + counts['crushed'],
           'catapult_games_observed': sum(peak > 0 for peak in peaks),
           'mean_sampled_peak_catapults': sum(peaks) / len(peaks),
           'mean_villager_idle_pct': sum(p['Economy']['VillagerIdlePct'] for p in players) / len(players),
           'mean_buildings_razed': sum(p['Razes'] for p in players) / len(players)}
    if baseline is not None:
        changes = Counter('better' if ORDER.index(score) > ORDER.index(baseline[seed]) else
                          'worse' if ORDER.index(score) < ORDER.index(baseline[seed]) else 'same'
                          for seed, score in scores.items())
        row['paired'] = {key: changes[key] for key in ('better', 'same', 'worse')}
    return row


def main() -> None:
    manifest = json.loads((SIM / f'inputs/{RUN}_manifest.json').read_text())
    baseline = load_games('eval_baseline_v07_v06_20261017')
    assert len(baseline) == 300
    seeds = {case['seed'] for case in manifest['cases']}
    selected = [g for g in baseline if g['Result']['Seed'] in seeds]
    assert len(selected) == 30
    rows = [summarize('baseline_full', baseline), summarize('baseline_sample', selected)]
    for name in manifest['scenarios']:
        games = load_games(name)
        assert len(games) == 30 and set(outcomes(games)) == seeds, name
        original_seats = {g['Result']['Seed']: g['Seats'] for g in selected}
        assert all(g['Seats'] == original_seats[g['Result']['Seed']] for g in games), name
        rows.append(summarize(name.removeprefix(RUN + '_'), games, outcomes(selected)))
    output = SIM / f'outputs/{RUN}_comparison.json'
    output.write_text(json.dumps({'draw_seed': manifest['draw_seed'], 'rows': rows}, indent=2) + '\n')
    print(json.dumps(rows, indent=2))


if __name__ == '__main__':
    main()
