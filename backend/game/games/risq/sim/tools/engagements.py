"""Soldier-vs-soldier fights in a debug replay, per turn and space: soldiers each side committed (dealt or took
damage), power (blunt 1, piercer 2, heavy 4), losses, and overkill % (damage that landed on units already dead that tick).
usage: python3 tools/engagements.py outputs/<name>/debug.log <seat of the AI to report as "v0.6"> [max fights] [min soldiers per side]"""
import re,json,sys,collections
log, me = sys.argv[1], int(sys.argv[2])
HP={1:8,11:11,12:16,13:50}; SOLD={11,12,13}; POW={11:1,12:2,13:4}
pos={}; hp={}; kind={}; turn=0
E=collections.defaultdict(lambda: {'part':{'mine':set(),'theirs':set()},'lost':{'mine':collections.Counter(),'theirs':collections.Counter()},'eff':collections.Counter(),'waste':collections.Counter()})
ticks=collections.defaultdict(list)
def flush():
    for key,hits in sorted(ticks.items()):
        by=collections.defaultdict(list)
        for h in hits: by[h[2]].append(h)
        for tgt,hs in by.items():
            if kind.get(tgt) not in SOLD: continue
            t=hs[0][0]; sp=pos.get(tgt); e=E[(t,sp)]
            side=hs[0][1]; other='theirs' if side=='mine' else 'mine'
            for h in hs:
                if kind.get(h[3]) in SOLD: e['part'][side].add(h[3])
            e['part'][other].add(tgt)
            total=sum(h[4] for h in hs); left=hp.get(tgt,HP.get(kind.get(tgt),10))
            eff=min(total,max(0,left)); e['eff'][side]+=eff; e['waste'][side]+=total-eff
            hp[tgt]=left-eff
            if hp[tgt]<=1e-6 and left>1e-6: e['lost'][other][kind[tgt]]+=1
    ticks.clear()
for line in open(log):
    m=re.search(r'ai (ai-\d): new turn data (.*)$',line)
    if m:
        if m.group(1)=='ai-0': turn+=1
        for u in json.loads(m.group(2))['units']:
            k='u'+str(u['InternalID']); pos[k]=(u['Location']['Space']['X'],u['Location']['Space']['Y']); kind[k]=u['UnitID']; hp.setdefault(k,HP.get(u['UnitID'],10))
        continue
    m=re.search(r'combat turn=(\d+) tick=(\d+): (\d+) ([ub])(\d+) \(player (\d).*hits (\d+) ([ub])(\d+) \(player (\d)\) for ([\d.]+)',line)
    if m:
        t,tk,aid,ak,auid,ap,tid,tk2,tuid,tp,dd=m.groups()
        a=ak+aid; tg=tk2+tid
        if ak=='u': kind.setdefault(a,int(auid))
        if tk2=='u': kind.setdefault(tg,int(tuid))
        ticks[(int(t),int(tk))].append((int(t),'mine' if int(ap)==me else 'theirs',tg,a,float(dd)))
    elif 'state_hash' in line: flush()
flush()
def power(ids): return sum(POW.get(kind.get(i),0) for i in ids)
shown=0
MINP=int(sys.argv[4]) if len(sys.argv)>4 else 3
for (t,sp),e in sorted(E.items(), key=lambda kv:(kv[0][0], str(kv[0][1]))):
    pm,pt=e['part']['mine'],e['part']['theirs']
    if len(pm)<MINP or len(pt)<MINP: continue
    om=100*e['waste']['mine']/max(1,e['eff']['mine']+e['waste']['mine']); ot=100*e['waste']['theirs']/max(1,e['eff']['theirs']+e['waste']['theirs'])
    lm=sum(e['lost']['mine'].values()); lt=sum(e['lost']['theirs'].values())
    print(f"  t{t:2} {str(sp):9} committed v0.6 {len(pm):2} (pow {power(pm):2}) vs v0.5 {len(pt):2} (pow {power(pt):2}) | lost v0.6 {lm:2} v0.5 {lt:2} | overkill v0.6 {om:3.0f}% v0.5 {ot:3.0f}%")
    shown+=1
    if shown>=int(sys.argv[3] if len(sys.argv)>3 else 4): break
