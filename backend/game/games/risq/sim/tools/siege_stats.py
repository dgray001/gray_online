import json
from itertools import combinations
from pathlib import Path
from typing import Any

Record = dict[str, Any]


def load_definitions(config: str) -> dict[str, list[Record]]:
    return {name: json.loads((Path(config) / f'{name}.json').read_text())
            for name in ('units', 'buildings', 'bonuses', 'techs')}


def applies(effect: Record, unit: Record) -> bool:
    return (unit['unit_id'] in effect.get('affects_unit_ids', [])
            or unit['unit_type'] in effect.get('affects_unit_types', []))


def equal(expression: str, value: float) -> str:
    return f'max(0, 1 - max(({expression}) - {value}, {value} - ({expression})))'


def damage(attacker: Record, target: Record, definitions: dict[str, list[Record]], techs: tuple[int, ...] = ()) -> float:
    category = 'building' if 'building_id' in target else target['unit_type']
    bonuses = [b for b in definitions['bonuses'] if applies(b, attacker) and category in b['target_types']]
    upgrades = [t for t in definitions['techs'] if t['tech_id'] in techs and applies(t, attacker)]
    attack, defense = 0.0, 0.0
    for kind in attacker['attack_type'].split('_'):
        attack += attacker.get(f'attack_{kind}', 0) + sum(b.get(f'bonus_attack_{kind}', 0) for b in bonuses + upgrades)
        penetration = attacker.get(f'penetration_{kind}', 0) + sum(t.get(f'bonus_penetration_{kind}', 0) for t in upgrades)
        defense += target.get(f'defense_{kind}', 0) * max(0, 1 - penetration / 100)
    reference = attack - defense if defense / attack <= 0.5 else attack * 4 ** (-defense / attack)
    return 3 * max(0.1, reference / 10)


def damage_expression(attacker: Record, target: Record, definitions: dict[str, list[Record]]) -> str:
    value = f'{damage(attacker, target, definitions):.6f}'
    techs = [t['tech_id'] for t in definitions['techs'] if applies(t, attacker)
             and any(key.startswith(('bonus_attack_', 'bonus_penetration_')) for key in t)]
    for size in range(1, len(techs) + 1):
        for selected in combinations(techs, size):
            gain = sum((-1) ** (size - count) * damage(attacker, target, definitions, subset)
                       for count in range(size + 1) for subset in combinations(selected, count))
            value += f' + {gain:.6f} * ' + ' * '.join(f'var(tech_researched_{tech})' for tech in selected)
    return f'({value})'


def hit_table(definitions: dict[str, list[Record]]) -> str:
    targets = definitions['units'] + definitions['buildings']
    villager = next(u for u in definitions['units'] if u['unit_id'] == 1)
    farming = next(t for t in definitions['techs'] if t['tech_id'] == 1)
    upgraded = dict(villager, max_health=villager['max_health'] + farming['bonus_max_health'])
    for kind in ('blunt', 'piercing', 'magic'):
        upgraded[f'defense_{kind}'] = villager.get(f'defense_{kind}', 0) + farming.get(f'bonus_defense_{kind}', 0)
    targets.append(upgraded)
    groups = {health: [t for t in targets if t['max_health'] == health] for health in sorted({t['max_health'] for t in targets})}
    terms = []
    for attacker in definitions['units']:
        for health, group in groups.items():
            estimates = [damage_expression(attacker, target, definitions) for target in group]
            estimate = estimates[0] if len(estimates) == 1 else f"min({', '.join(estimates)})"
            terms.append(f"{equal('var(unit_id)', attacker['unit_id'])} * {equal('var(target_max_health)', health)} * {estimate}")
    return ' + '.join(terms)


def utility(unit: Record, definitions: dict[str, list[Record]], building_weight: float, survival_weight: float) -> str:
    terms = [f"(1 + var(enemy_units_visible_{t['unit_id']})) * {damage_expression(unit, t, definitions)}"
             for t in definitions['units'] if t['unit_type'] != 'economic']
    terms += [f"{building_weight} * var(enemy_buildings_known_{t['building_id']}) * {damage_expression(unit, t, definitions)}"
              for t in definitions['buildings']]
    cost = ' + '.join(f"{amount} * (1 + {amount} / max({amount}, {resource}))"
                      for resource, amount in unit['cost'].items())
    survival = unit['max_health'] ** survival_weight
    reach = 1.25 if unit.get('range', 'zone') != 'zone' else 1.0
    tempo = unit['turn_stamina'] / unit['production_stamina']
    return f"(({ ' + '.join(terms) }) * {survival * reach * tempo:.6f} / max(1, {cost}))"
