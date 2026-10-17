from typing import Any

from siege_stats import damage, hit_table, utility


def policy(definitions: dict[str, list[dict[str, Any]]], approach: int, variant: int) -> tuple[list[dict[str, Any]], dict[int, float], list[int]]:
    assert approach in (1, 2, 3) and variant in (1, 2, 3)
    units = {u['unit_id']: u for u in definitions['units']}
    military = [u for u in units.values() if u['unit_type'] != 'economic']
    escorts = [u['unit_id'] for u in military if u.get('range', 'zone') == 'zone']
    escort = (8, 6, 4)[variant - 1]
    building = (0.5, 1.0, 1.5)[variant - 1]
    survival = (0.75, 0.5, 0.25)[variant - 1] if approach == 3 else 0.5
    values = {u['unit_id']: utility(u, definitions, building, survival) for u in military}
    infantry = ' + '.join(f'var(population_{uid})' for uid in escorts)
    actions = [{'action': 'set_var', 'name': 'siege_escort', 'value': infantry}]
    actions += [{'action': 'set_var', 'name': f'siege_value_{uid}', 'value': value} for uid, value in values.items()]
    cap = (0.15, 0.22, 0.30)[variant - 1]
    share = f'{cap} * (0.4 + 0.6 * min(1, var(enemy_buildings_known) / 3))'
    if approach == 2:
        share = f"min(0.4, {(0.8, 1.0, 1.2)[variant - 1]} * var(siege_value_401) / max(0.01, " + ' + '.join(f'var(siege_value_{uid})' for uid in units if uid != 1) + '))'
    if approach == 3:
        share = f"min(0.45, var(siege_value_401) / max(0.01, " + ' + '.join(f'var(siege_value_{u["unit_id"]})' for u in military) + '))'
    share += f' * min(1, max(0, var(siege_escort) / {escort} - 1)) / (1 + var(enemy_units_visible_military_within_1))'
    actions += [{'action': 'set_var', 'name': 'siege_share', 'value': share},
                {'action': 'set_var', 'name': 'siege_need', 'value': 'max(0, var(siege_escort) * var(siege_share) / max(0.1, 1 - var(siege_share)) - var(population_401))'}]
    weights = {101: 1.0, 102: 2.0, 103: 4.0, 401: 1.5}
    if approach == 3:
        strength = {u['unit_id']: (u['max_health'] * u['turn_stamina'] * sum(damage(u, t, definitions) for t in military)) ** 0.5 for u in military}
        weights = {uid: value / strength[101] for uid, value in strength.items()}
    return actions, weights, escorts


def siege_orders(definitions: dict[str, list[dict[str, Any]]], escaping: dict[str, Any]) -> list[dict[str, Any]]:
    active = {'not': escaping}
    eligible = ['move', 'attack_unit', 'attack_building', 'attack_space', 'attack_zone', 'auto_attack_unit', 'auto_attack_building']
    common = {'action': 'attack', 'unit_ids': [401], 'eligible': eligible, 'min_score': 1,
              'score': '100 - 1000 * var(target_distance) - var(unit_distance) / 100 + 5 * (1 - var(target_health) / max(1, var(target_max_health)))',
              'spread': {'capacity': 'max(0.1, var(target_health))', 'weight': hit_table(definitions), 'queue': 3, 'overflow': 'round_robin'}}
    fighting = {'value_at_least': {'value': 'var(enemy_units_visible_military_within_0_of_unit)', 'amount': 1}}
    return [{'when': active, 'then': [
        {'action': 'set_unit_behavior', 'unit_ids': [401], 'stance': 'stand_ground', 'attack_back': True, 'interrupt_current': False},
        dict(common, targets='enemy_units', target_unit_types=['military'], unit_when=fighting),
        dict(common, targets='enemy_buildings', unit_when={'not': fighting}),
    ]}]
