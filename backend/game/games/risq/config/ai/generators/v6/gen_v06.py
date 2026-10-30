"""Generates config/ai/v0.6.json from v0.6_source.json using the four retained actions."""
from __future__ import annotations
import copy, json, os, re, sys
from typing import Any, Callable
Record = dict[str, Any]
KINDS = {'build', 'set_unit_behavior', 'unit_stop', 'garrison', 'ungarrison', 'create', 'research', 'hire', 'buckets', 'repair', 'build_foundations', 'cancel_foundations', 'renew', 'gather', 'drain_bucket', 'for_each', 'select_space', 'move'}
MILITARY_TYPES = ('military', 'infantry', 'archer', 'cavalry', 'siege', 'economic')
ORDER_KIND_NAMES = ('move', 'gather', 'build', 'repair', 'attack_space', 'attack_zone', 'attack_unit', 'attack_building', 'garrison', 'ungarrison', 'delete', 'renew', 'auto_attack_unit', 'auto_attack_building')
COMBAT_UNIT_IDS = (1, 101, 102, 103, 401)
PRIMARY_RELATION = 'overflow_primary'

def cnt(kind: str, where: Record) -> Record:
    return {'count': {'kind': kind, 'where': where}}

def sv(name: str, value: Any, persist: bool=False) -> Record:
    a = {'action': 'set_var', 'name': name, 'value': value}
    if persist:
        a['persist'] = True
    return a

def unit_where(action: Record, role: str) -> Record:
    """where-clause for the units a legacy action filter selects (no military default)."""
    where = {}
    for key in ('unit_ids', 'unit_types'):
        if key in action:
            where[key] = action[key]
    if 'in_bucket' in action and action.get('exclude_buckets'):
        where['if'] = '0'
    elif 'in_bucket' in action:
        where['bucket'] = action['in_bucket']
    elif action.get('exclude_buckets'):
        where['bucket'] = 'none'
    return where

def eligible_in(action: Record) -> Record:
    names = list(action.get('eligible') or [])
    return {'order_kind': names + ['none']}

class Cond:
    """Builds one boolean expression over the actor plus the lets it needs."""

    def __init__(self, role: str='actor', anchors: dict[str, str]=None) -> None:
        (self.role, self.lets, self.names, self.anchors) = (role, {}, {}, anchors or {})

    def counter(self, name: str) -> str:
        if name in self.names:
            return self.names[name]
        key = 'c%d' % len(self.lets)
        self.lets[key] = counter_spec(name, {'unit': self.role, **self.anchors})
        self.names[name] = 'var(%s.%s)' % (self.role, key)
        return self.names[name]

    def value(self, text: str) -> str:
        return re.sub('var\\((\\w+_of_unit)\\)', lambda m: self.counter(m.group(1)), text)

    def expr(self, cond: Any) -> str:
        ((key, body),) = cond.items()
        if key == 'all':
            return '(' + ' * '.join((self.expr(c) for c in body)) + ')' if body else '1'
        if key == 'any':
            return '((' + ' + '.join((self.expr(c) for c in body)) + ') > 0)'
        if key == 'not':
            return '(1 - %s)' % self.expr(body)
        if key == 'always':
            return '1'
        if key in ('value_at_least', 'value_at_most', 'value_equals'):
            op = {'value_at_least': '>=', 'value_at_most': '<=', 'value_equals': '=='}[key]
            return '((%s) %s %s)' % (self.value(body['value']), op, body['amount'])
        cmp_name = next((c for c in ('least', 'most', 'equals') if key == 'population_at_' + c or (c == 'equals' and key == 'population_equals')), None)
        if cmp_name:
            where = {'owner': 'own', 'near': {'distance': body['within'], 'of': body['from']}}
            if 'in_bucket' in body:
                where['bucket'] = body['in_bucket']
            if 'unit_ids' in body:
                where['unit_ids'] = body['unit_ids']
            let = 'c%d' % len(self.lets)
            self.lets[let] = cnt('units', where)
            return '(var(%s.%s) %s %s)' % (self.role, let, {'least': '>=', 'most': '<=', 'equals': '=='}[cmp_name], expr(body['amount']))
        raise NotImplementedError('condition ' + key)

def actors_for(action: Record, extra_if: str | None=None, role: str='actor', anchors: dict[str, str]=None) -> Record:
    """The actor selector for a legacy unit action: filters, bucket scope, unit_when."""
    where = unit_where(action, role)
    conds = [extra_if] if extra_if else []
    lets = {}
    if 'unit_when' in action:
        c = Cond(role, anchors)
        conds.append(c.expr(action['unit_when']))
        lets = c.lets
    if where.get('if') == '0':
        conds = ['0']
    if conds:
        where['if'] = ' * '.join(('(%s)' % c for c in conds))
    sel = {'kind': 'units', 'where': where}
    if lets:
        sel['let'] = lets
    return sel
STANCES = {'passive': 1, 'aggressive': 2, 'defensive': 3, 'stand_ground': 4}

def migrate_behavior(action: Record) -> Record:
    """Only military units whose current setting differs get the message, judged on the state before this turn's orders."""
    if 'target_priority' in action:
        raise NotImplementedError('behavior target_priority')
    setting = {k: action[k] for k in ('stance', 'attack_back', 'interrupt_current') if k in action}
    fields = {'stance': ('actor.stance', STANCES.get(setting.get('stance'))), 'attack_back': ('actor.attack_back', int(bool(setting.get('attack_back')))), 'interrupt_current': ('actor.interrupt_current', int(bool(setting.get('interrupt_current'))))}
    differs = ' + '.join(('(var(%s) != %s)' % fields[k] for k in setting))
    return {'action': 'select', 'actors': actors_for(action, 'var(actor.is_military) * ((%s) > 0)' % differs), 'issue': [{'set_behavior': setting}]}

def migrate_stop(action: Record) -> Record:
    spec = {}
    if 'orders' in action:
        spec['orders'] = action['orders']
    sel = {'action': 'select', 'actors': actors_for(action), 'issue': [{'stop': spec}]}
    if 'max' in action:
        raise NotImplementedError('unit_stop max')
    return sel

def migrate_garrison(action: Record) -> Record:
    where = dict(unit_where(action, 'actor'))
    where['in'] = eligible_in(action)
    conds = ['1 - var(actor.is_garrisoned)']
    lets = {}
    if 'unit_when' in action:
        c = Cond('actor')
        conds.append(c.expr(action['unit_when']))
        lets = c.lets
    where['if'] = ' * '.join(('(%s)' % c for c in conds))
    actors = {'kind': 'units', 'where': where}
    if lets:
        actors['let'] = lets
    actors['rank'] = '0 - var(actor.uid)'
    out = {'action': 'select', 'actors': actors, 'pick': {'kind': 'buildings', 'where': {'owner': 'own', 'if': ['1 - var(target.is_under_construction)', 'var(target.garrison_capacity) - var(target.garrison_count) > 0']}, 'capacity': 'var(target.garrison_capacity) - var(target.garrison_count)'}, 'match': {'score': ['0 - 1000000 * distance(actor, target, 1) - var(target.uid)']}, 'issue': [{'garrison': {}}]}
    if 'max' in action:
        raise NotImplementedError('garrison max')
    return out

def migrate_ungarrison(action: Record) -> Record:
    if 'max' in action:
        raise NotImplementedError('ungarrison max')
    actors = actors_for(action, 'var(actor.is_garrisoned)')
    actors['where']['in'] = eligible_in(action)
    return {'action': 'select', 'actors': actors, 'issue': [{'ungarrison': {}}]}

def expr(v: Any) -> str:
    return str(v).lower() if not isinstance(v, str) else v
RESOURCES = ('food', 'wood', 'stone', 'gold')
ZERO_VARS: set[str] = set()

def literal_zero_vars(model: Record) -> set[str]:
    """Vars the source sets to a literal zero; score terms weighted by them are dropped."""
    zeros: set[str] = set()
    def walk(o: Any) -> None:
        if isinstance(o, dict):
            v = o.get('value')
            if o.get('action') == 'set_var' and isinstance(v, (int, float)) and not isinstance(v, bool) and v == 0:
                zeros.add(str(o.get('name')))
            for c in o.values():
                walk(c)
        elif isinstance(o, list):
            for c in o:
                walk(c)
    walk(model['rules'])
    return zeros

def drop_zero_terms(text: str) -> str:
    """Drops top-level `var(zero) * (...)` terms so their lets are never evaluated."""
    (terms, depth, start, i) = ([], 0, 0, 0)
    while i < len(text):
        depth += text[i] == '('
        depth -= text[i] == ')'
        if depth == 0 and text.startswith(' + ', i):
            terms.append(text[start:i])
            (start, i) = (i + 3, i + 2)
        i += 1
    terms.append(text[start:])
    kept = [t for t in terms if not (m := re.match(r'\s*var\((\w+)\)\s*\*', t)) or m.group(1) not in ZERO_VARS]
    return ' + '.join(kept) if kept else '0'

ENEMY_BUILDING_SPACES_USED = False

def enemy_building_spaces() -> str:
    global ENEMY_BUILDING_SPACES_USED
    ENEMY_BUILDING_SPACES_USED = True
    return 'enemy_building_spaces'

def enemy_building_space_store() -> list[Record]:
    return [{'action': 'select', 'pick': {'kind': 'spaces', 'where': {'building_owner': 'enemy'}}, 'into': {'set': 'enemy_building_spaces'}}]

def room(kind: str, item_id: int) -> list[str]:
    """How many of an item the bank covers, per resource, from the game's own costs."""
    out = []
    for r in RESOURCES:
        c = 'config(%s, cost_%s, %d)' % (kind, r, item_id)
        out.append('((%s > 0) * max(0, round(gameVar(%s) / max(%s, 0.000001) - 0.5)) + (%s <= 0) * 1000000)' % (c, r, c, c))
    return out

def migrate_create(action: Record) -> Record:
    (unit_id, queue) = (int(action['unit_id']), expr(action.get('queue', 1)))
    add = 'min(max(0, (%s) - var(actor.queue_length)), %s)' % (queue, ', '.join(room('unit', unit_id) + ['gameVar(population_limit) - gameVar(population)']))
    where = {'can_create': unit_id, 'if': '((%s) > 1) + ((%s) <= 1) * var(actor.idle)' % (queue, queue)}
    return {'action': 'select', 'actors': {'kind': 'buildings', 'where': where}, 'issue': [{'create': {'unit_id': unit_id, 'queue': 'var(actor.queue_length) + max(0, %s)' % add}}]}

def migrate_research(action: Record) -> Record:
    (tech_id, queue) = (int(action['tech_id']), expr(action.get('queue', 1)))
    afford = ' * '.join(('(gameVar(%s) >= config(tech, cost_%s, %d))' % (r, r, tech_id) for r in RESOURCES))
    slot = '(((%s) > 1) * (var(actor.queue_length) < (%s)) + ((%s) <= 1) * var(actor.idle))' % (queue, queue, queue)
    return {'action': 'select', 'actors': {'kind': 'buildings', 'where': {'can_research': tech_id, 'if': '%s * (%s)' % (slot, afford)}, 'rank': '0 - var(actor.uid)', 'limit': 1}, 'issue': [{'research': {'tech_id': tech_id, 'queue': queue}}]}

def nat(v: Any) -> str:
    return '(2 * var(item.%s) + (var(item.%s) < 0) * (0 - 4 * var(item.%s) - 1))' % (v, v, v)
PAIR = '(%s + %s) * (%s + %s + 1) / 2 + %s' % (nat('x'), nat('y'), nat('x'), nat('y'), nat('y'))
HOME_BUILDING = {'kind': 'buildings', 'where': {'owner': 'own'}, 'rank': '0 - ((1 - (var(item.building_id) == 1)) * 1000000000000 + var(item.uid))', 'limit': 1}
HOME_AT = {'x': 'var(hx)', 'y': 'var(hy)', 'zone_x': 'var(hzx)', 'zone_y': 'var(hzy)'}
HOME_USED = False
CENTER_SEQ = 0
ENEMY_DISTANCE_USED = False
MIL_UNITS_USED = False
MIL_BUILDINGS_USED = False
MIL_SPACES_USED = False

