from math import isfinite
from typing import Any

Record = dict[str, Any]
DEFAULTS = dict(vc_site_walk=3.617635, vc_site_food=1.148617, vc_site_food_second=0.093866,
                vc_site_build=68.95007, vc_site_limit=6.260566, vc_age_scale=3.134538, vc_site_threat=10.0)


def variable(name: str, value: Any) -> Record:
    return dict(action='set_var', name=name, value=value)


def compare(value: Any, operator: str, amount: Any) -> Record:
    return {f'value_{operator}': dict(value=value, amount=amount)}


def branch(when: Record, actions: list[Record]) -> Record:
    return dict(action='if', when=when, then=actions)


def coordinate(name: str) -> Record:
    return dict(x=f'var({name}.x)', y=f'var({name}.y)')


def each(source: str, name: str, actions: list[Record], where: Record) -> Record:
    return dict(action='for_each', source=source, **{'as': name}, do=actions, where=where)


def measure(origin: Record, target: Record, score: str, name: str) -> Record:
    return dict(action='select_space', **{'from': origin}, targets='coordinate',
                coordinate=target, score=score, var_score=name)


KNOWN = dict(passable=True, vision_at_least=1)
SITE = coordinate('vc_site')
WORKERS = dict(unit_count_at_least=1)


def walking() -> list[Record]:
    return [variable('vc_walk', 0), each('spaces', 'vc_workers', [
        measure(SITE, coordinate('vc_workers'),
                'var(target_distance) * var(population_1_within_0_of_target)', 'vc_step'),
        variable('vc_walk', 'var(vc_walk) + var(vc_step)')], WORKERS),
        variable('vc_walk', 'var(vc_walk) / max(1, var(vils))')]


def food_distances() -> list[Record]:
    return [variable('vc_food_first', 'var(vc_distance_bound)'),
            variable('vc_food_second', 'var(vc_distance_bound)'),
            each('resources', 'vc_food', [
                measure(SITE, coordinate('vc_food'), 'var(target_distance)', 'vc_step'),
                variable('vc_food_second', 'min(var(vc_food_second), max(var(vc_food_first), var(vc_step)))'),
                variable('vc_food_first', 'min(var(vc_food_first), var(vc_step))')], dict(category='food'))]


def placement_cost(parameters: dict[str, float], threat: str) -> list[Record]:
    cost = (f"{parameters['vc_site_walk']} * (var(vc_walk) - var(vc_walk_min))"
            f" + {parameters['vc_site_food']} * var(vc_food_first)"
            f" + {parameters['vc_site_food_second']} * var(vc_food_second)"
            f" + {parameters['vc_site_build']} * (var(vc_build_time) - 1)")
    extra = f"{parameters['vc_site_threat']} * var(had_city) * ({threat})" if parameters['evacuate'] else '0'
    extra += f" - {parameters['vc_cluster']} * var(population_1_within_1_of_target)"
    return [variable('vc_build_time', 0),
            branch({'not': compare('var(vc_site.build_speed_modifier)', 'at_most', 0)},
                   [variable('vc_build_time', '1 / var(vc_site.build_speed_modifier)')]),
            *walking(), *food_distances(), measure(SITE, SITE, extra, 'vc_site_extra'),
            variable('vc_site_cost', cost + ' + var(vc_site_extra)')]


def free_zone() -> list[Record]:
    scan = each('zones', 'vc_zone', [variable('vc_has_zone', 1)],
                dict(building=False, resource=False, within=0, **{'from': SITE}))
    scan['max'] = 1
    return [variable('vc_has_zone', 0), scan]


