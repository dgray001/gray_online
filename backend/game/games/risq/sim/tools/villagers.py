"""What one AI's villagers do each turn in a debug replay (food/wood/stone/gold/build/move/idle), its bank, army and
building production. usage: python3 tools/villagers.py outputs/<name>/debug.log ai-<seat>"""
import json, re, sys, collections
from pathlib import Path
from typing import Any
from tool_search import AI_LINE, AI_ORDER, unpair
ZoneKey = tuple[int, int, int, int]
FrameKey = tuple[str, int]
MOVE = re.compile(r"Moving unit (\d+) Villager to zone\((-?\d+), (-?\d+)\) in space \((-?\d+), (-?\d+)\) tick=\d+ turn=(\d+) stamina=(\d+)")
TURN_METRICS = re.compile(r"turn report turn=(\d+) player=(\d+) metrics=(.*?) report=")


def location_key(location: dict[str, Any]) -> ZoneKey:
    return tuple(location[group][axis] for group in ("Space", "Zone") for axis in ("X", "Y"))


def gather_destination(target: int) -> ZoneKey:
    space, zone = unpair(target)
    return (*unpair(space), *unpair(zone))


def read_churn(path: Path) -> tuple[dict[FrameKey, Any], dict[FrameKey, Any], dict[FrameKey, Any], list[tuple[int, ...]]]:
    frames: dict[FrameKey, Any] = {}
    orders: dict[FrameKey, Any] = {}
    metrics: dict[FrameKey, Any] = {}
    moves: list[tuple[int, ...]] = []
    turns: collections.Counter[str] = collections.Counter()
    with path.open() as source:
        for line in source:
            if match := AI_LINE.search(line):
                ai, kind, payload = match.groups()
                if kind == "new turn data":
                    turns[ai] += 1
                    frames[(ai, turns[ai])] = json.loads(payload)
                elif kind == "submitting orders":
                    orders[(ai, turns[ai])] = {int(uid): (int(kind), int(target))
                        for subjects, kind, target, _ in AI_ORDER.findall(payload) for uid in subjects.split()}
            elif match := TURN_METRICS.search(line):
                turn, player, payload = match.groups()
                metrics[(f"ai-{player}", int(turn))] = json.loads(payload)
            elif match := MOVE.search(line):
                moves.append(tuple(map(int, match.groups())))
    return frames, orders, metrics, moves


def churn_metrics(path: Path, limit: int) -> dict[str, collections.Counter[str]]:
    frames, orders, metrics, moves = read_churn(path)
    owners: dict[int, str] = {}
    sources: dict[tuple[int, ZoneKey], dict[str, Any]] = {}
    for (ai, turn), frame in frames.items():
        for unit in frame["units"]:
            if unit["UnitID"] != 1:
                continue
            owners[unit["InternalID"]] = ai
            target = (unit.get("CurrentOrder") or {}).get("TargetResource")
            if target:
                sources[(turn, location_key(target["Location"]))] = target
        for building in frame["buildings"]:
            if building["Gatherable"]:
                sources[(turn, location_key(building["Location"]))] = {"Category": building["ResourceCategory"]}
    totals: dict[str, collections.Counter[str]] = collections.defaultdict(collections.Counter)
    tags: dict[tuple[int, int], str] = {}
    for (ai, turn), metric in metrics.items():
        if not 0 < turn <= limit:
            continue
        total = totals[ai]
        total["turns"] += 1
        total["granted"] += metric["VillagerStamina"]["Granted"]
        total["idle"] += metric["VillagerStamina"]["Wasted"]
        units = {unit["InternalID"]: unit for unit in frames[(ai, turn)]["units"] if unit["UnitID"] == 1}
        for uid, (kind, target_id) in orders.get((ai, turn), {}).items():
            if uid not in units or kind != 3:
                continue
            total["gather_orders"] += 1
            unit, destination = units[uid], gather_destination(target_id)
            previous = (unit.get("CurrentOrder") or {}).get("TargetResource")
            if not previous:
                total["new_jobs"] += 1
                tags[(uid, turn)] = "new"
                continue
            old_location = location_key(previous["Location"])
            if old_location == destination:
                total["same_target"] += 1
                tags[(uid, turn)] = "same"
                continue
            total["target_changes"] += 1
            if previous["AmountLeft"] <= 0:
                total["depleted_changes"] += 1
                tags[(uid, turn)] = "depleted"
                continue
            total["live_changes"] += 1
            current = sources.get((turn, destination), sources.get((turn + 1, destination)))
            if current is None:
                total["unknown_category"] += 1
            elif current["Category"] != previous["Category"]:
                total["category_changes"] += 1
            else:
                total["same_category_changes"] += 1
            tag = "working" if location_key(unit["Location"]) == old_location else "enroute"
            total[tag + "_changes"] += 1
            tags[(uid, turn)] = tag
    for uid, zx, zy, sx, sy, turn, cost in moves:
        ai = owners.get(uid, "")
        if (ai, turn) not in metrics or not 0 < turn <= limit:
            continue
        total = totals[ai]
        total["move_stamina"] += cost
        tag = tags.get((uid, turn))
        if tag:
            total["gather_move_stamina"] += cost
            total[tag + "_move_stamina"] += cost
    return totals


def churn_report(folder: Path, limit: int) -> None:
    results = json.loads((folder / "results.json").read_text())
    combined: dict[str, collections.Counter[str]] = collections.defaultdict(collections.Counter)
    games: list[dict[str, Any]] = []
    for sim in results["Sims"]:
        for game in sim["Games"]:
            seed = game["Result"]["Seed"]
            totals = churn_metrics(folder / f"debug_{seed}.log", limit)
            for seat, player in enumerate(game["Seats"]):
                name = sim["Stats"]["Players"][player]["Name"]
                counters = totals[f"ai-{seat}"]
                combined[name].update(counters)
                games.append({"map": sim["Name"], "seed": seed, "ai": name, **counters})
    print(json.dumps({"through_turn": limit, "by_ai": combined, "games": games}, indent=2))


if len(sys.argv) >= 3 and sys.argv[2] == "--churn":
    churn_report(Path(sys.argv[1]), int(sys.argv[3]) if len(sys.argv) > 3 else 100)
    sys.exit(0)


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
