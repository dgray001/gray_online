from __future__ import annotations

import json
from collections import Counter

from v07_generalist import MODELS, ROOT

RANK = {name: index for index, name in enumerate(
    ('defeated', 'crushed', 'losing', 'behind', 'ahead', 'winning', 'crush', 'defeat'))}


def games(name: str) -> dict[tuple[str, int], dict]:
    rows = map(json.loads, (ROOT / f'outputs/{name}/results.json').read_text().splitlines())
    return {(row['Sim'].split('__')[0], row['Seed']): row for row in rows}


def outcome(row: dict) -> str:
    return row['Outcomes'][row['Seats'].index(0)]


def paired(rows: dict, baseline: dict) -> dict[str, int]:
    changes: Counter[str] = Counter()
    for key, row in rows.items():
        assert row['Seats'] == baseline[key]['Seats']
        delta = RANK[outcome(row)] - RANK[outcome(baseline[key])]
        changes['better' if delta > 0 else 'worse' if delta < 0 else 'same'] += 1
    return dict(changes)


def main() -> None:
    summaries: list[dict] = []
    for model in MODELS:
        label = model.removeprefix('v0.7_')
        for opponent, parent_offset, official in [('v06', 1, 'v07_frozen_l'), ('v07', 3, 'v07_ablation_7')]:
            name = f'v07_generalist_{label}_{opponent}'
            if not (ROOT / f'outputs/{name}/results.json').exists():
                continue
            rows = games(name)
            assert len(rows) == 30 and all(not row['Error'] for row in rows.values())
            counts = Counter(outcome(row) for row in rows.values())
            parent = f"v07_ablation_{parent_offset + int(label.startswith('b'))}"
            summary = {'Model': model, 'Opponent': opponent, 'Games': len(rows), 'Outcomes': dict(counts),
                       'AheadOrBetter': sum(counts[name] for name in ('ahead', 'winning', 'crush', 'defeat')),
                       'CrushOrElimination': counts['crush'] + counts['defeat'],
                       'SevereLosses': counts['crushed'] + counts['defeated'],
                       'ParentChanges': paired(rows, games(parent)), 'OfficialChanges': paired(rows, games(official))}
            summaries.append(summary)
            print(json.dumps(summary))
    (ROOT / 'outputs/v07_generalist_summary.json').write_text(json.dumps(summaries, indent=2) + '\n')


if __name__ == '__main__':
    main()
