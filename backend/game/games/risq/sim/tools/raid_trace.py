"""Positions and orders of one AI's units (blunts by default) turn by turn in a debug replay, with the enemy villagers,
soldiers and defensive buildings it can see. usage: python3 tools/raid_trace.py outputs/<name>/debug.log ai-<seat> <from turn> <to turn> [unit ids]"""
import json, re, sys, collections
log, who, t0, t1 = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
unit_ids = [int(x) for x in sys.argv[5].split(',')] if len(sys.argv) > 5 else [11]
OT = {1:'moveS',2:'moveZ',3:'gather',4:'build',7:'atkSpace',8:'atkZone',9:'atkUnit',10:'atkBldg',11:'autoAtkU',12:'autoAtkB',13:'garrison'}
turn = 0; mine = {}
sp = lambda l: f"{l['Space']['X']},{l['Space']['Y']}"
for line in open(log):
    m = re.search(r'ai (ai-\d+): new turn data (.*)$', line)
    if m and m.group(1) == who:
        d = json.loads(m.group(2)); turn += 1
        if not (t0 <= turn <= t1): continue
        mine = {u['InternalID']: u for u in d['units'] if u['UnitID'] in unit_ids}
        pos = collections.Counter(sp(u['Location']) for u in mine.values())
        cur = collections.Counter(OT.get((u.get('CurrentOrder') or {}).get('Kind', -1) + 0, 'idle') if u.get('CurrentOrder') else 'idle' for u in mine.values())
        ev = collections.Counter(sp(u['Location']) for u in (d.get('enemy_units') or []) if u['UnitID'] == 1)
        em = collections.Counter(sp(u['Location']) for u in (d.get('enemy_units') or []) if u['UnitID'] != 1)
        eb = sorted({f"{b['BuildingID']}@{sp(b['Location'])}" for b in (d.get('enemy_buildings') or []) if b['BuildingID'] in (1, 21, 23)})
        print(f"t{turn}: mine {dict(pos)} | enemy vils {dict(ev)} | enemy mil {dict(em)} | def bldgs {eb}")
        continue
    m = re.search(r'ai (ai-\d+): submitting orders (.*)$', line)
    if m and m.group(1) == who and t0 <= turn <= t1:
        orders = re.findall(r'Subjects:\[([\d ]*)\] OrderType:(\d+) TargetID:(-?\d+)', m.group(2))
        mine_orders = collections.Counter(OT.get(int(o[1]), o[1]) for o in orders for s in o[0].split() if int(s) in mine)
        if mine_orders: print(f"     orders to them: {dict(mine_orders)}")
