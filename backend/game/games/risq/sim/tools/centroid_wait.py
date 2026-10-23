from typing import Any
from vc_placement import branch, compare, each, variable

def apply(rules: list[dict[str, Any]]) -> None:
    actions = [variable('vc_center_x', 0), variable('vc_center_y', 0), variable('vc_center_count', 0)]
    actions.append(each('units', 'vc_worker', [
        variable('vc_center_x', 'var(vc_center_x) + var(vc_worker.x)'),
        variable('vc_center_y', 'var(vc_center_y) + var(vc_worker.y)'),
        variable('vc_center_count', 'var(vc_center_count) + 1')], dict(unit_ids=[1])))
    actions.extend(variable(f'vc_center_{axis}', f'var(vc_center_{axis}) / max(1, var(vc_center_count))') for axis in ['x', 'y'])
    actions.append(variable('vc_center_best', 1e9))
    dx, dy = '(var(vc_candidate.x) - var(vc_center_x))', '(var(vc_candidate.y) - var(vc_center_y))'
    actions.append(each('spaces', 'vc_candidate', [variable('vc_center_cost', f'{dx} * {dx} + {dx} * {dy} + {dy} * {dy}'),
        branch({'not': compare('var(vc_center_cost)', 'at_least', 'var(vc_center_best)')}, [
            variable('vc_center_best', 'var(vc_center_cost)'), variable('vc_center_target_x', 'var(vc_candidate.x)'),
            variable('vc_center_target_y', 'var(vc_candidate.y)')])], dict(any=[dict(vision=0), dict(passable=True)])))
    actions.append(dict(action='move', unit_ids=[1], eligible=['gather', 'move'], targets='coordinate',
                        coordinate=dict(x='var(vc_center_target_x)', y='var(vc_center_target_y)')))
    waiting = {'all': [compare('var(building_count_1)', 'at_most', 0), compare('var(foundation_count_1)', 'at_most', 0), compare('var(had_city)', 'at_most', 0)]}
    index = next(i for i, rule in enumerate(rules) if any(a.get('action') == 'run_bucket' and a.get('bucket') == 'food' for a in rule['then']))
    rules.insert(index, dict(when=waiting, then=actions))
