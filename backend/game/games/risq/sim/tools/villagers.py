"""What one AI's villagers do each turn in a debug replay (food/wood/stone/gold/build/move/idle), its bank, army and
building production. usage: python3 tools/villagers.py outputs/<name>/debug.log ai-<seat>"""
import json, re, sys, collections
log, who = sys.argv[1], sys.argv[2]
CAT = {0: 'food', 1: 'wood', 2: 'stone', 3: 'gold'}
KIND = {0: 'move', 1: 'gather', 2: 'build', 3: 'repair', 4: 'renew'}
turn = 0
print("turn | vils: food wood stone gold build move idle other | res f/w/s/g | military | producing")
for line in open(log):
    m = re.search(r'ai (ai-\d+): new turn data (.*)$', line)
    if not m or m.group(1) != who:
        continue
    d = json.loads(m.group(2)); turn += 1
    act = collections.Counter()
    for u in d['units']:
        if u['UnitID'] != 1: continue
        o = u.get('CurrentOrder')
        if not o: act['idle'] += 1
        elif o.get('TargetResource'): act[CAT.get(o['TargetResource']['Category'], '?')] += 1
        elif o.get('Kind') == 2: act['build'] += 1
        elif o.get('Kind') == 0: act['move'] += 1
        else: act['other'] += 1
    mil = collections.Counter(u['UnitID'] for u in d['units'] if u['UnitID'] != 1)
    r = d['resources']
    prod = collections.Counter()
    for b in d['buildings']:
        for o in b.get('ActiveOrders') or []:
            prod[f"b{b['BuildingID']}:{o.get('Kind')}"] += 1
    if turn % 3 == 0 or turn < 6:
        print(f"{turn:4d} | {sum(act.values()):2d}: {act['food']:2d} {act['wood']:2d} {act['stone']:2d} {act['gold']:2d} {act['build']:2d} {act['move']:2d} {act['idle']:2d} {act['other']:2d} | {int(r['food'])}/{int(r['wood'])}/{int(r['stone'])}/{int(r['gold'])} | {dict(mil)} | {dict(prod)}")