def select_candidate(parameters: dict[str, float], threat: str) -> Record:
    later = {'any': [compare('var(vc_attempted)', 'at_most', 0),
                    {'not': compare('var(vc_site_cost)', 'at_most', 'var(vc_last_cost)')},
                    {'all': [compare('var(vc_site_cost)', 'equals', 'var(vc_last_cost)'),
                             {'not': compare('var(vc_site_index)', 'at_most', 'var(vc_last_index)')}]}]}
    better = {'any': [compare('var(vc_found)', 'at_most', 0),
                     {'not': compare('var(vc_site_cost)', 'at_least', 'var(vc_best_cost)')}]}
    save = [variable('vc_found', 1), variable('vc_best_cost', 'var(vc_site_cost)'),
            variable('vc_best_x', 'var(vc_site.x)'), variable('vc_best_y', 'var(vc_site.y)'),
            variable('vc_best_index', 'var(vc_site_index)')]
    delta_x, delta_y = '(var(vc_site.x) - var(vc_home_x))', '(var(vc_site.y) - var(vc_home_y))'
    bad_x, bad_y = '(var(vc_bad_x) - var(vc_home_x))', '(var(vc_bad_y) - var(vc_home_y))'
    away = f'{delta_x} * {bad_x} + {delta_y} * {bad_y} + ({delta_x} + {delta_y}) * ({bad_x} + {bad_y})'
    save.extend(variable(f'vc_best_{name}', f'var({source})') for name, source in
                [('walk', 'vc_walk'), ('food_first', 'vc_food_first'),
                 ('food_second', 'vc_food_second'), ('build_speed', 'vc_site.build_speed_modifier')])
    allowed = {'all': [compare('var(vc_has_zone)', 'at_least', 1),
                      {'any': [compare('land', 'at_most', 0), compare('var(vc_owned_near)', 'at_least', 1)]},
                      {'not': compare('var(vc_site.build_speed_modifier)', 'at_most', 0)},
                      {'any': [compare('var(vc_ban)', 'at_most', 0), {'not': compare(away, 'at_least', 0)}]}]}
    body = [variable('vc_site_index', 'var(vc_site_index) + 1'), *free_zone(),
            variable('vc_owned_near', {'count_spaces': dict(where=dict(owner='own'), within=1, **{'from': SITE})}),
            *placement_cost(parameters, threat), branch({'all': [allowed, later, better]}, save)]
    where = dict(all=[KNOWN, dict(any=[dict(owner='own'), dict(owner='unowned')])])
    return each('spaces', 'vc_site', body, where)


def placement_attempt(build: Record, parameters: dict[str, float], threat: str) -> list[Record]:
    build = dict(build, score='0 - var(target_distance)', min_score=0,
                 eligible=['gather', 'move'],
                 **{'from': dict(x='var(vc_best_x)', y='var(vc_best_y)')})
    for key in ('direction', 'direction_from'):
        build.pop(key, None)
    accepted = compare('var(vc_best_cost)', 'at_most', 'var(vc_cost_limit)')
    commit = [variable('vc_bank_before', 'food + wood + stone + gold'), build,
              variable('vc_done', 'min(1, max(0, var(vc_bank_before) - food - wood - stone - gold))'),
              variable('vc_attempted', 1), variable('vc_last_cost', 'var(vc_best_cost)'),
              variable('vc_last_index', 'var(vc_best_index)')]
    decision = dict(branch(accepted, commit), **{'else': [variable('vc_done', 1)]})
    chosen = dict(branch(compare('var(vc_found)', 'at_least', 1), [decision]),
                  **{'else': [variable('vc_done', 1)]})
    attempt = [variable('vc_found', 0), variable('vc_site_index', 0),
               select_candidate(parameters, threat), chosen]
    minimum = each('spaces', 'vc_site', [*walking(),
                   variable('vc_walk_min', 'min(var(vc_walk_min), var(vc_walk))')], KNOWN)
    retry = each('spaces', 'vc_retry', [branch(compare('var(vc_done)', 'at_most', 0), attempt)], KNOWN)
    return [variable('vc_distance_bound', {'count_spaces': {}}),
            variable('vc_walk_min', 'var(vc_distance_bound)'), minimum,
            dict(action='select_space', **{'from': 'home'}, targets='home', score=0,
                 var_x='vc_home_x', var_y='vc_home_y'),
            variable('vc_attempted', 0), variable('vc_done', 0), retry]


def builder_availability(build: Record) -> list[Record]:
    bucket = 'vc_builder_probe'
    fill = dict(action='fill_bucket', bucket=bucket, eligible=['gather', 'move'], unit_ids=[1])
    if 'unit_when' in build:
        fill['unit_when'] = build['unit_when']
    return [dict(action='set_bucket', bucket=bucket, size=1, task=variable('vc_has_builder', 0)),
            fill, variable('vc_has_builder', f'var(bucket_size_{bucket})'),
            dict(action='empty_bucket', bucket=bucket)]


def apply(rules: list[Record], parameters: dict[str, float], threat: str) -> None:
    for name in DEFAULTS:
        if not isfinite(parameters[name]) or parameters[name] < 0:
            raise ValueError(f'{name} must be finite and nonnegative')
    if parameters['vc_age_scale'] <= 0:
        raise ValueError('vc_age_scale must be positive')
    expanded: list[Record] = []
    for rule in rules:
        actions: list[Record] = []
        for action in rule['then']:
            if action.get('action') == 'build' and action.get('building_id') == 1:
                funded = {'all': [compare(resource, 'at_least', f'var(building_cost_{resource}_1)')
                                  for resource in ('food', 'wood', 'stone', 'gold')]}
                rule['when'] = {'all': [rule['when'], funded]}
                expanded.append(dict(when=rule['when'], then=builder_availability(action)))
                rule['when'] = {'all': [rule['when'], compare('var(vc_has_builder)', 'at_least', 1)]}
                actions.extend(placement_attempt(action, parameters, threat))
            else:
                actions.append(action)
        rule['then'] = actions
        expanded.append(rule)
    rules[:] = expanded
