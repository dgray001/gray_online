import re
from typing import Any

from siege_policy import policy, siege_orders
from siege_stats import load_definitions


def power_expression(value: str, weights: dict[int, float]) -> str:
    pattern = r'var\((population|enemy_units_visible|idle_units)_101([^)]*)\) \+ 2 \* var\(\1_102\2\) \+ 4 \* var\(\1_103\2\)'
    def replace(match: re.Match[str]) -> str:
        prefix, suffix = match.groups()
        return ' + '.join(f'{weight:.6f} * var({prefix}_{uid}{suffix})' for uid, weight in weights.items())
    value = re.sub(pattern, replace, value)
    for names, counter in ((('B', 'Pi', 'Hv'), 'population'), (('eB', 'eP', 'eH'), 'enemy_units_visible')):
        old = f'var({names[0]}) + 2 * var({names[1]}) + 4 * var({names[2]})'
        new = ' + '.join(f'{weights[uid]:.6f} * var({name})' for uid, name in zip((101, 102, 103), names))
        new += f" + {weights[401]:.6f} * var({'eS' if counter == 'enemy_units_visible' else 'population_401'})"
        value = value.replace(old, new)
    return value


def integrate(rules: list[dict[str, Any]], config: str, approach: int, variant: int, escaping: dict[str, Any]) -> list[dict[str, Any]]:
    definitions = load_definitions(config)
    state, weights, escorts = policy(definitions, approach, variant)
    catapult = next(u for u in definitions['units'] if u['unit_id'] == 401)
    producer = next(b for b in definitions['buildings'] if any(p.get('kind') == 'unit' and p['id'] == 401 for p in b.get('produces', [])))
    rate = producer['turn_stamina'] / catapult['production_stamina']
    rules = transform(rules, approach, weights, escorts, rate)
    state.append({'action': 'set_var', 'name': 'eS', 'value': 'max(var(eS) - 0.3, var(enemy_units_visible_401))', 'persist': True})
    rules.insert(0, {'when': {'always': {}}, 'then': state})
    for rule in rules:
        for index, action in enumerate(rule['then']):
            if action.get('name') == 'redoubt_H':
                rule['then'].insert(index + 1, {'action': 'set_var', 'name': 'redoubt_C', 'value': 'var(redoubts) * min(1, var(siege_need)) * var(siege_share)'})
            if action.get('action') == 'create' and action.get('unit_id') == 103:
                action['queue'] = 1
    ready = {'all': [{'value_at_least': {'value': expression, 'amount': amount}} for expression, amount in
                     (('var(siege_need)', 1), ('var(headroom)', 1), ('wood', catapult['cost']['wood']), ('stone', catapult['cost']['stone']))]}
    index = next(i for i, rule in enumerate(rules) if any(a.get('action') == 'create' and a.get('unit_id') == 103 for a in rule['then']))
    rules[index]['when'] = {'all': [rules[index]['when'], {'not': ready}]}
    rules.insert(index, {'when': ready, 'then': [{'action': 'create', 'unit_id': 401, 'queue': 1}]})
    rules.extend(siege_orders(definitions, transform(escaping, approach, weights, escorts, rate)))
    return rules


def variable_value(name: str, value: str, approach: int, rate: float) -> str:
    changes = {
        'mil': f'({value}) + var(population_401)',
        'eMil': f'({value}) + var(eS)',
        'redoubt_spend': f'({value}) * (1 - var(siege_share)) + (var(unit_cost_wood_401) + var(unit_cost_stone_401)) * {rate} * var(siege_share)',
        'redoubt_gold_ok': f'max({value}, min(1, var(siege_need)))',
        'shareH': f'max({value}, var(siege_share))',
        'tH': f'max(var(Hv), ({value}) - var(siege_need))',
        'redoubt_H': f'({value}) * (1 - var(siege_share))',
        'buffer_stone': 'var(unit_cost_stone_401) * min(1, var(siege_need)) * min(1, var(redoubts))',
        'buffer_wood': f'({value}) + var(unit_cost_wood_401) * min(1, var(siege_need)) * min(1, var(redoubts))',
        'prod_rate': f'({value}) + var(redoubts) * var(siege_share) * ({rate} - 0.25)',
    }
    if approach == 3 and name in ('vB', 'vP', 'vH'):
        uid = {'vB': 101, 'vP': 102, 'vH': 103}[name]
        return f'var(siege_value_{uid}) / max(0.001, var(siege_value_101))'
    return changes.get(name, value)


def transform(value: Any, approach: int, weights: dict[int, float], escorts: list[int], rate: float) -> Any:
    if isinstance(value, list):
        return [transform(item, approach, weights, escorts, rate) for item in value
                if not (isinstance(item, dict) and item.get('action') == 'drain_bucket' and item.get('to') == 'scout')]
    if isinstance(value, str):
        for resource in ('food', 'wood', 'stone'):
            old = f'var(n_redoubts) * var(unit_cost_{resource}_103) * 0.25'
            new = f'(var(n_redoubts) * ((1 - var(siege_share)) * var(unit_cost_{resource}_103) * 0.25 + var(siege_share) * var(unit_cost_{resource}_401) * {rate}))'
            value = value.replace(old, new)
        return power_expression(value, weights)
    if not isinstance(value, dict):
        return value
    value = {key: transform(item, approach, weights, escorts, rate) for key, item in value.items()}
    if value.get('action') == 'set_var':
        name = value['name']
        value['value'] = variable_value(name, str(value['value']), approach, rate)
        if name == 'simple_power':
            value['value'] += f' - {weights[401]:.6f} * var(population_401_bucket_army_within_1_of_army) * (0.25 * var(tech_researched_2) + 0.15 * var(tech_researched_3))'
        if name.endswith('_rate') and name.removesuffix('_rate') in ('food', 'wood', 'stone', 'gold'):
            resource = name.removesuffix('_rate')
            value['value'] += f' + var(redoubt_C) * var(unit_cost_{resource}_401) * {rate}'
    if value.get('action') in ('attack', 'set_unit_behavior') and value.get('unit_types') == ['military']:
        value.pop('unit_types')
        value['unit_ids'] = escorts
    if value.get('bucket') == 'scout' and value.get('action') == 'fill_bucket':
        value.pop('unit_types', None)
        value['unit_ids'] = escorts
    return value
