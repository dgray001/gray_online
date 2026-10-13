import { apiPost } from './api';
import { getUrlParam } from './url';
import { DEV } from './util';

const FLUSH_MS = 5000;
const OVER_BUDGET_MS = 20;

declare interface PerfAggregate {
  n: number;
  total_ms: number;
  max_ms: number;
  over_budget: number;
}

type PerfEntry = Record<string, unknown>;

const enabled = DEV && getUrlParam('perf') === '1';
const aggregates = new Map<string, PerfAggregate>();
let events: PerfEntry[] = [];
let started = false;

/** Perf logging is opt-in: DEV build plus ?perf=1 in the url */
export function perfEnabled(): boolean {
  return enabled;
}

/** Aggregates repeated timings (per-frame work) into one entry per flush */
export function perfRecord(kind: string, ms: number, budget_ms: number = OVER_BUDGET_MS): void {
  if (!enabled) {
    return;
  }
  const agg = aggregates.get(kind) ?? { n: 0, total_ms: 0, max_ms: 0, over_budget: 0 };
  agg.n++;
  agg.total_ms += ms;
  agg.max_ms = Math.max(agg.max_ms, ms);
  agg.over_budget += ms > budget_ms ? 1 : 0;
  aggregates.set(kind, agg);
}

/** Logs one discrete entry, such as a snapshot apply with its size and turn */
export function perfEvent(kind: string, fields: PerfEntry): void {
  if (enabled) {
    events.push({ ts: Date.now(), kind, ...fields });
  }
}

function drainEntries(): PerfEntry[] {
  const ts = Date.now();
  const memory = (performance as Performance & { memory?: Record<string, number> }).memory;
  perfEvent('heap', { used_mb: memory ? memory['usedJSHeapSize'] / 1048576 : undefined });
  const entries = events;
  events = [];
  for (const [kind, agg] of aggregates) {
    entries.push({
      ts,
      kind,
      n: agg.n,
      avg_ms: agg.total_ms / agg.n,
      max_ms: agg.max_ms,
      over_budget: agg.over_budget,
    });
  }
  aggregates.clear();
  return entries;
}

export function flushPerf(): void {
  if (enabled) {
    void apiPost('dev/perf/log', drainEntries());
  }
}

/** Idempotent; tells the backend to open a log file and starts periodic uploads */
export function startPerf(): void {
  if (!enabled || started) {
    return;
  }
  started = true;
  void apiPost('dev/perf/enable', {});
  setInterval(flushPerf, FLUSH_MS);
  window.addEventListener('pagehide', (): void => {
    navigator.sendBeacon('/api/dev/perf/log', new Blob([JSON.stringify(drainEntries())], { type: 'application/json' }));
  });
  new PerformanceObserver((list: PerformanceObserverEntryList): void => {
    for (const entry of list.getEntries()) {
      perfEvent('longtask', { ms: entry.duration });
    }
  }).observe({ type: 'longtask' });
}