def home_space() -> Record:
    """Saves the space at home once per turn, so home-distance reads need not scan every space."""
    return {'action': 'select', 'pick': {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': HOME_AT}}}, 'into': {'set': 'home_space', 'var': 'n_home_space'}}

def home_zone_store() -> Record:
    """Saves the zone at home once per turn, so home-zone reads need not scan every zone."""
    return {'action': 'select', 'pick': {'kind': 'zones', 'where': {'near': {'distance': 0, 'unit': 'zone', 'of': HOME_AT}}}, 'into': {'set': 'home_zone', 'var': 'n_home_zone'}}

def enemy_distance_setup() -> list[Record]:
    """Saves the distance from home to the nearest known enemy building once per turn."""
    nearest = {'min': {'kind': 'buildings', 'where': {'owner': 'enemy'}, 'of': 'distance(item, hm)', 'default': 0}}
    home = {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': {'x': 'var(hx)', 'y': 'var(hy)'}}}}
    return [sv('enemy_dist_home', 0), {'action': 'for_each', 'select': home, 'as': 'hm', 'do': [sv('enemy_dist_home', nearest)]}]

def mil_presence_sets() -> list[Record]:
    """Saves the spaces holding own military once per turn and the items standing in them, for presence-only reads."""
    spaces = {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': {'kind': 'units', 'where': {'owner': 'own', 'unit_types': ['military']}}}}}
    out = [{'action': 'select', 'pick': spaces, 'into': {'set': 'mil_presence_spaces', 'var': 'n_mil_presence_spaces'}}]
    if MIL_UNITS_USED:
        out.append({'action': 'select', 'pick': {'kind': 'units', 'where': {'owner': 'any', 'at': {'in_set': 'mil_presence_spaces'}}}, 'into': {'set': 'mil_presence_units', 'var': 'n_mil_presence_units'}})
    if MIL_BUILDINGS_USED:
        out.append({'action': 'select', 'pick': {'kind': 'buildings', 'where': {'owner': 'any', 'at': {'in_set': 'mil_presence_spaces'}}}, 'into': {'set': 'mil_presence_buildings', 'var': 'n_mil_presence_buildings'}})
    return out

def presence_membership(name: str, targets: Record, text: str) -> Any:
    """A kind-matched inSet test for the own-military-in-space counter when every use only tests presence."""
    global MIL_UNITS_USED, MIL_BUILDINGS_USED, MIL_SPACES_USED
    if name != 'population_military_within_0_of_target' or text.count('var(%s)' % name) != text.count('min(1, var(%s))' % name):
        return None
    kind = targets.get('kind')
    set_name = {'units': 'mil_presence_units', 'buildings': 'mil_presence_buildings', 'spaces': 'mil_presence_spaces'}.get(kind)
    if set_name is None:
        return None
    MIL_UNITS_USED = MIL_UNITS_USED or kind == 'units'
    MIL_BUILDINGS_USED = MIL_BUILDINGS_USED or kind == 'buildings'
    MIL_SPACES_USED = MIL_SPACES_USED or kind == 'spaces'
    return 'min(1, inSet(%s, target))' % set_name

def _units_space(owner: str) -> Record:
    return dict(kind='spaces', where={'owner': owner, 'if': 'var(item.own_units) >= 1'}, rank=['var(item.own_units)', '0 - (' + PAIR + ')'], limit=1)

def home_each(sel: Record, fields: list[tuple[str, str]]) -> list[Record]:
    """One scan reads every wanted field of the same top space or building."""
    sel = json.loads(json.dumps(sel).replace('var(item.', 'var(hs.'))
    return [sv(name, 0) for (_, name) in fields] + [{'action': 'for_each', 'select': dict(sel, limit=1), 'as': 'hs', 'do': [sv(name, 'var(hs.%s)' % f) for (f, name) in fields]}]

def home_vars() -> list[Record]:
    """Sets hx, hy, hzx, hzy: the first village center, else the first building, else the space holding most of our units."""
    out = [sv('has_b', cnt('buildings', {'owner': 'own'})), sv('n_owned', cnt('spaces', {'owner': 'own', 'if': 'var(item.own_units) >= 1'}))]
    out += home_each(HOME_BUILDING, [('x', 'hb_x'), ('y', 'hb_y'), ('zone_x', 'hb_zone_x'), ('zone_y', 'hb_zone_y')])
    out += home_each(_units_space('own'), [('x', 'ho_x'), ('y', 'ho_y')])
    out += home_each(_units_space('any'), [('x', 'ha_x'), ('y', 'ha_y')])
    for f in ('x', 'y'):
        out.append(sv('h' + f, '(var(has_b) > 0) * var(hb_%s) + (var(has_b) < 1) * ((var(n_owned) > 0) * var(ho_%s) + (var(n_owned) < 1) * var(ha_%s))' % (f, f, f)))
    out.append(sv('hzx', '(var(has_b) > 0) * var(hb_zone_x)'))
    out.append(sv('hzy', '(var(has_b) > 0) * var(hb_zone_y)'))
    return out

def migrate_hire(action: Record) -> list[Record]:
    """The legacy rule prices a mercenary at 1.3x and books exactly that; hire_adj carries the gap to the engine's real price."""
    global HOME_USED
    HOME_USED = True
    unit_id = int(action['unit_id'])
    price = '((config(unit, cost_food, %d) + config(unit, cost_wood, %d) + config(unit, cost_stone, %d) + 1.5 * config(unit, cost_gold, %d)) * 1.3)' % ((unit_id,) * 4)
    (limit, reserve) = (expr(action.get('max', 0)), expr(action.get('reserve', 0)))
    count = 'researched(101) * min((%s > 0) * (%s) + (%s <= 0) * 1000000, gameVar(population_limit) - gameVar(population), max(0, round((gameVar(gold) + var(hire_adj) - (%s)) / %s + 0.000000001 - 0.5)))' % (limit, limit, limit, reserve, price)
    pick = lambda f: '(var(mm.id) == %d) * var(mm.%s) + (var(mm.id) != %d) * var(%s)' % (unit_id, f, unit_id, 'hire_cost' if f == 'cost_gold' else 'hire_listed')
    return [sv('hire_n', count), {'action': 'select', 'pick': {'kind': 'zones', 'where': {'in_set': 'home_zone'}}, 'issue': [{'hire': {'unit_id': unit_id, 'count': 'var(hire_n)'}}]}, sv('hire_cost', 0), sv('hire_listed', 0), {'action': 'for_each', 'source': 'mercenaries', 'as': 'mm', 'do': [sv('hire_cost', pick('cost_gold')), sv('hire_listed', '(var(mm.id) == %d) + (var(mm.id) != %d) * var(hire_listed)' % (unit_id, unit_id))]}, sv('hire_adj', 'var(hire_adj) + var(hire_n) * var(hire_listed) * (%s - var(hire_cost))' % price)]
CATEGORY = {'food': 0, 'wood': 1, 'stone': 2, 'gold': 3}
KEYS = {'nx': '2 * max(var(target.x), 0 - var(target.x)) - (var(target.x) < 0)', 'ny': '2 * max(var(target.y), 0 - var(target.y)) - (var(target.y) < 0)', 'nzx': '2 * max(var(target.zone_x), 0 - var(target.zone_x)) - (var(target.zone_x) < 0)', 'nzy': '2 * max(var(target.zone_y), 0 - var(target.zone_y)) - (var(target.zone_y) < 0)', 'space_key': '(var(target.nx) + var(target.ny)) * (var(target.nx) + var(target.ny) + 1) / 2 + var(target.ny)', 'local_key': '(var(target.nzx) + var(target.nzy)) * (var(target.nzx) + var(target.nzy) + 1) / 2 + var(target.nzy)', 'zone_key': '(2 * var(target.space_key) + 2 * var(target.local_key)) * (2 * var(target.space_key) + 2 * var(target.local_key) + 1) / 2 + 2 * var(target.local_key)'}
BUCKET_TASKS = {}

def scan_buckets(rules: list[Record]) -> None:

    def walk(a: Record) -> Record:
        if isinstance(a, dict):
            if a.get('action') == 'set_bucket':
                variants = BUCKET_TASKS.setdefault(a['bucket'], [])
                if a['task'] not in variants:
                    variants.append(a['task'])
            for v in a.values():
                walk(v)
        elif isinstance(a, list):
            for v in a:
                walk(v)
    walk(rules)

def task_index(bucket: str, task: Record) -> int:
    return BUCKET_TASKS[bucket].index(task)

def migrate_set_bucket(action: Record) -> list[Record]:
    name = action['bucket']
    size = 'max(0, round(%s))' % expr(action['size'])
    return [sv('size_' + name, size, True), sv('task_' + name, task_index(name, action['task']), True), {'action': 'select', 'pick': {'kind': 'units', 'where': {'bucket': name}, 'rank': '0 - var(target.uid)', 'skip': 'var(size_%s)' % name}, 'into': {'bucket': name, 'mode': 'remove'}}]

def migrate_empty_bucket(action: Record) -> Record:
    name = action['bucket']
    return {'action': 'select', 'pick': {'kind': 'units', 'where': {'bucket': name}, 'limit': 0}, 'into': {'bucket': name, 'mode': 'replace'}}

def migrate_fill_bucket(action: Record) -> Record:
    name = action['bucket']
    where = unit_where({k: v for (k, v) in action.items() if k in ('unit_ids', 'unit_types')}, 'target')
    where['bucket'] = 'none'
    where['in'] = eligible_in(action)
    (lets, conds) = ({}, [])
    if 'unit_when' in action:
        c = Cond('target')
        conds.append(c.expr(action['unit_when']))
        lets.update(c.lets)
    rank = ['0 - var(target.uid)']
    tasks = BUCKET_TASKS.get(name, [])
    if tasks and all((t['action'] == 'gather' for t in tasks)):
        cat = CATEGORY[tasks[0]['category']]
        lets['gcat'] = {'max': {'kind': 'gatherables', 'where': {'if': 'var(item.uid) == var(target.order_target_uid)'}, 'of': 'var(item.category)', 'default': '0 - 1'}}
        rank = ['0 - ((var(target.order_kind) == 0 - 1) + 2 * (var(target.order_kind) != 0 - 1) * (var(target.gcat) != %d))' % cat, '0 - var(target.uid)']
    if conds:
        where['if'] = ' * '.join(('(%s)' % c for c in conds))
    pick = {'kind': 'units', 'where': where, 'rank': rank, 'limit': 'var(size_%s) - var(bucket_size_%s)' % (name, name)}
    if lets:
        pick['let'] = lets
    return {'action': 'select', 'pick': pick, 'into': {'bucket': name, 'mode': 'add'}}

def migrate_drain_bucket(action: Record) -> list[Record]:
    """Lowest-id members of each source move to the target while it is short, at most max in total; allow_reserved only."""
    (to, limit) = (action['to'], expr(action.get('max', 0)))
    cap = '((%s) > 0) * (%s) + ((%s) <= 0) * 1000000' % (limit, limit, limit)
    out = [sv('dr_left', 'max(0, var(size_%s) - var(bucket_size_%s))' % (to, to)), sv('dr_cap', cap)]
    for src in action['from']:
        pick = lambda : {'kind': 'units', 'where': {'bucket': src}, 'rank': '0 - var(target.uid)', 'limit': 'min(var(dr_left), var(dr_cap), var(bucket_size_%s))' % src}
        out += [sv('dr_n', 'min(var(dr_left), var(dr_cap), var(bucket_size_%s))' % src), {'action': 'select', 'pick': pick(), 'into': {'bucket': to, 'mode': 'add'}}, {'action': 'select', 'pick': pick(), 'into': {'bucket': src, 'mode': 'remove'}}, sv('dr_left', 'var(dr_left) - var(dr_n)'), sv('dr_cap', 'var(dr_cap) - var(dr_n)')]
    return out

def gather_loop(category: int, scope: Record, in_kinds: Record, limit: Any=None) -> Record:
    """One unit at a time (so each sees the others' orders), each taking the nearest node with room, preferring nodes not about to run dry."""
    where = {'unit_types': ['economic'], 'in': in_kinds}
    where.update(scope)
    lets = {'workers': {'count': {'kind': 'units', 'where': {'unit_types': ['economic'], 'in': {'order_kind': ['gather']}, 'if': 'var(item.order_target_uid) == var(target.uid)'}}}, 'own': 'var(target.uid) == var(gu.order_target_uid)'}
    lets.update(KEYS)
    loop = {'action': 'for_each', 'select': {'kind': 'units', 'where': where}, 'as': 'gu', 'do': [{'action': 'select', 'actors': {'kind': 'units', 'where': {'if': 'var(actor.uid) == var(gu.uid)'}}, 'pick': {'kind': 'gatherables', 'let': lets, 'where': {'if': ['var(target.category) == %s' % category, 'var(target.gather_capacity) - var(target.workers) + var(target.own) > 0']}, 'capacity': 'var(target.gather_capacity) - var(target.workers) + var(target.own)'}, 'match': {'score': ['0 - 10000000000000 * (var(target.amount_left) <= (var(target.workers) - var(target.own)) * var(target.gather_speed)) - 1000000000 * distance(actor, target, 1) - var(target.zone_key)']}, 'issue': [{'gather': {}}]}]}
    loop['do'][0]['pick']['where']['category'] = RESOURCES[category]
    if limit is not None:
        loop['max'] = limit
    return loop

def migrate_gather(action: Record, scope: Record) -> Record:
    if 'category' not in action:
        raise NotImplementedError('balanced gather')
    return gather_loop(CATEGORY[action['category']], scope, eligible_in(action), action.get('max'))

def _gcat(role: str) -> Record:
    """The category of the node a unit is gathering, or -1."""
    return {'max': {'kind': 'gatherables', 'where': {'if': '(var(%s.order_kind) == 1) * (var(item.uid) == var(%s.order_target_uid))' % (role, role)}, 'of': 'var(item.category)', 'default': '0 - 1'}}

def migrate_balanced_gather(action: Record) -> list[Record]:
    """Idle gatherers then working ones each take the category with the fewest workers; legacy demand weights are equal because add_q is never used."""
    mp = expr(action.get('move_penalty', 1))
    base = {'unit_types': ['economic'], 'in': eligible_in(action)}
    base.update(unit_where(action, 'gu'))
    room = ['var(target.gather_capacity) - var(target.workers) + var(target.own) > 0']
    lets = {'workers': {'count': {'kind': 'units', 'where': {'unit_types': ['economic'], 'in': {'order_kind': ['gather']}, 'if': 'var(item.order_target_uid) == var(target.uid)'}}}, 'own': 'var(target.uid) == var(gu.order_target_uid)'}
    lets.update(KEYS)

    def split(name: str, cond: Any) -> Record:
        w = dict(base)
        w['if'] = cond
        return {'action': 'select', 'pick': {'kind': 'units', 'where': w, 'let': {'gcat': _gcat('target')}}, 'into': {'set': name, 'var': 'n_' + name}}

    def avail(cat: Any, var: Any) -> Record:
        return {'action': 'select', 'pick': {'kind': 'gatherables', 'let': {k: lets[k] for k in ('workers', 'own')}, 'where': {'category': RESOURCES[cat], 'if': room}}, 'into': {'set': 'bg_t', 'var': var}}
    key = lambda c, i: '(var(c%s) * 10 + %d + (var(av%s) == 0) * 1000)' % (c, i, c)
    (kF, kW, kS) = (key('F', 0), key('W', 1), key('S', 2))
    deficit = lambda v, g: '((%s == 0) * (var(T) - var(cF)) + (%s == 1) * (var(T) - var(cW)) + (%s == 2) * (var(T) - var(cS))%s)' % (v, v, v, ' + (%s == 3) * (0 - var(cG))' % v if g else '')

    def issue(go: Any) -> Record:
        sel = {'kind': 'units', 'where': {'if': '(var(actor.uid) == var(gu.uid)) * (%s)' % go}}
        pick = {'kind': 'gatherables', 'let': lets, 'where': {'if': ['var(target.category) == var(ch)'] + room}, 'capacity': 'var(target.gather_capacity) - var(target.workers) + var(target.own)'}
        match = {'score': ['0 - 10000000000000 * (var(target.amount_left) <= (var(target.workers) - var(target.own)) * var(target.gather_speed)) - 1000000000 * distance(actor, target, 1) - var(target.zone_key)']}
        return {'action': 'select', 'actors': sel, 'pick': pick, 'match': match, 'issue': [{'gather': {}}]}
    choose = [avail(0, 'avF'), avail(1, 'avW'), avail(2, 'avS'), sv('kF', kF), sv('kW', kW), sv('kS', kS), sv('ch', '(var(kF) < var(kW)) * (var(kF) < var(kS)) * 0 + (1 - (var(kF) < var(kW)) * (var(kF) < var(kS))) * ((var(kW) < var(kS)) * 1 + (1 - (var(kW) < var(kS))) * 2)'), sv('ok', 'min(var(kF), var(kW), var(kS)) < 1000')]
    idle_body = choose + [issue('var(ok)')] + [sv('c' + c, 'var(c%s) + var(ok) * (var(ch) == %d)' % (c, i)) for (i, c) in enumerate('FWS')]
    work_body = choose + [sv('from', 'var(gu.gcat)'), sv('go', 'var(ok) * (var(ch) != var(from)) * (%s - %s > 2 + %s)' % (deficit('var(ch)', False), deficit('var(from)', True), mp)), issue('var(go)')] + [sv('c' + c, 'var(c%s) + var(go) * ((var(ch) == %d) - (var(from) == %d))' % (c, i, i)) for (i, c) in enumerate('FWS')] + [sv('cG', 'var(cG) - var(go) * (var(from) == 3)')]
    count = [sv(c, 0) for c in ('cF', 'cW', 'cS', 'cG')]
    count.append({'action': 'for_each', 'select': {'kind': 'units', 'where': {'in': {'order_kind': ['gather']}}}, 'as': 'gu', 'do': [sv('gcat', _gcat('gu')), sv('cF', 'var(cF) + (var(gcat) == 0)'), sv('cW', 'var(cW) + (var(gcat) == 1)'), sv('cS', 'var(cS) + (var(gcat) == 2)'), sv('cG', 'var(cG) + (var(gcat) == 3)')]})
    return count + [split('bg_gath', 'var(target.gcat) != 0 - 1'), split('bg_idle', 'var(target.gcat) == 0 - 1'), sv('T', '1 / 3 * (var(cF) + var(cW) + var(cS) + var(n_bg_idle))'), {'action': 'for_each', 'select': {'kind': 'units', 'where': {'in_set': 'bg_idle'}, 'rank': '0 - var(gu.uid)', 'let': {'gcat': _gcat('gu')}}, 'as': 'gu', 'do': idle_body}, {'action': 'for_each', 'select': {'kind': 'units', 'where': {'in_set': 'bg_gath'}, 'rank': '0 - var(gu.uid)', 'let': {'gcat': _gcat('gu')}}, 'as': 'gu', 'do': work_body}]

def explore_loop(action: Record, scope: Record) -> list[Record]:
    """Each idle unit in scope goes to the unexplored space (or stale fog) nearest its anchor, preferring one no other unit is heading to,
    then nearest to itself. A tie is settled by one random draw over the tied spaces, indexed in the engine's row-by-row order."""
    global HOME_USED
    anchor = action.get('anchor', 'self')
    qual = lambda v: '((var(%s.vision) == 0) + (var(%s.vision) == 1) * (var(xfog) >= 0) * (var(%s.turns_since_seen) >= var(xfog))) > 0' % (v, v, v)
    (from_x, from_y) = {'self': ('var(xux)', 'var(xuy)'), 'home': ('var(hx)', 'var(hy)'), 'center': ('0', '0')}[anchor]
    where = dict(scope)
    where['in'] = {'order_kind': ['none']}
    for key in ('unit_ids', 'unit_types'):
        if key in action:
            where[key] = action[key]
    around = lambda x, y, d: {'distance': d, 'of': {'x': x, 'y': y}}
    count_spaces = lambda w: {'count': {'kind': 'spaces', 'where': w}}
    dfrom = {'min': {'kind': 'spaces', 'where': {'if': '(var(item.x) == var(fx)) * (var(item.y) == var(fy))'}, 'of': 'var(item.distance_to_target)', 'default': 999999}}
    claimed = {'count': {'kind': 'units', 'where': {'in': {'order_kind': ['move']}, 'if': '(var(item.order_space_x) == var(target.x)) * (var(item.order_space_y) == var(target.y)) * (var(item.order_zone_x) == 0) * (var(item.order_zone_y) == 0)'}}}
    pick_into = lambda name, pick: {'action': 'select', 'pick': pick, 'into': {'set': name}}

    def ladder(count_at: Any, result: Any, base: Any) -> Record:
        out = [sv(result, 0)]
        for step in (32, 16, 8, 4, 2, 1):
            out.append(sv('xf', count_at('var(%s) + %d' % (result, step - 1))))
            out.append(sv(result, 'var(%s) + %d * ((var(xf) - %s) == 0)' % (result, step, base)))
        return out
    anchor_ladder = ladder(lambda d: count_spaces({'if': qual('item'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xmd) + ' + d)}), 'xe', 'var(xbase)')
    anchor_ladder += [sv('xf', count_spaces({'if': qual('item'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xmd) + 63')})), sv('xn', '(var(xf) - var(xbase)) == 0'), sv('xhi', '(var(xn) == 0) * (var(xmd) + var(xe)) + var(xn) * 40000'), sv('xlo', '(var(xn) == 0) * (var(xmd) + var(xe) - 1) + var(xn) * (var(xmd) - 1)')]
    unit_ladder = ladder(lambda d: count_spaces({'in_set': 'xt2', 'near': around('var(xux)', 'var(xuy)', d)}), 'xd', '0')
    unit_ladder += [sv('xf', count_spaces({'in_set': 'xt2', 'near': around('var(xux)', 'var(xuy)', '63')})), sv('xn2', 'var(xf) == 0'), sv('xhi2', '(var(xn2) == 0) * var(xd) + var(xn2) * 40000'), sv('xlo2', '(var(xn2) == 0) * (var(xd) - 1) + var(xn2) * (0 - 1)')]
    body = [sv('xuid', 'var(u.uid)'), sv('xux', 'var(u.x)'), sv('xuy', 'var(u.y)'), sv('fx', from_x), sv('fy', from_y), sv('xinner', count_spaces({'if': qual('item'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xmin) - 1')})), sv('xmd', 'var(xmin) * ((var(xcover) - var(xinner)) > 0)'), sv('xbase', count_spaces({'if': qual('item'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xmd) - 1')}))] + anchor_ladder + [pick_into('xr1', {'kind': 'spaces', 'where': {'if': qual('target'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xhi)')}}), pick_into('xr0', {'kind': 'spaces', 'where': {'if': qual('target'), 'vision_at_most': 1, 'near': around('var(fx)', 'var(fy)', 'var(xlo)')}}), pick_into('xt1', {'kind': 'spaces', 'where': {'in_set': 'xr1', 'not_in_set': 'xr0'}}), pick_into('xt1u', {'kind': 'spaces', 'let': {'claimed': claimed}, 'where': {'in_set': 'xt1', 'if': 'var(target.claimed) == 0'}}), sv('xfree', {'count': {'kind': 'spaces', 'where': {'in_set': 'xt1u'}}}), pick_into('xt2', {'kind': 'spaces', 'let': {'claimed': claimed}, 'where': {'in_set': 'xt1', 'if': '(var(target.claimed) == 0) + (var(xfree) == 0)'}})] + unit_ladder + [pick_into('xta', {'kind': 'spaces', 'where': {'in_set': 'xt2', 'near': around('var(xux)', 'var(xuy)', 'var(xhi2)')}}), pick_into('xtb', {'kind': 'spaces', 'where': {'in_set': 'xt2', 'near': around('var(xux)', 'var(xuy)', 'var(xlo2)')}}), pick_into('xt3', {'kind': 'spaces', 'where': {'in_set': 'xta', 'not_in_set': 'xtb'}}), pick_into('xs', {'kind': 'spaces', 'where': {'in_set': 'xt3'}, 'limit': 1}), sv('xsx', {'max': {'kind': 'spaces', 'where': {'in_set': 'xs'}, 'of': 'var(item.x)', 'default': 0}}), sv('xsy', {'max': {'kind': 'spaces', 'where': {'in_set': 'xs'}, 'of': 'var(item.y)', 'default': 0}}), sv('xj', count_spaces({'in_set': 'xt3', 'if': '(var(item.x) < var(xsx)) + (var(item.x) == var(xsx)) * (var(item.y) < var(xsy))'})), pick_into('xf', {'kind': 'spaces', 'where': {'in_set': 'xt3'}, 'rank': ['0 - var(target.y)', '0 - var(target.x)'], 'skip': 'var(xj)', 'limit': 1}), {'action': 'select', 'actors': {'kind': 'units', 'where': {'if': 'var(actor.uid) == var(xuid)'}}, 'pick': {'kind': 'spaces', 'where': {'in_set': 'xf'}}, 'issue': [{'move': {'to': 'zone'}}]}]
    pre = [sv('xfog', expr(action.get('include_fog', -1))), sv('xmin', 'max(0, round(%s))' % expr(action.get('min_distance', 0)))]
    if anchor == 'home':
        HOME_USED = True
    pre.append(sv('xcover', count_spaces({'if': qual('item'), 'vision_at_most': 1})))
    for_each_u = {'action': 'for_each', 'select': {'kind': 'units', 'where': where}, 'as': 'u', 'do': body}
    if 'max' in action:
        for_each_u['max'] = action['max']
    loop = {'action': 'if', 'when': {'value_at_least': {'value': 'var(xcover)', 'amount': 1}}, 'then': [for_each_u]}
    return pre + [loop]

def run_bucket(action: Record) -> list[Record]:
    name = action['bucket']
    branches = []
    for (i, task) in enumerate(BUCKET_TASKS[name]):
        if task['action'] == 'gather':
            body = [migrate_gather(task, {'bucket': name})]
        elif task['action'] == 'explore':
            body = explore_loop(task, {'bucket': name})
        branches.append({'action': 'if', 'when': {'value_equals': {'value': 'var(task_%s)' % name, 'amount': i}}, 'then': body})
    return branches
RUN_BUCKETS = set()

def scan_runs(rules: list[Record]) -> None:

    def walk(a: Record) -> Record:
        if isinstance(a, dict):
            if a.get('action') == 'run_bucket':
                RUN_BUCKETS.add(a['bucket'])
            for v in a.values():
                walk(v)
        elif isinstance(a, list):
            for v in a:
                walk(v)
    walk(rules)

def bucket_supported(name: str) -> bool:
    """Buckets whose task is never run only need their size and members; run ones need every task to be translatable."""
    if name not in BUCKET_TASKS:
        return False
    if name not in RUN_BUCKETS:
        return True
    return all((t['action'] == 'gather' and 'category' in t or t['action'] == 'explore' for t in BUCKET_TASKS[name]))

def work_loop(action: Record, source: Record, order_kind: int, verb: str, same: Callable[[str], str], outer_if: str, extra: str='1') -> Record:
    """Each source item in turn takes the nearest eligible economic unit; max counts successful orders."""
    sel = actors_for(action, '(1 - var(actor.is_military)) * (var(w_n) < %s) * (%s)' % (expr(action['max']) if action.get('max') else '1000000', extra))
    sel['where'].setdefault('unit_types', ['economic'])
    sel['where']['in'] = eligible_in(action)
    sel.setdefault('let', {})['d'] = {'min': {'kind': source['kind'], 'where': {'if': same('item')}, 'of': 'var(item.zone_distance_to_actor)', 'default': 0}}
    (sel['rank'], sel['limit']) = (['0 - var(actor.d)', '0 - var(actor.uid)'], 1)
    wh = dict(source.get('where', {}))
    wh['if'] = outer_if
    count = cnt('units', {'owner': 'own', 'unit_types': ['economic'], 'if': 'var(item.order_kind) == %d' % order_kind})
    inner = [sv('w_before', count), {'action': 'select', 'actors': sel, 'pick': {'kind': source['kind'], 'where': {'if': same('target')}, 'limit': 1}, 'issue': [{verb: {}}]}, sv('w_after', count), sv('w_n', 'var(w_n) + (var(w_after) > var(w_before))')]
    if action.get('max'):
        inner = [{'action': 'if', 'when': {'value_at_most': {'value': 'var(w_n)', 'amount': '(%s) - 1' % expr(action['max'])}}, 'then': inner}]
    return [sv('w_n', 0), {'action': 'for_each', 'as': 'w', 'select': dict(source, where=wh), 'do': inner}]

def migrate_repair(action: Record) -> list[Record]:
    ids = {'building_ids': action['building_ids']} if 'building_ids' in action else {}
    taken = cnt('units', {'owner': 'own', 'unit_types': ['economic'], 'if': '(var(item.order_kind) == 3) * (var(item.order_target_uid) == var(w.uid))'})
    same = lambda role: 'var(%s.uid) == var(w.uid)' % role
    outer = '(var(w.is_under_construction) == 0) * (var(w.health) < var(w.max_health)) * (%s == 0)' % 'var(w.taken)'
    return work_loop(action, {'kind': 'buildings', 'where': dict(owner='own', **ids), 'rank': '0 - var(w.uid)', 'let': {'taken': taken}}, 3, 'repair', same, outer)

def migrate_renew(action: Record) -> list[Record]:
    ids = {'building_ids': action['building_ids']} if 'building_ids' in action else {}
    taken = cnt('units', {'owner': 'own', 'unit_types': ['economic'], 'if': '(var(item.order_kind) == 11) * (var(item.order_target_uid) == var(w.uid))'})
    afford = 'var(w.is_renewing) + (1 - var(w.is_renewing)) * ' + ' * '.join(('(%s >= var(w.renew_cost_%s))' % (r, r) for r in RESOURCES))
    outer = 'var(w.is_gatherable) * (var(w.is_under_construction) == 0) * (var(w.resources_left) <= 0) * (var(w.taken) == 0)'
    same = lambda role: 'var(%s.uid) == var(w.uid)' % role
    return work_loop(action, {'kind': 'buildings', 'where': dict(owner='own', **ids), 'rank': '0 - var(w.uid)', 'let': {'taken': taken}}, 11, 'renew', same, outer, '(%s) > 0' % afford)

def migrate_build_foundations(action: Record) -> list[Record]:
    ids = {'building_ids': action['building_ids']} if 'building_ids' in action else {}
    same = lambda role: ' * '.join(('(var(%s.%s) == var(w.%s))' % (role, f, f) for f in ('x', 'y', 'zone_x', 'zone_y')))
    rank = ['0 - var(w.x)', '0 - var(w.y)', '0 - var(w.zone_x)', '0 - var(w.zone_y)']
    return work_loop(action, {'kind': 'foundations', 'where': dict(ids), 'rank': rank}, 2, 'build', same, 'var(w.builders) == 0')

def migrate_cancel_foundations(action: Record) -> Record:
    where = {'if': 'var(target.is_planned) * (var(target.builders) == 0)'}
    if 'building_ids' in action:
        where['building_ids'] = action['building_ids']
    return {'action': 'select', 'pick': {'kind': 'foundations', 'where': where}, 'issue': [{'cancel_foundation': {}}]}
COUNTER = re.compile('^(enemy_units_visible|enemy_units_unidentified|enemy_buildings_known|population|building_count)(?:_(.+?))?_within_(\\d+)_of_(unit|target|army)$')

def counter_spec(name: str, anchor_items: dict[str, str]) -> Record:
    """The aggregate for a legacy 'X_within_N_of_<anchor>' counter, measured with distance() to the named anchor item."""
    base = re.fullmatch(r'base_spaces_within_(\d+)_of_(unit|target|army)', name)
    if base:
        return cnt('spaces', {'in_set': 'legacy_bases', 'if': 'distance(item, %s) <= %s' % (anchor_items[base[2]], base[1])})
    m = COUNTER.match(name)
    if not m:
        raise NotImplementedError('counter ' + name)
    (base, tokens, within, anchor) = (m.group(1), m.group(2), int(m.group(3)), m.group(4))
    near = 'distance(item, %s) <= %d' % (anchor_items[anchor], within)
    if within == 0:
        near = '(var(item.x) == var(%s.x)) * (var(item.y) == var(%s.y))' % ((anchor_items[anchor],) * 2)
    if base == 'enemy_units_unidentified':
        return {'sum': {'kind': 'spaces', 'where': {'in_set': 'poor_vision', 'if': near}, 'of': 'var(item.unit_count)'}}
    (where, toks, i) = ({}, (tokens or '').split('_') if tokens else [], 0)
    (ids, types) = ([], [])
    while i < len(toks):
        t = toks[i]
        if t.isdigit():
            ids.append(int(t))
        elif t in MILITARY_TYPES:
            types.append(t)
        elif t == 'bucket':
            where['bucket'] = toks[i + 1]
            i += 1
        elif t == 'complete':
            where['state'] = 'complete'
        else:
            raise NotImplementedError('counter filter ' + t)
        i += 1
    if base in ('enemy_buildings_known', 'building_count'):
        if ids:
            where['building_ids'] = ids
        where['owner'] = 'enemy' if base == 'enemy_buildings_known' else 'own'
        kind = 'buildings'
    else:
        if ids:
            where['unit_ids'] = ids
        if types:
            where['unit_types'] = types
        where['owner'] = 'enemy' if base == 'enemy_units_visible' else 'own'
        kind = 'units'
    where['if'] = near
    return {'count': {'kind': kind, 'where': where}}

def replace_counters(text: str, anchor_items: dict[str, str], prefix: str) -> tuple[str, list[Record]]:
    """Swaps each legacy counter in an expression for a variable, returning the new text and the set_vars computing them."""
    (sets, names) = ([], {})

    def swap(m: Any) -> Record:
        name = m.group(1)
        if name not in names:
            names[name] = '%s%d' % (prefix, len(names))
            sets.append(sv(names[name], counter_spec(name, anchor_items)))
        return 'var(%s)' % names[name]
    return (re.sub('var\\((\\w+_within_\\d+_of_(?:unit|target|army))\\)', swap, text), sets)

def loc_less(a: Record, b: str) -> str:
    """Is candidate a before b in (space, zone, identity) order, given the variables holding each key."""
    k = lambda p, f: 'var(%s%s)' % (p, f)
    return '((%s < %s) + (%s == %s) * ((%s < %s) + (%s == %s) * (%s < %s)))' % (k(a, 'k1'), k(b, 'k1'), k(a, 'k1'), k(b, 'k1'), k(a, 'k2'), k(b, 'k2'), k(a, 'k2'), k(b, 'k2'), k(a, 'id'), k(b, 'id'))

def candidate_keys(prefix: str, item: str, kind: str) -> list[Record]:
    """Variables describing a candidate: its space key, zone key and identity, as compared by candidateLess."""
    ident = {'units': 'var(%s.uid)' % item, 'buildings': '1000000000000 + var(%s.uid)' % item}.get(kind, '2000000000000')
    zone = ('var(%s.zone_x)' % item, 'var(%s.zone_y)' % item) if kind != 'spaces' else ('0', '0')
    return [sv(prefix + 'k1', '(var(%s.x) + 2048) * 4096 + var(%s.y) + 2048' % (item, item)), sv(prefix + 'k2', '(%s + 8) * 16 + %s + 8' % zone), sv(prefix + 'id', ident)]
SS_KINDS = {'own_buildings': 'buildings', 'enemy_units': 'units', 'enemy_buildings': 'buildings', 'unidentified_units': 'spaces', 'known_resources': 'resources', 'closest_spaces': 'spaces', 'home': 'zones', 'coordinate': 'spaces'}

def option_origin(opt: Record, fallback: str) -> tuple[str, Record | None]:
    frm = opt.get('from')
    if not isinstance(frm, dict):
        return fallback, None
    if alias := item_ref(frm):
        return alias, None
    return 'ss_orig', {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': frm}}}

def option_candidates(opt: Record, origin: str, tg: str) -> tuple[Record, list[Record]]:
    """(loop selector, setup actions) for the candidates of one select_space option."""
    global HOME_USED
    (t, setup) = (opt['targets'], [])
    where = {}
    if t == 'own_buildings':
        where = {'owner': 'own'}
    elif t in ('enemy_units', 'enemy_buildings'):
        where = {'owner': 'enemy'}
    elif t == 'known_resources':
        where = {'category': opt['category'], 'if': 'var(%s.amount_left) > 0' % tg}
    elif t == 'unidentified_units':
        where = {'in_set': 'poor_vision', 'if': 'var(%s.unit_count) > 0' % tg}
    elif t == 'home':
        HOME_USED = True
        where = {'in_set': 'home_zone'}
    elif t == 'coordinate':
        where = {'near': {'distance': 0, 'of': opt['coordinate']}}
    elif t == 'closest_spaces':
        where = dict(opt.get('where', {}))
        far = ['distance(item, %s) > 0' % origin] if opt.get('exclude_reference') else []
        setup.append(sv('ss_dstar', {'min': {'kind': 'spaces', 'where': dict(where, **{'if': far} if far else {}), 'of': 'distance(item, %s)' % origin, 'default': '0 - 1'}}))
        where['if'] = ['distance(%s, %s) == var(ss_dstar)' % (tg, origin)] + ['distance(%s, %s) > 0' % (tg, origin)] * bool(far)
    if 'target_unit_ids' in opt:
        where['unit_ids'] = opt['target_unit_ids']
    if 'target_unit_types' in opt:
        where['unit_types'] = opt['target_unit_types']
    if 'target_building_ids' in opt:
        where['building_ids'] = opt['target_building_ids']
    return ({'kind': SS_KINDS[t], 'where': where}, setup)

def score_text(text: str, origin: str, tg: str, kind: str, anchors: dict[str, str]) -> tuple[str, list[Record]]:
    """The legacy score with target_* builtins and counters replaced by variables and distance() calls."""
    (text, sets) = replace_counters(text, anchors, 'ssc')
    text = text.replace('var(target_distance)', 'distance(%s, %s)' % (origin, tg)).replace('var(target_distance_home)', 'distance(%s, %s)' % (origin, tg))
    if kind in ('units', 'buildings'):
        text = text.replace('var(target_health)', 'var(%s.health)' % tg).replace('var(target_max_health)', 'var(%s.max_health)' % tg)
    for builtin in re.findall('var\\((target_\\w+|unit_\\w+)\\)', text):
        raise NotImplementedError('score builtin ' + builtin)
    return (text, sets)

def item_ref(coord: Any) -> str | None:
    """The loop item a coordinate is read from, as in {x: var(NAME.x), y: var(NAME.y)}, else None."""
    if not isinstance(coord, dict):
        return None
    names = {m.group(1) for m in (re.fullmatch('var\\((\\w+)\\.[xy]\\)', str(coord.get(k, ''))) for k in ('x', 'y')) if m}
    return names.pop() if len(names) == 1 else None

def migrate_select_space(a: Record) -> list[Record]:
    """The best-scoring candidate over every option, stored as x, y and score variables when one clears its minimum."""
    global HOME_USED
    options = a.get('options') or [a]
    name = a.get('var')
    (vx, vy, vs) = (a.get('var_x') or (name and name + '_x'), a.get('var_y') or (name and name + '_y'), a.get('var_score'))
    persist = bool(a.get('persist'))
    frm = a.get('from', 'home')
    O = item_ref(frm) or 'orig'
    if frm == 'home' or any((o['targets'] == 'home' for o in options)):
        HOME_USED = True
    out = [sv('ss_found', 0)]
    body = [sv('ss_found', 0), sv('ss_sc', 0), sv('ss_x', 0), sv('ss_y', 0)] + [sv('ss_b' + k, 0) for k in ('k1', 'k2', 'id')]
    for opt in options:
        alias = O if opt['targets'] == 'home' and frm == 'home' else item_ref(opt.get('coordinate')) if opt['targets'] == 'coordinate' else None
        T = alias or 'tg'
        candidate_origin, origin_select = option_origin(opt, O)
        (sel, setup) = option_candidates(opt, candidate_origin, T)
        (text, counters) = score_text(expr(opt.get('score', 0)), O, T, sel['kind'], {'army': O, 'target': T, 'unit': O})
        keys = candidate_keys('ss_c', T, sel['kind'])
        minimum = ['(var(ss_s) >= %s)' % expr(opt['min_score'])] if 'min_score' in opt else []
        better = '(var(ss_found) == 0) + (var(ss_found) == 1) * ((var(ss_s) > var(ss_sc)) + (var(ss_s) == var(ss_sc)) * %s)' % loc_less('ss_c', 'ss_b')
        take = ' * '.join(['((%s) > 0)' % better] + minimum)
        t = 'var(ss_take)'
        update = [sv('ss_take', take), sv('ss_x', '%s * var(%s.x) + (1 - %s) * var(ss_x)' % (t, T, t)), sv('ss_y', '%s * var(%s.y) + (1 - %s) * var(ss_y)' % (t, T, t))]
        update += [sv('ss_b' + k, '%s * var(ss_c%s) + (1 - %s) * var(ss_b%s)' % (t, k, t, k)) for k in ('k1', 'k2', 'id')]
        update += [sv('ss_sc', '%s * var(ss_s) + (1 - %s) * var(ss_sc)' % (t, t)), sv('ss_found', 'var(ss_found) + (1 - var(ss_found)) * %s' % t)]
        inner = counters + keys + [sv('ss_s', text)] + update
        scan = setup + (inner if alias else [{'action': 'for_each', 'select': sel, 'as': T, 'do': inner}])
        body += [{'action': 'for_each', 'select': origin_select, 'as': candidate_origin, 'do': scan}] if origin_select else scan
    final_minimum = [sv('ss_found', 'var(ss_found) * (var(ss_sc) >= (%s))' % expr(a['min_score']))] if 'min_score' in a else []
    done = final_minimum + [sv(n, 'var(ss_found) * var(%s) + (1 - var(ss_found)) * var(%s)' % (src, n), persist) for (n, src) in ((vx, 'ss_x'), (vy, 'ss_y'), (vs, 'ss_sc')) if n]
    if item_ref(frm):
        return out + body + done
    if frm == 'army':
        origin = {'kind': 'units', 'where': {'bucket': 'army'}, 'rank': ['0 - var(orig.sd)', '0 - var(orig.uid)'], 'limit': 1, 'let': {'sd': {'sum': {'kind': 'units', 'where': {'bucket': 'army'}, 'of': 'distance(item, orig)'}}}}
    elif frm == 'home':
        origin = {'kind': 'zones', 'where': {'in_set': 'home_zone'}}
    elif isinstance(frm, dict):
        origin = {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': frm}}}
    else:
        raise NotImplementedError('select_space from ' + str(frm))
    return out + [{'action': 'for_each', 'as': 'orig', 'select': origin, 'do': body + done}]

def target_lets(text: str, role: str='target') -> tuple[str, Record]:
    """Replaces target_* builtins and X_within_N_of_target counters in a score with lets on the candidate, returning text and lets."""
    (lets, names) = ({}, {})

    def put(key: Any, spec: Record) -> Record:
        lets[key] = spec
        return 'var(%s.%s)' % (role, key)
    near0 = '(var(item.x) == var(%s.x)) * (var(item.y) == var(%s.y))' % (role, role)
    builtins = {'target_distance': None, 'target_own_buildings': lambda : '(%s + %s)' % (put('ob', cnt('buildings', {'owner': 'own', 'if': near0})), put('pf', cnt('foundations', {'if': ['var(item.is_planned)', near0]}))), 'target_enemy_distance': lambda : 'max(0, %s)' % put('ed', {'min': {'kind': 'spaces', 'where': {'in_set': enemy_building_spaces()}, 'of': 'distance(item, %s)' % role, 'default': '0 - 1'}}), 'target_vision': lambda : 'var(%s.space.vision)' % role, 'target_build_speed_modifier': lambda : put('bsm', {'max': {'kind': 'spaces', 'where': {'if': near0}, 'of': 'var(item.build_speed_modifier)', 'default': 1}}), 'target_region_progress': lambda : '((1 - %s) * (var(%s.region.index) >= 0) * (var(%s.region.owned_spaces) + 1) / max(1, var(%s.region.known_spaces)))' % (put('os', cnt('spaces', {'owner': 'own', 'if': near0})), role, role, role)}

    def swap(m: Any) -> Record:
        name = m.group(1)
        if name in builtins and builtins[name]:
            return builtins[name]()
        if COUNTER.match(name):
            if name not in names:
                names[name] = put('c%d' % len(names), counter_spec(name, {'target': role, 'unit': role, 'army': role}))
            return names[name]
        return m.group(0)
    return (re.sub('var\\((\\w+)\\)', swap, text), lets)
OUTPOST = 21
BUILD_USED = []

def migrate_build(a: Record) -> list[Record]:
    global HOME_USED
    """One site (an unbuilt foundation, else the best-scoring buildable zone) for the first max builders able to build it."""
    BUILD_USED.append(1)
    (bid, builders) = (int(a['building_id']), 'max(1, %s)' % expr(a.get('max', 0)))
    if a.get('targets') != 'build_sites' or 'where' in a or 'held_by_bucket' in a or ('direction_away_from' in a):
        raise NotImplementedError('build site query')
    pick = unit_set(dict(a, unit_types=a.get('unit_types', ['economic'])), 'bu')['pick']
    pick['where']['can_build'] = bid
    pick['limit'] = builders
    out = [{'action': 'select', 'pick': pick, 'into': {'set': 'bu', 'var': 'n_bu'}}]
    anchor = a.get('from')
    afford = ' * '.join(('(gameVar(%s) >= config(building, cost_%s, %d))' % (r, r, bid) for r in RESOURCES))
    reuse = a.get('reuse_unbuilt_foundation', True)
    unbuilt = {'kind': 'foundations', 'where': {'building_ids': [bid], 'if': 'var(target.builders) == 0'}}
    out.append(sv('b_reuse', cnt('foundations', {'building_ids': [bid], 'if': 'var(item.builders) == 0'}) if reuse else 0))
    first = {'kind': 'units', 'where': {'in_set': 'bu'}, 'rank': '0 - var(fu.uid)', 'limit': 1}
    zone_rank = ['0 - var(target.x)', '0 - var(target.y)', '0 - var(target.zone_x)', '0 - var(target.zone_y)']
    (text, lets) = target_lets(drop_zero_terms(expr(a['score'])))
    origin = 'fu'
    if anchor:
        text = text.replace('var(target_distance)', 'distance(fa, target)')
    else:
        text = text.replace('var(target_distance)', 'distance(fu, target)')
    lets.update({'nf': cnt('foundations', {'if': 'distance(item, target, 1) == 0'}), 'sf': cnt('foundations', {'if': 'distance(item, target) == 0'})})
    conds = ['var(target.nf) == 0'] + (['(var(target.space.owner) < 0) * (var(target.sf) == 0)'] if bid == OUTPOST else [])
    conds += ['(%s) >= %s' % (text, expr(a['min_score']))] if 'min_score' in a else []
    if 'direction_from' in a:
        sign = {'away': -1, 'toward': 1}[a['direction']]
        (da, db, dx, dy) = ('(var(target.x) - var(hx))', '(var(target.y) - var(hy))', '(%s - var(hx))' % a['direction_from']['x'], '(%s - var(hy))' % a['direction_from']['y'])
        conds.append('(%d * (%s * %s + %s * %s + (%s + %s) * (%s + %s))) > 0' % (sign, da, dx, db, dy, da, db, dx, dy))
    site = {'kind': 'zones', 'where': {'in_set': 'buildable_zones', 'if': conds}, 'let': lets, 'rank': [text, '0 - distance(%s, target, 1)' % ('fa' if anchor else 'fu')] + zone_rank, 'limit': 1}
    reuse_pick = dict(unbuilt, rank=['0 - distance(fu, target, 1)'] + zone_rank, limit=1)
    inner = [
        {'action': 'if', 'when': {'value_at_least': {'value': 'var(b_reuse)', 'amount': 1}}, 'then': [
            {'action': 'select', 'actors': {'kind': 'units', 'where': {'in_set': 'bu'}}, 'pick': reuse_pick, 'issue': [{'build': {'building_id': bid}}]}
        ]},
        {'action': 'if', 'when': {'all': [{'value_at_most': {'value': 'var(b_reuse)', 'amount': 0}}, {'value_at_least': {'value': afford, 'amount': 1}}]}, 'then': [
            {'action': 'select', 'actors': {'kind': 'units', 'where': {'in_set': 'bu'}}, 'pick': site, 'issue': [{'build': {'building_id': bid}}]}
        ]}
    ]
    if anchor == 'home':
        HOME_USED = True
        inner = [{'action': 'for_each', 'as': 'fa', 'select': {'kind': 'zones', 'where': {'in_set': 'home_zone'}}, 'do': inner}]
    elif anchor:
        inner = [{'action': 'for_each', 'as': 'fa', 'select': {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': anchor}}}, 'do': inner}]
    return out + [{'action': 'for_each', 'as': 'fu', 'select': first, 'do': inner}]

def unit_set(action: Record, name: str, order_by_uid: bool=True) -> Record:
    """Selects the units a legacy unit action applies to into a set, in unit id order, honouring max."""
    c = Cond('target')
    where = unit_where(action, 'target')
    where['in'] = eligible_in(action)
    conds = [c.expr(action['unit_when'])] if 'unit_when' in action else []
    if where.get('if') == '0':
        conds = ['0']
    if conds:
        where['if'] = ' * '.join(('(%s)' % x for x in conds))
    pick = {'kind': 'units', 'where': where, 'rank': '0 - var(target.uid)'}
    if c.lets:
        pick['let'] = c.lets
    if action.get('max'):
        pick['limit'] = expr(action['max'])
    return {'action': 'select', 'pick': pick, 'into': {'set': name, 'var': 'n_' + name}}

def center_vars(set_name: str, out: str, uid_name: str) -> Record:
    """The unit of a set nearest all others (by summed space distance, lowest id on ties) as variables out_x, out_y, out_zx, out_zy."""
    sel = {'kind': 'units', 'where': {'in_set': set_name}, 'rank': ['0 - var(gc.sd)', '0 - var(gc.uid)'], 'limit': 1, 'let': {'sd': {'sum': {'kind': 'units', 'where': {'in_set': set_name}, 'of': 'distance(item, gc)'}}}}
    return {'action': 'for_each', 'select': sel, 'as': 'gc', 'do': [sv(uid_name, 'var(gc.uid)'), sv(out + '_x', 'var(gc.x)'), sv(out + '_y', 'var(gc.y)'), sv(out + '_zx', 'var(gc.zone_x)'), sv(out + '_zy', 'var(gc.zone_y)')]}

def target_near(target: str, zone: bool) -> Record:
    """The zero-distance filter matching the space or exact zone holding the target variables."""
    of = {'x': 'var(%s_x)' % target, 'y': 'var(%s_y)' % target}
    if zone:
        of.update({'zone_x': 'var(%s_zx)' % target, 'zone_y': 'var(%s_zy)' % target})
    return {'distance': 0, 'of': of, **({'unit': 'zone'} if zone else {})}

def target_pick(target: str, zone: bool) -> Record:
    """The pick of a target saved once into a set, instead of rescanning every space or zone per unit."""
    return {'kind': 'zones' if zone else 'spaces', 'where': {'in_set': target + ('_zone' if zone else '_space')}}

def target_store(target: str, zone: bool) -> Record:
    name = target + ('_zone' if zone else '_space')
    return {'action': 'select', 'pick': {'kind': 'zones' if zone else 'spaces', 'where': {'near': target_near(target, zone)}}, 'into': {'set': name, 'var': 'n_' + name}}

def move_issue(set_name: str, guard: str, target: str, zone: bool, hoisted: bool = False) -> Record:
    """Moves the set's units passing guard to the target variables: a bare space (zone centre) or an exact zone."""
    actors = {'kind': 'units', 'where': {'in_set': set_name, 'if': guard}}
    pick = target_pick(target, zone) if hoisted else {'kind': 'zones' if zone else 'spaces', 'where': {'near': target_near(target, zone)}}
    return {'action': 'select', 'actors': actors, 'pick': pick, 'issue': [{'move': {'to': 'zone'}}]}

def together_each(guard: str, near_selects: Any, uid_name: str) -> Record:
    """Walks the group in unit id order: units over 2 spaces from the group centre regroup there, the others run near_selects(guard)."""
    gc = {'kind': 'units', 'where': {'in_set': 'mv_a', 'if': '(var(gc.uid) == var(%s))' % uid_name}, 'limit': 1}
    each = {'kind': 'units', 'where': {'in_set': 'mv_a'}, 'rank': '0 - var(au.uid)'}
    me = '(var(actor.uid) == var(au.uid))'
    body = [move_issue_actor('%s * (distance(au, gc) > 2) * (%s)' % (me, guard), 'mv_ctr', True)] + near_selects(me + ' * (distance(au, gc) <= 2)')
    return {'action': 'for_each', 'as': 'gc', 'select': gc, 'do': [{'action': 'for_each', 'as': 'au', 'select': each, 'do': body}]}

def move_issue_actor(guard: str, target: str, zone: bool) -> Record:
    actors = {'kind': 'units', 'where': {'in_set': 'mv_a', 'if': guard}}
    return {'action': 'select', 'actors': actors, 'pick': target_pick(target, zone), 'issue': [{'move': {'to': 'zone'}}]}

def migrate_retreat(a: Record) -> list[Record]:
    global HOME_USED
    HOME_USED = True
    lets = dict(KEYS)
    lets['home_distance'] = {'min': {'kind': 'spaces', 'where': {'in_set': 'home_space'}, 'of': 'distance(item, target)', 'default': 0}}
    pick = {'kind': 'spaces', 'where': {'in_set': 'legacy_bases'}, 'let': lets, 'rank': ['0 - distance(au, target)', '0 - var(target.home_distance)', '0 - var(target.space_key)'], 'limit': 1}
    actors = {'kind': 'units', 'where': {'if': '(var(actor.uid) == var(au.uid)) * (var(mv_at_base) == 0)'}}
    choose = {'action': 'select', 'actors': actors, 'pick': pick, 'issue': [{'move': {'to': 'zone'}}]}
    fallback = {'action': 'select', 'actors': {'kind': 'units', 'where': {'if': '(var(actor.uid) == var(au.uid)) * (var(mv_base_count) == 0)'}}, 'pick': {'kind': 'zones', 'where': {'in_set': 'home_zone'}}, 'issue': [{'move': {}}]}
    body = [sv('mv_at_base', cnt('spaces', {'in_set': 'legacy_bases', 'if': 'distance(item, au) == 0'})), choose, fallback]
    if a.get('together'):
        center = {'kind': 'units', 'where': {'in_set': 'mv_a'}, 'rank': ['0 - var(ac.total_distance)', '0 - var(ac.uid)'], 'limit': 1,
                  'let': {'total_distance': {'sum': {'kind': 'units', 'where': {'in_set': 'mv_a'}, 'of': 'distance(item, ac)'}}}}
        pick['rank'][0] = '0 - distance(ac, target)'
        regroup = {'action': 'select', 'actors': {'kind': 'units', 'where': {'if': 'var(actor.uid) == var(au.uid)'}}, 'pick': {'kind': 'units', 'where': {'if': 'var(target.uid) == var(ac.uid)'}}, 'issue': [{'move': {'to': 'zone'}}]}
        branch = {'action': 'if', 'when': {'value_at_least': {'value': 'distance(au, ac)', 'amount': 3}}, 'then': [regroup], 'else': [choose, fallback]}
        members = {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'mv_a'}}, 'do': [branch]}
        body = [sv('mv_at_base', cnt('spaces', {'in_set': 'legacy_bases', 'if': 'distance(item, ac) == 0'})), {'action': 'if', 'when': {'value_equals': {'value': 'var(mv_at_base)', 'amount': 0}}, 'then': [members]}]
        return [unit_set(a, 'mv_a'), sv('mv_base_count', cnt('spaces', {'in_set': 'legacy_bases'})), {'action': 'for_each', 'as': 'ac', 'select': center, 'do': body}]
    return [unit_set(a, 'mv_a'), sv('mv_base_count', cnt('spaces', {'in_set': 'legacy_bases'})), {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'mv_a'}}, 'do': body}]

def closest_target(a: Record, tag: str, origin: str='orig', actor: str='gc') -> tuple[list[Record], list[Record]]:
    """The best candidate (closest spaces to the from-anchor, or known resources) by score, as variables tag_x, tag_y, tag_zx, tag_zy and mv_found."""
    resources = a['targets'] != 'closest_spaces'
    where = dict(a.get('where', {}))
    if a.get('held_by'):
        where['near'] = {'distance': 0, 'of': {'kind': 'units', 'where': {'bucket': a['held_by']}}}
    far = ['distance(item, %s) > 0' % origin] if a.get('exclude_reference') else []
    if a['targets'] == 'known_resources':
        (tg_where, kind) = ({'category': a['category'], 'if': 'var(tg.amount_left) > 0'}, 'resources')
    elif resources:
        (tg_where, kind) = ({'owner': 'enemy'}, {'enemy_units': 'units', 'enemy_buildings': 'buildings'}[a['targets']])
        for (key, to) in (('target_unit_ids', 'unit_ids'), ('target_unit_types', 'unit_types'), ('target_building_ids', 'building_ids')):
            if key in a:
                tg_where[to] = a[key]
    else:
        (tg_where, kind) = (dict(where, **{'if': ['distance(tg, %s) == var(mv_dstar)' % origin] + ['distance(tg, %s) > 0' % origin] * bool(far)}), 'spaces')
    (score, sets) = score_text(expr(a.get('score', 0)), actor, 'tg', kind, {'army': origin, 'target': 'tg', 'unit': actor})
    face = a.get('face')
    keys = candidate_keys('mv_c', 'tg', kind)
    if resources:
        zone = [sv('mv_czx', 'var(tg.zone_x)'), sv('mv_czy', 'var(tg.zone_y)'), sv('mv_ck2', '(var(mv_czx) + 8) * 16 + var(mv_czy) + 8')]
    else:
        zone = [sv('mv_cfdx', '%s - var(tg.x)' % face['x']), sv('mv_cfdy', '%s - var(tg.y)' % face['y']), sv('mv_cfok', 'var(mv_fexists) * (max(var(mv_cfdx), 0 - var(mv_cfdx)) <= 1) * (max(var(mv_cfdy), 0 - var(mv_cfdy)) <= 1) * (var(mv_cfdx) != var(mv_cfdy))')] if face else [sv('mv_cfok', 0), sv('mv_cfdx', 0), sv('mv_cfdy', 0)]
        zone += [sv('mv_czx', 'var(mv_cfok) * var(mv_cfdx)'), sv('mv_czy', 'var(mv_cfok) * var(mv_cfdy)'), sv('mv_ck2', '(var(mv_czx) + 8) * 16 + var(mv_czy) + 8')]
    minimum = ' * (var(mv_s) >= %s)' % expr(a['min_score']) if 'min_score' in a else ''
    take = '((var(mv_found) == 0) + (var(mv_found) == 1) * ((var(mv_s) > var(mv_sc)) + (var(mv_s) == var(mv_sc)) * %s) > 0)%s' % (loc_less('mv_c', 'mv_b'), minimum)
    t = 'var(mv_take)'
    update = [sv('mv_take', take)] + [sv('mv_b' + k, '%s * var(mv_c%s) + (1 - %s) * var(mv_b%s)' % (t, k, t, k)) for k in ('k1', 'k2', 'id')]
    if kind in ('units', 'buildings'):
        update.append(sv(tag + '_uid', '%s * var(tg.uid) + (1 - %s) * var(%s_uid)' % (t, t, tag)))
    update += [sv(tag + '_x', '%s * var(tg.x) + (1 - %s) * var(%s_x)' % (t, t, tag)), sv(tag + '_y', '%s * var(tg.y) + (1 - %s) * var(%s_y)' % (t, t, tag)), sv(tag + '_zx', '%s * var(mv_czx) + (1 - %s) * var(%s_zx)' % (t, t, tag)), sv(tag + '_zy', '%s * var(mv_czy) + (1 - %s) * var(%s_zy)' % (t, t, tag)), sv('mv_sc', '%s * var(mv_s) + (1 - %s) * var(mv_sc)' % (t, t)), sv('mv_found', 'var(mv_found) + (1 - var(mv_found)) * %s' % t)]
    setup = [sv('mv_found', 0), sv('mv_sc', 0), sv('mv_bk1', 0), sv('mv_bk2', 0), sv('mv_bid', 0), sv(tag + '_uid', 0), sv(tag + '_x', 0), sv(tag + '_y', 0), sv(tag + '_zx', 0), sv(tag + '_zy', 0)]
    if not resources:
        setup.append(sv('mv_dstar', {'min': {'kind': 'spaces', 'where': dict(where, **{'if': far} if far else {}), 'of': 'distance(item, %s)' % origin, 'default': '0 - 1'}}))
    if face:
        setup.append(sv('mv_fexists', cnt('spaces', {'near': {'distance': 0, 'of': {'x': face['x'], 'y': face['y']}}})))
    body = [{'action': 'for_each', 'select': {'kind': kind, 'where': tg_where}, 'as': 'tg', 'do': sets + keys + zone + [sv('mv_s', score)] + update}]
    return (setup, body)

def migrate_move(a: Record) -> list[Record]:
    global HOME_USED, CENTER_SEQ
    (tg, together) = (a['targets'], a.get('together'))
    if a.get('spread'):
        return migrate_spread_move(a)
    if tg == 'retreat' and a.get('to') != 'home':
        return migrate_retreat(a)
    if tg == 'retreat':
        if a.get('to') != 'home':
            raise NotImplementedError('move targets retreat')
        tg = 'home'
    single = str(a.get('max')) == '1' and (not together)
    out = [unit_set(a, 'mv_a')]
    uid_var = None
    if together:
        CENTER_SEQ += 1
        uid_var = 'mv_ctr_uid_%d' % CENTER_SEQ
        out.append(center_vars('mv_a', 'mv_ctr', uid_var))
    if tg == 'coordinate':
        out += [sv('mv_t_x', a['coordinate']['x']), sv('mv_t_y', a['coordinate']['y']), sv('mv_t_zx', 0), sv('mv_t_zy', 0), sv('mv_t_found', 1)]
    elif tg == 'home':
        HOME_USED = True
        out += [sv('mv_t_x', 'var(hx)'), sv('mv_t_y', 'var(hy)'), sv('mv_t_zx', 'var(hzx)'), sv('mv_t_zy', 'var(hzy)'), sv('mv_t_found', 1)]
    elif tg in ('closest_spaces', 'known_resources'):
        if not together and (not single) and ('target_distance' in str(a.get('score', ''))):
            raise NotImplementedError('per-unit move score')
        if 'from' not in a and not together:
            setup, body = closest_target(a, 'mv_t', 'au', 'au')
            guard = '(var(mv_found) > 0) * (var(actor.uid) == var(au.uid))'
            issue = issue_actors('mv_a', guard, tg)
            each = {'kind': 'units', 'where': {'in_set': 'mv_a'}, 'rank': '0 - var(au.uid)'}
            return out + [{'action': 'for_each', 'as': 'au', 'select': each, 'do': setup + body + issue}]
        if 'from' not in a:
            setup, body = closest_target(a, 'mv_t', 'orig', 'orig')
            center = {'kind': 'units', 'where': {'in_set': 'mv_a', 'if': '(var(orig.uid) == var(%s))' % uid_var}, 'limit': 1}
            scan = [{'action': 'for_each', 'as': 'orig', 'select': center, 'do': setup + body}]
            out += [sv('mv_t_found', 0)] + scan + [sv('mv_t_found', 'var(mv_found)')]
        else:
            setup, body = closest_target(dict(a, held_by=a.get('held_by_bucket')), 'mv_t')
            frm = a['from']
            orig_where = {'bucket': 'army'} if frm == 'army' else {'bucket': frm}
            origin = {'kind': 'units', 'where': orig_where, 'rank': ['0 - var(orig.sd)', '0 - var(orig.uid)'], 'limit': 1, 'let': {'sd': {'sum': {'kind': 'units', 'where': orig_where, 'of': 'distance(item, orig)'}}}} if frm == 'army' else {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': frm}}}
            gc_where = {'in_set': 'mv_a', 'if': '(var(gc.uid) == var(%s))' % uid_var} if uid_var else {'in_set': 'mv_a'}
            gc_sel = {'kind': 'units', 'where': gc_where, 'limit': 1}
            if not uid_var:
                gc_sel.update({'rank': ['0 - var(gc.sd)', '0 - var(gc.uid)'], 'let': {'sd': {'sum': {'kind': 'units', 'where': {'in_set': 'mv_a'}, 'of': 'distance(item, gc)'}}}})
            gc = {'action': 'for_each', 'as': 'gc', 'select': gc_sel, 'do': body}
            scan = setup + [gc] if tg == 'known_resources' else [{'action': 'for_each', 'as': 'orig', 'select': origin, 'do': setup + [gc]}]
            out += [sv('mv_t_found', 0)] + scan + [sv('mv_t_found', 'var(mv_found)')]
    else:
        raise NotImplementedError('move targets ' + tg)
    guard = '(var(mv_t_found) > 0)'
    flat = '(var(mv_t_zx) == 0) * (var(mv_t_zy) == 0)'
    if tg == 'home':
        stores = [target_store('mv_t', True)]
    elif tg == 'coordinate':
        stores = [target_store('mv_t', False)]
    else:
        stores = [target_store('mv_t', False), target_store('mv_t', True)]
    if together:
        stores.append(target_store('mv_ctr', True))

    def issue(set_name: str, g: Any) -> Record:
        if tg == 'home':
            return [move_issue(set_name, g, 'mv_t', True, True)]
        if tg == 'coordinate':
            return [move_issue(set_name, g, 'mv_t', False, True)]
        return [move_issue(set_name, '(%s) * %s' % (g, flat), 'mv_t', False, True), move_issue(set_name, '(%s) * (1 - %s)' % (g, flat), 'mv_t', True, True)]
    if not together:
        return out + stores + issue('mv_a', guard)
    return out + stores + [together_each(guard, lambda g: issue_actors('mv_a', '(%s) * %s' % (g, guard), tg, True), uid_var)]

def issue_actors(set_name: str, g: Any, tg: str, hoisted: bool = False) -> Record:
    """The move selects of migrate_move for a guard that already names the acting unit."""
    flat = '(var(mv_t_zx) == 0) * (var(mv_t_zy) == 0)'
    if tg == 'home':
        return [move_issue(set_name, g, 'mv_t', True, hoisted)]
    if tg == 'coordinate':
        return [move_issue(set_name, g, 'mv_t', False, hoisted)]
    return [move_issue(set_name, '(%s) * %s' % (g, flat), 'mv_t', False, hoisted), move_issue(set_name, '(%s) * (1 - %s)' % (g, flat), 'mv_t', True, hoisted)]

def migrate_for_each(action: Record) -> Record:
    """The source form becomes the select form; within/from turn into a near filter on a coordinate."""
    if action['source'] not in ('spaces', 'units', 'buildings', 'resources', 'zones'):
        raise NotImplementedError('for_each source ' + action['source'])
    where = dict(action.get('where', {}))
    if 'within' in where or 'from' in where:
        (anchor, within) = (where.pop('from', None), where.pop('within'))
        names = {m.group(1) for m in (re.fullmatch('var\\((\\w+)\\.[xy]\\)', str(anchor.get(k, ''))) for k in ('x', 'y')) if m} if isinstance(anchor, dict) else set()
        if len(names) != 1:
            raise NotImplementedError('for_each from ' + str(anchor))
        near = 'distance(%s, %s) <= %s' % (action['as'], names.pop(), within)
        where['if'] = [near] + ([where['if']] if isinstance(where.get('if'), str) else list(where.get('if', [])))
    out = {'action': 'for_each', 'select': {'kind': action['source'], 'where': where}, 'as': action['as']}
    if 'max' in action:
        out['max'] = action['max']
    out['do'] = [b for child in action.get('do', []) for b in migrate_action(child)]
    return out
SIMPLE = {'set_unit_behavior': migrate_behavior, 'unit_stop': migrate_stop, 'garrison': migrate_garrison, 'ungarrison': migrate_ungarrison}

def combat_expression(value: Any, unit: str, actors: Record, targets: Record) -> str:
    global HOME_USED
    text = expr(value)
    if 'var(target_distance_home)' in text:
        HOME_USED = True
        targets.setdefault('let', {})['combat_home_distance'] = {'min': {'kind': 'spaces', 'where': {'in_set': 'home_space'}, 'of': 'distance(item, target)', 'default': 0}}
        text = text.replace('var(target_distance_home)', 'var(target.combat_home_distance)')
    names = set(re.findall(r'var\((\w+)\)', text))
    for name in sorted(names):
        if COUNTER.match(name) or name.startswith('base_spaces_within_'):
            owner, role = (actors, unit) if name.endswith('_of_unit') and unit == 'actor' else (targets, 'target')
            key = 'combat_' + name
            membership = presence_membership(name, targets, text)
            owner.setdefault('let', {})[key] = membership if membership is not None else counter_spec(name, {'unit': unit, 'target': 'target', 'army': 'ac'})
            text = text.replace('var(%s)' % name, 'var(%s.%s)' % (role, key))
    replacements = {'unit_id': 'var(%s.unit_id)' % unit, 'unit_health': 'var(%s.health)' % unit,
                    'unit_stamina': 'var(%s.stamina)' % unit, 'unit_distance': 'distance(%s, target, 1)' % unit,
                    'target_distance': 'distance(%s, target)' % unit, 'target_id': 'var(target.uid)',
                    'target_health': 'var(target.health)', 'target_max_health': 'var(target.max_health)'}
    for name, replacement in replacements.items():
        text = text.replace('var(%s)' % name, replacement)
    return text

def attack_targets(a: Record) -> Record:
    kind = {'enemy_units': 'units', 'enemy_buildings': 'buildings', 'closest_spaces': 'spaces'}[a['targets']]
    where = dict(a.get('where', {})) if kind == 'spaces' else {'owner': 'enemy'}
    for old, new in (('target_unit_ids', 'unit_ids'), ('target_unit_types', 'unit_types'), ('target_building_ids', 'building_ids')):
        if old in a:
            where[new] = a[old]
    pick = {'kind': kind, 'where': where}
    if kind == 'spaces':
        search = dict(where)
        if a.get('exclude_reference'):
            search['if'] = 'distance(item, ac) > 0'
        pick['let'] = {'nearest': {'min': {'kind': 'spaces', 'where': search, 'of': 'distance(item, ac)', 'default': -1}}}
        where['if'] = 'distance(target, ac) == var(target.nearest)'
    return pick

def pair_terms(text: str) -> str:
    return text.replace('var(target.zone_distance_to_actor)', 'distance(actor, target, 1)').replace('var(target.distance_to_actor)', 'distance(actor, target)').replace('var(target_assigned)', 'var(target.assigned)')

def lexicographic_ties(kind: str) -> list[str]:
    keys = ['0 - var(target.x)', '0 - var(target.y)']
    if kind != 'spaces':
        keys += ['0 - var(target.zone_x)', '0 - var(target.zone_y)']
    if kind in ('units', 'buildings'):
        keys.append('0 - var(target.uid)')
    return keys

def attack_spread(a: Record, actors: Record, pick: Record, chain: bool=False) -> Record:
    spread = dict(a['spread'])
    grouped = spread.pop('group_by', None)
    if grouped == 'space' and pick['kind'] != 'spaces':
        source = dict(pick['where'])
        source['if'] = 'distance(item, target) == 0'
        pick.clear()
        pick.update({'kind': 'spaces', 'where': {'if': 'var(target.enemy_here) > 0'}, 'let': {'enemy_here': cnt('units', source)}})
    base = combat_expression(a.get('score', 0), 'actor', actors, pick)
    score = base + ' + (' + combat_expression(spread.get('unit_score', 0), 'actor', actors, pick) + ')'
    if grouped and 'min_score' in a:
        pick['where']['if'] = [pick['where']['if'], '(%s) >= (%s)' % (base, expr(a['min_score']))]
    elif 'min_score' in a and 'actor.' not in score and 'distance(actor,' not in score:
        existing = pick['where'].get('if', [])
        pick['where']['if'] = ([existing] if isinstance(existing, str) else existing) + ['(%s) >= (%s)' % (score, expr(a['min_score']))]
    if pick['kind'] in ('units', 'buildings'):
        tie_breaker = ' - (((((var(target.x) + 32) * 64 + (var(target.y) + 32)) * 8 + (var(target.zone_x) + 2)) * 8 + (var(target.zone_y) + 2)) * 100000 + var(target.uid)) / 10000000000000'
    elif pick['kind'] == 'zones':
        tie_breaker = ' - ((((var(target.x) + 32) * 64 + (var(target.y) + 32)) * 8 + (var(target.zone_x) + 2)) * 8 + (var(target.zone_y) + 2)) / 100000000'
    else:
        tie_breaker = ' - ((var(target.x) + 32) * 64 + (var(target.y) + 32)) / 1000000000'
    actors['rank'] = combat_expression(spread.get('unit_order', '0 - var(unit_uid)'), 'actor', actors, pick).replace('var(unit_uid)', 'var(actor.uid)')
    if chain:
        actors['rank'] = [actors['rank'], '0 - var(actor.uid)']
    scores = [pair_terms(score)] + lexicographic_ties(pick['kind']) if chain else [pair_terms(score + tie_breaker)]
    match = {'weight': combat_expression(spread.get('weight', 1), 'actor', actors, pick), 'score': scores}
    if 'capacity' in spread:
        capacity = combat_expression(spread['capacity'], 'actor', actors, pick)
        if pick['kind'] != 'spaces':
            outside = {'count': {'kind': 'orders', 'where': {'subject_not_in_set': 'attack_units', 'in': {'subject_kind': ['unit'], 'order_kind': ['attack_unit', 'attack_building', 'auto_attack_unit', 'auto_attack_building']}, 'if': 'var(item.target_uid) == var(target.uid)'}, 'distinct': 'subject_uid'}}
            if chain:
                pick['initial_assigned'] = outside
            else:
                pick.setdefault('let', {})['outside_attackers'] = outside
                capacity = '(%s) - var(target.outside_attackers)' % capacity
        if 'min_score' in a:
            match['where'] = {'if': '(%s) >= (%s)' % (pair_terms(score), expr(a['min_score']))}
        pick['capacity'] = capacity
    return {'action': 'select', 'actors': actors, 'pick': pick, 'match': match, 'issue': [{'attack': {'to': a['order']} if a.get('order', 'unit') != 'unit' else {}}]}

def attack_single(a: Record, actors: Record, pick: Record, unit: str) -> Record:
    score = combat_expression(a.get('score', 0), unit, actors, pick)
    pick['rank'] = [score, '0 - var(target.x)', '0 - var(target.y)']
    if pick['kind'] != 'spaces':
        pick['rank'] += ['0 - var(target.zone_x)', '0 - var(target.zone_y)', '0 - var(target.uid)']
    pick['limit'] = 1
    if 'min_score' in a:
        existing = pick['where'].get('if', [])
        pick['where']['if'] = ([existing] if isinstance(existing, str) else existing) + ['(%s) >= (%s)' % (score, expr(a['min_score']))]
    action = {'action': 'select', 'actors': actors, 'pick': pick, 'issue': [{'attack': {'to': a['order']} if a.get('order', 'unit') != 'unit' else {}}]}
    if a.get('resume'):
        resumed = dict(action, mode='prepend')
        return {'action': 'if', 'when': {'value_at_least': {'value': '(var(%s.order_kind) == 1) + (var(%s.order_kind) == 2) + (var(%s.order_kind) == 3) + (var(%s.order_kind) == 11)' % ((unit,) * 4), 'amount': 1}}, 'then': [resumed], 'else': [action]}
    return action

def queued_attack_loads(a: Record, actors: Record, pick: Record, target_kind: str, target_match: str) -> tuple[list[Record], list[str]]:
    lets = pick.setdefault('let', {})
    setup, loads = [], []
    for unit_id in COMBAT_UNIT_IDS:
        name = 'attack_units_%d' % unit_id
        setup.append({'action': 'select', 'pick': {'kind': 'units', 'where': {'in_set': 'attack_units', 'unit_ids': [unit_id]}}, 'into': {'set': name}})
        key = 'queued_%d' % unit_id
        lets[key] = cnt('orders', {'subject_in_set': name, 'in': {'order_kind': [target_kind]}, 'if': target_match})
        weight = combat_expression(a['spread'].get('weight', 1), 'actor', actors, pick).replace('var(actor.unit_id)', str(unit_id))
        loads.append('var(target.%s) * (%s)' % (key, weight))
    return setup, loads

def queued_attack_followup(a: Record, action: Record, actors: Record) -> tuple[list[Record], Record]:
    followup = copy.deepcopy(action)
    queued = followup['do'][0] if followup.get('action') == 'for_each' else followup
    queued['mode'] = 'append'
    where = queued.setdefault('match', {}).setdefault('where', {})
    existing = where.get('if')
    order = a.get('order', 'unit')
    target_kind = 'attack_space' if order == 'space' else 'attack_zone' if order == 'zone' else ('attack_building' if queued['pick']['kind'] == 'buildings' else 'attack_unit')
    target_match = '(var(item.target_space_x) == var(target.x)) * (var(item.target_space_y) == var(target.y))' if order == 'space' else ('(var(item.target_space_x) == var(target.x)) * (var(item.target_space_y) == var(target.y)) * (var(item.target_zone_x) == var(target.zone_x)) * (var(item.target_zone_y) == var(target.zone_y))' if order == 'zone' else 'var(item.target_uid) == var(target.uid)')
    setup, loads = queued_attack_loads(a, actors, queued['pick'], target_kind, target_match)
    queued['pick']['capacity'] = '(%s) - (%s)' % (queued['pick']['capacity'], ' + '.join(loads))
    condition = '1 - var(target.already_targeted)'
    if order != 'unit':
        queued['pick'].setdefault('let', {})['queued_actor'] = cnt('orders', {'in': {'order_kind': [target_kind]}, 'if': '(var(item.subject_uid) == var(au.uid)) * (%s)' % target_match})
        condition = '1 - var(target.queued_actor)'
        actor_if = queued['actors']['where'].get('if')
        queued['actors']['where']['if'] = '(%s) * (var(actor.uid) == var(au.uid))' % actor_if if actor_if else 'var(actor.uid) == var(au.uid)'
        followup = {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': '0 - var(au.uid)'}, 'do': [followup]}
    where['if'] = [existing, condition] if existing else condition
    return setup, followup

def round_robin_fallback(action: Record) -> list[Record]:
    if action.get('action') != 'select':
        raise NotImplementedError('round_robin overflow outside a plain select')
    fallback = copy.deepcopy(action)
    pick, actors, match = fallback['pick'], fallback['actors'], fallback['match']
    pick.pop('capacity')
    pick['let']['primary_load'] = {'sum': {'kind': 'assignments', 'where': {'relation': PRIMARY_RELATION, 'if': 'var(item.target_uid) == var(target.uid)'}, 'of': 'var(item.weight)'}}
    pick['initial_assigned'] = 'var(target.outside_attackers) + var(target.primary_load)'
    actors['let'] = {'primary_hits': cnt('assignments', {'relation': PRIMARY_RELATION, 'if': 'var(item.actor_uid) == var(actor.uid)'})}
    unassigned = '1 - var(actor.primary_hits)'
    existing = actors['where'].get('if')
    actors['where']['if'] = [existing, unassigned] if existing else unassigned
    match['score'] = ['0 - var(target.assigned)'] + match['score']
    fallback['into'] = {'assign': 'overflow_fallback'}
    return [fallback]

def migrate_attack(a: Record) -> list[Record]:
    global HOME_USED
    if 'target_distance_home' in json.dumps(a):
        HOME_USED = True
    return migrate_attack_orders(a)

def is_assignment_chain(a: Record) -> bool:
    spread = a.get('spread', {})
    return (isinstance(spread.get('queue'), int) and spread['queue'] > 0 and spread.get('overflow') == 'none' and not spread.get('group_by')
            and a.get('order') == 'space' and a.get('targets') == 'enemy_units' and not a.get('together'))

def assignment_chain(a: Record, actors: Record, pick: Record) -> list[Record]:
    init = attack_spread(a, actors, pick, True)
    init['into'] = {'assign': 'attack_spread'}
    return [init] + [{'action': 'select', 'match': {'carry': 'attack_spread'}} for _ in range(a['spread']['queue'])]

def migrate_attack_orders(a: Record) -> list[Record]:
    filtered = dict(a)
    if not a.get('unit_ids') and not a.get('unit_types'):
        filtered['unit_types'] = ['military']
    selected = unit_set(filtered, 'attack_units')
    if a.get('resume'):
        selected['pick']['where']['in']['order_kind'] = [kind for kind in selected['pick']['where']['in']['order_kind'] if kind not in ORDER_KIND_NAMES[4:8] + ORDER_KIND_NAMES[12:14]]
    actors = {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': '0 - var(actor.uid)'}
    pick = attack_targets(a)
    if a.get('spread'):
        if a['spread'].get('group_by') == 'space' and a['targets'] == 'enemy_units':
            return [selected] + grouped_defense(a)
        if is_assignment_chain(a):
            return [selected] + assignment_chain(a, actors, pick)
        action = attack_spread(a, actors, pick)
        if a['targets'] == 'closest_spaces':
            action = {'action': 'for_each', 'as': 'ac', 'select': {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': a['from']}}}, 'do': [action]}
        queue = a['spread'].get('queue', 0)
        if not isinstance(queue, int):
            raise NotImplementedError('spread queue expression')
        actions = [action]
        if a['spread'].get('overflow', 'round_robin') == 'round_robin':
            actions = [dict(action, into={'assign': PRIMARY_RELATION})] + round_robin_fallback(action)
        setup = []
        if queue:
            setup, followup = queued_attack_followup(a, action, actors)
            if isinstance(action, dict) and action.get('action') == 'select' and a['spread'].get('overflow', 'round_robin') == 'round_robin':
                actions.extend({'action': 'select', 'match': {'carry': PRIMARY_RELATION}} for _ in range(queue))
            elif isinstance(followup, dict) and followup.get('action') == 'select':
                followup['mode'] = 'append'
                followup['into'] = {'assign': 'queued_attack'}
                actions.append(followup)
                actions.extend({'action': 'select', 'match': {'carry': 'queued_attack'}} for _ in range(queue - 1))
            else:
                actions.extend(copy.deepcopy(followup) for _ in range(queue))
        return [selected] + setup + actions
    if a.get('together'):
        return [selected] + attack_together(a, actors, pick)
    actors['where']['if'] = 'var(actor.uid) == var(au.uid)'
    return [selected, {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': '0 - var(au.uid)'}, 'do': [attack_single(a, actors, pick, 'au')]}]

def attack_together(a: Record, actors: Record, pick: Record) -> list[Record]:
    center = {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': ['0 - var(ac.total_distance)', '0 - var(ac.uid)'], 'limit': 1,
              'let': {'total_distance': {'sum': {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'of': 'distance(item, ac)'}}}}
    single = attack_single(a, actors, pick, 'ac')
    target_check = copy.deepcopy(single['pick'])
    target_check['limit'] = 1
    find_tgt = {'action': 'select', 'pick': target_check, 'into': {'set': 'atk_tgt', 'var': 'has_atk_tgt'}}
    pick_tgt = {'kind': single['pick']['kind'], 'where': {'in_set': 'atk_tgt'}}
    actor_au = {'kind': 'units', 'where': {'if': 'var(actor.uid) == var(au.uid)'}}
    attack = {'action': 'select', 'actors': actor_au, 'pick': pick_tgt, 'issue': [{'attack': {'to': a['order']} if a.get('order', 'unit') != 'unit' else {}}]}
    if a.get('resume'):
        resumed = dict(attack, mode='prepend')
        attack = {'action': 'if', 'when': {'value_at_least': {'value': '(var(au.order_kind) == 1) + (var(au.order_kind) == 2) + (var(au.order_kind) == 3) + (var(au.order_kind) == 11)', 'amount': 1}}, 'then': [resumed], 'else': [attack]}
    regroup = {'action': 'select', 'actors': actor_au, 'pick': {'kind': 'units', 'where': {'if': 'var(target.uid) == var(ac.uid)'}}, 'issue': [{'move': {'to': 'zone'}}]}
    branch = {'action': 'if', 'when': {'value_at_least': {'value': 'distance(au, ac)', 'amount': 3}}, 'then': [regroup], 'else': [attack]}
    members = {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': '0 - var(au.uid)'}, 'do': [branch]}
    guarded_members = {'action': 'if', 'when': {'value_at_least': {'value': 'var(has_atk_tgt)', 'amount': 1}}, 'then': [members]}
    return [{'action': 'for_each', 'as': 'ac', 'select': center, 'do': [find_tgt, guarded_members]}]

def migrate_spread_move(a: Record) -> list[Record]:
    if a['targets'] != 'closest_spaces' or not isinstance(a.get('from'), dict):
        raise NotImplementedError('spread move source')
    actors = {'kind': 'units', 'where': {'in_set': 'move_units'}, 'rank': '0 - var(actor.uid)'}
    pick = attack_targets(a)
    action = attack_spread(a, actors, pick)
    face = a.get('face')
    dx, dy = ('(%s - var(target.x))' % expr(face['x']), '(%s - var(target.y))' % expr(face['y'])) if face else ('0', '0')
    faced = '(max(%s, 0 - %s) <= 1) * (max(%s, 0 - %s) <= 1) * (%s != %s)' % (dx, dx, dy, dy, dx, dy) if face else '0'
    if face:
        pick.setdefault('let', {})['face_exists'] = cnt('spaces', {'near': {'distance': 0, 'of': {'x': expr(face['x']), 'y': expr(face['y'])}}})
        faced = 'var(target.face_exists) * ' + faced
    pick['kind'] = 'zones'
    pick['where'] = {'at': a.get('where', {}), 'if': ['distance(target, ac) == var(spread_nearest)', 'var(target.zone_x) == (%s) * (%s)' % (faced, dx), 'var(target.zone_y) == (%s) * (%s)' % (faced, dy)]}
    nearest = pick.get('let', {}).pop('nearest', None)
    action['issue'] = [{'move': {'to': 'zone'}}]
    anchor = {'kind': 'spaces', 'where': {'near': {'distance': 0, 'of': a['from']}}}
    body = ([sv('spread_nearest', nearest)] if nearest is not None else []) + [action]
    return [unit_set(a, 'move_units'), {'action': 'for_each', 'as': 'ac', 'select': anchor, 'do': body}]

def grouped_defense_load(a: Record, pick: Record) -> list[Record]:
    with open(os.path.join(os.path.dirname(__file__), '..', '..', '..', 'units.json')) as source:
        unit_ids = [entry['unit_id'] for entry in json.load(source)]
    setup, loads = [], []
    for unit_id in unit_ids:
        name = 'defense_type_%d' % unit_id
        setup.append({'action': 'select', 'pick': {'kind': 'units', 'where': {'in_set': 'attack_units', 'unit_ids': [unit_id]}}, 'into': {'set': name}})
        where = {'subject_in_set': name, 'subject_not_in_set': 'defense_pending', 'in': {'order_kind': ['attack_space']},
                 'if': '(var(item.target_space_x) == var(target.x)) * (var(item.target_space_y) == var(target.y))'}
        pick['let'][name] = {'count': {'kind': 'orders', 'where': where, 'distinct': 'subject_uid'}}
        weight = expr(a['spread'].get('weight', 1)).replace('var(unit_id)', str(unit_id))
        loads.append('var(target.%s) * (%s)' % (name, weight))
    pick['let']['defense_load'] = ' + '.join(loads)
    return setup

def grouped_defense(a: Record) -> list[Record]:
    source = attack_targets(a)['where']
    source['if'] = '(var(item.x) == var(target.x)) * (var(item.y) == var(target.y))'
    ready = {'action': 'select', 'pick': {'kind': 'spaces', 'where': {'in_set': 'legacy_bases', 'if': 'var(target.enemy_here) > 0'}, 'let': {'enemy_here': cnt('units', source)}}, 'into': {'set': 'defense_spaces'}}
    pick = {'kind': 'spaces', 'where': {'in_set': 'defense_spaces'}, 'let': {}}
    setup = grouped_defense_load(a, pick)
    raw_score = '(%s) + (%s)' % (expr(a.get('score', 0)), expr(a['spread'].get('unit_score', 0)))
    raw_score = raw_score.replace('var(target_assigned)', 'var(target.defense_load)')
    score = combat_expression(raw_score, 'au', {}, pick)
    capacity = combat_expression(a['spread']['capacity'], 'au', {}, pick)
    opened = '(var(target.defense_load) < (%s))' % capacity
    pick['where']['if'] = '(%s) >= (%s)' % (score, expr(a.get('min_score', 0)))
    pick['rank'] = [opened, '(%s) * (%s) - (1 - %s) * var(target.defense_load)' % (opened, score, opened), '(1 - %s) * (%s)' % (opened, score), '0 - var(target.x)', '0 - var(target.y)']
    pick['limit'] = 1
    issue = {'action': 'select', 'actors': {'kind': 'units', 'where': {'if': 'var(actor.uid) == var(au.uid)'}}, 'pick': pick, 'into': {'set': 'defense_target', 'var': 'defense_assigned'}, 'issue': [{'attack': {'to': 'space'}}]}
    release = {'action': 'select', 'pick': {'kind': 'units', 'where': {'in_set': 'defense_pending', 'if': '(var(target.uid) != var(au.uid)) + (var(defense_assigned) == 0)'}}, 'into': {'set': 'defense_pending'}}
    loop = [
        {'action': 'select', 'pick': {'kind': 'units', 'where': {'in_set': 'attack_units'}}, 'into': {'set': 'defense_pending'}},
        {'action': 'for_each', 'as': 'au', 'select': {'kind': 'units', 'where': {'in_set': 'attack_units'}, 'rank': '0 - var(au.uid)'}, 'do': [issue, release]}
    ]
    guarded = {'action': 'if', 'when': {'value_at_least': {'value': cnt('spaces', {'in_set': 'legacy_bases', 'if': 'var(item.enemy_units) > 0'}), 'amount': 1}}, 'then': [ready] + setup + loop}
    return [guarded]

def split_queued_rules(model: Record) -> None:
    """Gives each select of an always-rule holding queued order loads its own rule, so the per-rule step cap cannot truncate one."""
    rules = []
    for rule in model['rules']:
        then = rule['then']
        if rule['when'] != {'always': {}} or 'queued_1' not in json.dumps(then):
            rules.append(rule)
            continue
        chunk = []
        for action in then:
            if chunk and action.get('action') == 'select':
                rules.append({**rule, 'then': chunk})
                chunk = []
            chunk.append(action)
        if chunk:
            rules.append({**rule, 'then': chunk})
    model['rules'] = rules

def validate_actions(value: Any) -> None:
    if isinstance(value, dict):
        kind = value.get('action')
        if kind is not None and kind not in ('select', 'if', 'for_each', 'set_var'):
            raise ValueError('untranslated action: ' + str(kind))
        for child in value.values():
            validate_actions(child)
    elif isinstance(value, list):
        for child in value:
            validate_actions(child)

def migrate_children(action: Record) -> Record:
    out = dict(action)
    for key in ('then', 'else', 'do'):
        if key in out:
            out[key] = [b for child in out[key] for b in migrate_action(child)]
    return [out]

def migrate_action(action: Record) -> list[Record]:
    return _migrate_action(action)

def _migrate_action(action: Record) -> list[Record]:
    kind = action.get('action')
    if kind == 'attack':
        return migrate_attack(action)
    if kind in SIMPLE and kind in KINDS:
        return [SIMPLE[kind](action)]
    if kind == 'gather' and 'category' not in action and (kind in KINDS):
        return migrate_balanced_gather(action)
    if kind == 'renew' and kind in KINDS:
        return migrate_renew(action)
    if kind == 'repair' and kind in KINDS:
        return migrate_repair(action)
    if kind == 'build_foundations' and kind in KINDS:
        return migrate_build_foundations(action)
    if kind == 'cancel_foundations' and kind in KINDS:
        return [migrate_cancel_foundations(action)]
    if kind == 'create' and kind in KINDS:
        return [migrate_create(action)]
    if kind == 'research' and kind in KINDS:
        return [migrate_research(action)]
    if kind == 'hire' and kind in KINDS:
        return migrate_hire(action)
    if 'buckets' in KINDS and kind in ('set_bucket', 'fill_bucket', 'run_bucket') and bucket_supported(action['bucket']):
        return {'set_bucket': migrate_set_bucket, 'fill_bucket': lambda a: [migrate_fill_bucket(a)], 'run_bucket': run_bucket}[kind](action)
    if kind == 'drain_bucket' and kind in KINDS:
        return migrate_drain_bucket(action)
    if 'buckets' in KINDS and kind == 'empty_bucket':
        return [migrate_empty_bucket(action)]
    if kind == 'build' and 'build' in KINDS:
        return migrate_build(action)
    if kind == 'move' and 'move' in KINDS:
        return migrate_move(action)
    if kind == 'select_space' and 'select_space' in KINDS:
        return migrate_select_space(action)
    if kind == 'for_each' and 'source' in action and ('for_each' in KINDS):
        return [migrate_for_each(action)]
    out = dict(action)
    for key in ('then', 'else', 'do'):
        if key in out:
            out[key] = [b for child in out[key] for b in migrate_action(child)]
    return [out]

def buildable_prelude() -> Record:
    """A first rule saving this turn's buildable spaces, then the empty zones in them, for every build rule to reuse."""
    anchor = {'kind': 'spaces', 'where': {'if': 'inSet(owned_spaces, anchor) + (var(n_land) == 0) > 0'}}
    spaces = {'kind': 'spaces', 'where': {'near': {'distance': 1, 'of': anchor, 'through': {}}, 'if': 'inSet(owned_spaces, target) + (var(target.owner) < 0) > 0'}}
    zones = {'kind': 'zones', 'where': {'at': {'in_set': 'buildable_spaces'}, 'if': '(1 - var(target.has_resource)) * (var(target.building_player) < 0)'}}
    owned = {'action': 'select', 'pick': {'kind': 'spaces', 'where': {'owner': 'own'}}, 'into': {'set': 'owned_spaces', 'var': 'n_land'}}
    return {'when': {'always': {}}, 'then': [owned, {'action': 'select', 'pick': spaces, 'into': {'set': 'buildable_spaces'}}, {'action': 'select', 'pick': zones, 'into': {'set': 'buildable_zones'}}]}

def poor_vision_prelude() -> Record:
    """A first rule saving this turn's poor-vision spaces, so unidentified-unit reads never scan the whole map."""
    pick = {'kind': 'spaces', 'where': {'vision': 2}}
    actions = [{'action': 'select', 'pick': pick, 'into': {'set': 'poor_vision', 'var': 'n_poor_vision'}}]
    if ENEMY_BUILDING_SPACES_USED:
        actions += enemy_building_space_store()
    return {'when': {'always': {}}, 'then': actions}

def base_prelude() -> Record:
    has_vc = {'building_count_at_least': {'building_ids': [1], 'amount': 1}}
    with_vc = {
        'kind': 'spaces',
        'where': {
            'if': 'var(target.own_buildings) > 0',
            'near': {
                'distance': 100,
                'of': {'kind': 'buildings', 'where': {'building_ids': [1], 'owner': 'own'}},
                'through': {'if': 'var(place.own_buildings) > 0'},
            },
        },
    }
    without_vc = {
        'kind': 'spaces',
        'where': {'if': 'var(target.own_buildings) > 0'},
    }
    action = {
        'action': 'if',
        'when': has_vc,
        'then': [{'action': 'select', 'pick': with_vc, 'into': {'set': 'legacy_bases'}}],
        'else': [{'action': 'select', 'pick': without_vc, 'into': {'set': 'legacy_bases'}}],
    }
    return {'when': {'always': {}}, 'then': [action]}

FAMILIES = sorted(('bucket_size', 'economic_producers', 'building_count', 'foundation_count', 'foundation_space_x', 'foundation_space_y', 'idle_units', 'population', 'enemy_units_visible', 'enemy_units_unidentified', 'enemy_buildings_known', 'resource_remaining', 'resource_available', 'enemy_distance'), key=len, reverse=True)

def parse_legacy(name: str) -> Record | None:
    """The filters encoded in a legacy builtin var name, or None when the name is not one."""
    base = next((f for f in FAMILIES if name == f or name.startswith(f + '_')), None)
    if base is None or name == 'population_max':
        return None
    spec = dict(base=base, ids=[], types=[], bucket=None, state=None, builders=False, within=None, anchor=None, category=None)
    toks = name[len(base) + 1:].split('_') if len(name) > len(base) else []
    if base == 'bucket_size':
        return dict(spec, bucket=name[len(base) + 1:])
    return parse_tokens(spec, toks)

def parse_tokens(spec: Record, toks: list[str]) -> Record:
    i = 0
    while i < len(toks):
        (t, i) = (toks[i], i + 1)
        if t.isdigit():
            spec['ids'].append(int(t))
        elif t in MILITARY_TYPES:
            spec['types'].append(t)
        elif t in RESOURCES:
            spec['category'] = t
        elif t in ('complete', 'damaged', 'depleted'):
            spec['state'] = t
        elif t in ('bucket', 'within'):
            (spec['bucket' if t == 'bucket' else 'within'], i) = (toks[i] if t == 'bucket' else int(toks[i]), i + 1)
            if t == 'within' and toks[i:i + 1] == ['of']:
                (spec['anchor'], i) = ('_'.join(toks[i + 1:]), len(toks))
        elif (t, toks[i:i + 1]) in (('under', ['construction']), ('without', ['builders'])):
            (spec['state'], spec['builders'], i) = ('under_construction' if t == 'under' else spec['state'], t == 'without', i + 1)
        else:
            raise NotImplementedError('legacy filter ' + t)
    return spec

def legacy_where(spec: Record, owner: str, near: Record | None) -> Record:
    where = {'owner': owner}
    for (key, field) in (('ids', 'unit_ids'), ('types', 'unit_types'), ('bucket', 'bucket'), ('near', 'near')):
        value = near if key == 'near' else spec[key]
        if value:
            where[field] = value
    return where

def legacy_aggregate(spec: Record, near: Record | None) -> Record:
    """The aggregate equal to one legacy counter, with near already resolved to a coordinate filter."""
    base = spec['base']
    if base == 'enemy_units_unidentified':
        where = {'in_set': 'poor_vision'}
        if near:
            where['near'] = near
        return {'sum': {'kind': 'spaces', 'where': where, 'of': 'var(item.unit_count)'}}
    if base in ('enemy_buildings_known', 'building_count'):
        where = legacy_where(dict(spec, ids=[]), 'enemy' if base == 'enemy_buildings_known' else 'own', near)
        where.update({'building_ids': spec['ids']} if spec['ids'] else {}, **({'state': spec['state']} if spec['state'] else {}))
        return cnt('buildings', where)
    where = legacy_where(spec, 'enemy' if base == 'enemy_units_visible' else 'own', near)
    if base == 'idle_units':
        where['in'] = {'order_kind': ['none']}
    return cnt('units', where)

def foundation_aggregate(spec: Record) -> Record:
    where = {'building_ids': spec['ids']} if spec['ids'] else {}
    if spec['builders']:
        where['if'] = '1 - var(item.builders)'
    if spec['base'] == 'foundation_count':
        return cnt('foundations', where)
    axis = spec['base'][-1]
    lowest = ['0 - var(item.x)', '0 - var(item.y)', '0 - var(item.zone_x)', '0 - var(item.zone_y)']
    return {'sum': {'kind': 'foundations', 'where': where, 'rank': lowest, 'limit': 1, 'of': 'var(item.%s)' % axis}}

def aggregate_for(spec: Record, near: Record | None) -> Record:
    if spec['base'].startswith('foundation_'):
        return foundation_aggregate(spec)
    if spec['base'] == 'resource_remaining':
        where = {'category': spec['category'], **({'near': near} if near else {})}
        return {'sum': {'kind': 'resources', 'where': where, 'of': 'var(item.amount_left)'}}
    if spec['base'] == 'bucket_size':
        return cnt('units', {'owner': 'own', 'bucket': spec['bucket']})
    if spec['base'] == 'economic_producers':
        return cnt('buildings', {'owner': 'own', 'can_create': 1})
    return legacy_aggregate(spec, near)

def bucket_center(name: str, bucket: str) -> tuple[str, list[Record]]:
    """Variables name_x, name_y and name_ok holding the bucket's group centre, as the legacy 'from: bucket' anchor computed it."""
    members = {'kind': 'units', 'where': {'owner': 'own', 'bucket': bucket}}
    pick = dict(members, rank=['0 - var(gc.sd)', '0 - var(gc.uid)'], limit=1, let={'sd': {'sum': dict(members, of='distance(item, gc)')}})
    body = [sv(name + '_x', 'var(gc.x)'), sv(name + '_y', 'var(gc.y)'), sv(name + '_ok', 1)]
    return (name, [sv(name + '_ok', 0), {'action': 'for_each', 'select': pick, 'as': 'gc', 'do': body}])

NAME_KEYS = ('bucket', 'category', 'set', 'in_set', 'not_in_set', 'subject_in_set', 'subject_not_in_set', 'as', 'name', 'assign', 'carry', 'relation')
TECH_TERM = re.compile(r'(?<!\w)\((?=[^()]*researched\()(?:[\d.\s+*\-]|researched\(\d+\))+\)')

class Modernizer:
    """Replaces each legacy counter read with a set_var of an equal aggregate placed right before the read."""

    def __init__(self, defined: set[str]) -> None:
        (self.defined, self.count, self.uses_home, self.memo, self.centers) = (defined, 0, False, {}, {})
        self.tech_terms: dict[str, str] = {}

    def hoist_tech_terms(self, value: str) -> str:
        def swap(m: Any) -> str:
            name = self.tech_terms.setdefault(m.group(0), 'tech_term_%d' % len(self.tech_terms))
            return 'var(%s)' % name
        return TECH_TERM.sub(swap, value)

    def tmp(self) -> str:
        self.count += 1
        return 'lg%d' % self.count

    def anchor(self, within: int | None, anchor: Any, setups: list[Record]) -> tuple[Record | None, str]:
        """(near filter, guard factor) for 'within N of anchor'; the guard is 0 when a bucket anchor has no members."""
        if within is None:
            return (None, '1')
        if isinstance(anchor, dict):
            return ({'distance': within, 'of': anchor}, '1')
        if anchor is None or anchor == 'home':
            self.uses_home = True
            return ({'distance': within, 'of': {'x': 'var(hx)', 'y': 'var(hy)'}}, '1')
        name = self.centers.get(anchor)
        if name is None:
            (name, steps) = bucket_center(self.tmp(), anchor)
            self.centers[anchor] = name
            setups.extend(steps)
        return ({'distance': within, 'of': {'x': 'var(%s_x)' % name, 'y': 'var(%s_y)' % name}}, 'var(%s_ok)' % name)

    def distance_setup(self, name: str) -> list[Record]:
        global ENEMY_DISTANCE_USED
        ENEMY_DISTANCE_USED = True
        self.uses_home = True
        return [sv(name, 'var(enemy_dist_home)')]

    def availability_setup(self, name: str, category: str) -> list[Record]:
        """name = how many gatherables of the category have a free gather slot."""
        taking = {'owner': 'own', 'unit_types': ['economic'], 'in': {'order_kind': ['gather']}, 'if': 'var(item.order_target_uid) == var(rg.uid)'}
        body = [sv(name + '_n', cnt('units', taking)), sv(name, 'var(%s) + (var(%s_n) < var(rg.gather_capacity))' % (name, name))]
        return [sv(name, 0), {'action': 'for_each', 'select': {'kind': 'gatherables', 'where': {'category': category}}, 'as': 'rg', 'do': body}]

    def reference(self, spec: Record, setups: list[Record]) -> str:
        key = json.dumps(spec, sort_keys=True)
        if key not in self.memo:
            (near, guard) = self.anchor(spec['within'], spec['anchor'], setups)
            name = self.tmp()
            if spec['base'] == 'resource_available':
                setups.extend(self.availability_setup(name, spec['category']))
                self.memo[key] = 'min(1, var(%s))' % name
                return self.memo[key]
            setups.extend(self.distance_setup(name) if spec['base'] == 'enemy_distance' else [sv(name, aggregate_for(spec, near))])
            self.memo[key] = 'var(%s)' % name if guard == '1' else '(%s * var(%s))' % (guard, name)
        return self.memo[key]

    def text(self, value: str, setups: list[Record]) -> str:
        def swap(m: Any) -> str:
            spec = None if m.group(1) in self.defined else parse_legacy(m.group(1))
            return m.group(0) if spec is None else self.reference(spec, setups)
        value = re.sub('var\\((\\w+)\\)', swap, value)
        value = re.sub(r'var\((unit|building|tech)_cost_(\w+)_(\d+)\)', r'config(\1, cost_\2, \3)', value)
        value = re.sub(r'var\(tech_researched_(\d+)\)', r'researched(\1)', value)
        value = re.sub(r'var\(population_max\)', r'gameVar(population_max)', value)
        value = re.sub(r'(?<![.\w])(?<!gameVar\()(population_limit|population_max|population|food|wood|stone|gold|turn|map_size|num_players)(?![.\w])', r'gameVar(\1)', value)
        if re.search(r'(?<![.\w])land(?![.\w])', value):
            name = self.tmp()
            setups.append(sv(name, cnt('spaces', {'owner': 'own'})))
            value = re.sub(r'(?<![.\w])land(?![.\w])', 'var(%s)' % name, value)
        return self.hoist_tech_terms(value)

    def count_condition(self, key: str, body: Record, setups: list[Record]) -> Record:
        (base, op) = re.fullmatch('(\\w+?)_(at_least|at_most|equals)', key).groups()
        spec = dict(base=base, ids=list(body.get('unit_ids') or body.get('building_ids') or []), types=list(body.get('unit_types', [])), bucket=body.get('in_bucket'), state=body.get('state'), builders=False, within=body.get('within'), anchor=body.get('from'), category=None)
        (near, guard) = self.anchor(spec['within'], spec['anchor'], setups)
        if base == 'count_spaces':
            total = cnt('spaces', {**body.get('where', {}), **({'near': near} if near else {})})
        elif base == 'enemies_found':
            total = {'count': {'kind': 'zones', 'where': {'owner': 'enemy'}, 'distinct': 'building_player'}}
        else:
            total = aggregate_for(spec, near)
        name = self.tmp()
        setups.append(sv(name, total))
        amount = self.text(body['amount'], setups) if isinstance(body['amount'], str) else body['amount']
        return {'value_' + op: {'value': 'var(%s)' % name if guard == '1' else '(%s * var(%s))' % (guard, name), 'amount': amount}}

    def cond(self, c: Record, setups: list[Record]) -> Record:
        ((key, body),) = c.items()
        if key in ('all', 'any'):
            return {key: [self.cond(x, setups) for x in body]}
        if key == 'not':
            return {key: self.cond(body, setups)}
        if key == 'always':
            return c
        if key.startswith('value_'):
            return {key: dict(body, value=self.value(body['value'], setups))}
        return self.count_condition(key, body, setups)

    def value(self, o: Any, setups: list[Record]) -> Any:
        if isinstance(o, str):
            return self.text(o, setups)
        if isinstance(o, list):
            return [self.value(x, setups) for x in o]
        if not isinstance(o, dict):
            return o
        nested = o.get('action') in ('if', 'for_each')
        return {k: v if k in NAME_KEYS else self.actions(v) if nested and k in ('then', 'else', 'do') else self.cond(v, setups) if nested and k == 'when' and o['action'] == 'if' else self.value(v, setups) for (k, v) in o.items()}

    def actions(self, items: list[Record]) -> list[Record]:
        (out, saved) = ([], self.memo)
        for a in items:
            (self.memo, setups) = ({}, [])
            out.extend(self.worker_count(a) if a.get('action') == 'set_var' and 'unit_when' in a else (lambda new: setups + [new])(self.value(a, setups)))
        self.memo = saved
        return out

    def worker_count(self, a: Record) -> list[Record]:
        """set_var of available_gatherers under unit_when: count the eligible gatherers the condition keeps."""
        if a['value'] != 'var(available_gatherers)':
            raise NotImplementedError('unit_when on ' + a['value'])
        (c, name) = (Cond('target'), self.tmp())
        where = {'unit_types': ['economic'], 'in': {'order_kind': ['gather', 'none']}, 'if': '(1 - var(target.is_garrisoned)) * (%s)' % c.expr(a['unit_when'])}
        pick = {'kind': 'units', 'where': where, 'let': c.lets}
        return [{'action': 'select', 'pick': pick, 'into': {'set': name + '_set', 'var': name}}, sv(a['name'], 'var(%s)' % name, a.get('persist', False))]

def defined_names(value: Any, found: set[str] | None=None) -> set[str]:
    found = set() if found is None else found
    if isinstance(value, dict):
        if value.get('action') == 'set_var':
            found.add(value['name'])
        if isinstance(value.get('into'), dict) and 'var' in value['into']:
            found.add(value['into']['var'])
    for child in (value.values() if isinstance(value, dict) else value if isinstance(value, list) else []):
        defined_names(child, found)
    return found

def modernize(model: Record) -> None:
    m = Modernizer(defined_names(model))
    rules = []
    for r in model['rules']:
        (m.memo, m.centers, setups) = ({}, {}, [])
        when = m.cond(r['when'], setups)
        then = m.actions(r['then'])
        rules += ([{'when': {'always': {}}, 'then': setups}] if setups else []) + [{**r, 'when': when, 'then': then}]
    tech_terms = [{'when': {'always': {}}, 'then': [sv(name, term) for term, name in m.tech_terms.items()]}] if m.tech_terms else []
    home_actions = home_vars() + [home_space(), home_zone_store()] if (m.uses_home or HOME_USED) else []
    if ENEMY_DISTANCE_USED:
        home_actions += enemy_distance_setup()
    mil_actions = mil_presence_sets() if (MIL_UNITS_USED or MIL_BUILDINGS_USED or MIL_SPACES_USED) else []
    model['rules'] = tech_terms + ([{'when': {'always': {}}, 'then': home_actions}] if home_actions else []) + ([{'when': {'always': {}}, 'then': mil_actions}] if mil_actions else []) + rules

def dedupe_repeats(model: Record) -> None:
    """Reuses an earlier stored value for an identical pure aggregate when only untouched set_vars lie between."""
    reads = lambda value: set(re.findall(r'var\(([A-Za-z_][A-Za-z0-9_]*)', json.dumps(value, sort_keys=True)))

    def walk(actions: list[Record]) -> None:
        for a in actions:
            if a.get('action') == 'if':
                walk(a.get('then', []))
                walk(a.get('else', []))
            elif a.get('action') == 'for_each':
                walk(a.get('do', []))
        seen: dict[str, tuple[int, str, set[str]]] = {}
        for i, a in enumerate(actions):
            if a.get('action') != 'set_var':
                seen = {}
                continue
            if not isinstance(a.get('value'), dict):
                continue
            text = json.dumps(a['value'], sort_keys=True)
            if 'random(' in text:
                continue
            if text in seen:
                (j, first, used) = seen[text]
                between = actions[j + 1:i]
                if all(b.get('action') == 'set_var' and b.get('name') != first and b.get('name') not in used for b in between):
                    a['value'] = 'var(%s)' % first
                    continue
            seen[text] = (i, a.get('name'), reads(a['value']))

    for rule in model['rules']:
        walk(rule.get('then', []))

def dedupe_setups(model: Record) -> None:
    """Rewrites a repeated always-rule setup whose inputs are unchanged as copies of the earlier setup's variables."""
    ident = re.compile(r'\blg\d+[A-Za-z0-9_]*')
    read_names = lambda text: set(re.findall(r'var\(([A-Za-z_][A-Za-z0-9_]*)', text))
    read_sets = lambda text: set(re.findall(r'"(?:in_set|not_in_set|bucket)": "([^"]+)"', text))
    risky = ('order_', 'stance', 'attack_back', 'interrupt', 'is_garrisoned', 'garrison', 'assignment', 'foundation', '"orders"', 'behavior', 'random(', 'gameVar(')

    def canon(o: Any, mapping: dict[str, str]) -> Any:
        if isinstance(o, str):
            return ident.sub(lambda m: mapping.setdefault(m.group(0), 'V%d' % len(mapping)), o)
        if isinstance(o, dict):
            return {k: canon(v, mapping) for k, v in o.items()}
        if isinstance(o, list):
            return [canon(v, mapping) for v in o]
        return o

    def pure(then: list[Record]) -> bool:
        return all(a.get('action') == 'set_var' or (a.get('action') == 'for_each' and all(b.get('action') == 'set_var' for b in a.get('do', []))) for a in then)

    def writes(rule: Record) -> tuple[set[str], set[str], set[str], bool]:
        vs: set[str] = set()
        st: set[str] = set()
        bk: set[str] = set()
        issue = [False]

        def walk(o: Any) -> None:
            if isinstance(o, dict):
                it = o.get('into')
                if isinstance(it, dict):
                    if 'set' in it:
                        st.add(it['set'])
                    if 'bucket' in it:
                        bk.add(it['bucket'])
                if o.get('action') == 'set_var' and isinstance(o.get('name'), str):
                    vs.add(o['name'])
                if 'issue' in o:
                    issue[0] = True
                for v in o.values():
                    walk(v)
            elif isinstance(o, list):
                for v in o:
                    walk(v)
        walk(rule.get('then', []))
        return vs, st, bk, issue[0]

    def correspondence(a: list[Record], b: list[Record], mp: dict[str, str]) -> None:
        for x, y in zip(a, b):
            if x.get('action') == 'set_var' and y.get('action') == 'set_var':
                mp[y.get('name')] = x.get('name')
            elif x.get('action') == 'for_each' and y.get('action') == 'for_each':
                correspondence(x.get('do', []), y.get('do', []), mp)

    rules = model['rules']
    original = [copy.deepcopy(r.get('then', [])) for r in rules]
    groups: dict[str, list[int]] = {}
    for i, rule in enumerate(rules):
        if rule.get('when') == {'always': {}} and pure(original[i]):
            groups.setdefault(json.dumps(canon(original[i], {}), sort_keys=True), []).append(i)
    for idxs in groups.values():
        for a, b in zip(idxs, idxs[1:]):
            if b - a <= 1:
                continue
            a_text, b_text = json.dumps(original[a]), json.dumps(original[b])
            av, aset, bv, bset = read_names(a_text), read_sets(a_text), read_names(b_text), read_sets(b_text)
            ok, issued = True, False
            for i in range(a + 1, b):
                vs, st, bk, issue = writes(rules[i])
                issued = issued or issue
                if vs & (av | bv) or st & (aset | bset) or bk & (aset | bset):
                    ok = False
                    break
            if not ok or (issued and any(t in a_text or t in b_text for t in risky)):
                continue
            mp: dict[str, str] = {}
            correspondence(original[a], original[b], mp)
            rules[b]['then'] = [sv(bn, 'var(%s)' % an) for bn, an in mp.items()]

def split_long_rules(model: Record) -> None:
    """Splits long rules into chunks of actions under the same condition, giving each chunk its own selector budget."""
    out = []
    for rule in model['rules']:
        then = rule.get('then', [])
        when_text = json.dumps(rule.get('when', {}), sort_keys=True)
        live = any(k in when_text for k in ('"count"', '"sum"', '"min"', '"max"', 'distance(', 'gameVar(', 'config(', 'random(', 'in_set', 'bucket', 'near'))
        if not live and len(then) >= 12:
            written = set()
            def collect(x: Any) -> None:
                if isinstance(x, dict):
                    if x.get('action') == 'set_var' and isinstance(x.get('name'), str):
                        written.add(x['name'])
                    for v in x.values():
                        collect(v)
                elif isinstance(x, list):
                    for v in x:
                        collect(v)
            collect(then)
            live = bool(set(re.findall(r'var\(([A-Za-z_][A-Za-z0-9_]*)', when_text)) & written)
        if live or len(then) < 12:
            out.append(rule)
        else:
            for i in range(0, len(then), 6):
                out.append({**rule, 'then': then[i:i + 6]})
    model['rules'] = out

def migrate_model(src: str, dst: str) -> None:
    global HOME_USED, CENTER_SEQ, ENEMY_DISTANCE_USED, MIL_UNITS_USED, MIL_BUILDINGS_USED, MIL_SPACES_USED, ZERO_VARS, ENEMY_BUILDING_SPACES_USED
    HOME_USED = False
    CENTER_SEQ = 0
    ENEMY_DISTANCE_USED = False
    MIL_UNITS_USED = False
    MIL_BUILDINGS_USED = False
    MIL_SPACES_USED = False
    ENEMY_BUILDING_SPACES_USED = False
    BUCKET_TASKS.clear()
    RUN_BUCKETS.clear()
    BUILD_USED.clear()
    model = json.load(open(src))
    ZERO_VARS = literal_zero_vars(model)
    scan_buckets(model['rules'])
    scan_runs(model['rules'])
    model['rules'] = [{**r, 'then': [b for a in r['then'] for b in migrate_action(a)]} for r in model['rules']]
    if BUILD_USED:
        model['rules'].insert(0, buildable_prelude())
    model['rules'].insert(0, base_prelude())
    model['rules'].insert(0, poor_vision_prelude())
    modernize(model)
    split_queued_rules(model)
    dedupe_repeats(model)
    dedupe_setups(model)
    split_long_rules(model)
    validate_actions(model)
    with open(dst, 'w') as output:
        output.write(json.dumps(model, indent=2) + '\n')

def main() -> None:
    here = os.path.dirname(os.path.abspath(__file__))
    out = os.path.join(here, '..', '..', 'v0.6.json')
    source = os.path.join(here, 'v0.6_source.json')
    for arg in sys.argv[1:]:
        if arg.startswith('out='):
            out = arg[4:]
        else:
            raise ValueError('unsupported argument: ' + arg)
    migrate_model(source, out)
if __name__ == '__main__':
    main()
