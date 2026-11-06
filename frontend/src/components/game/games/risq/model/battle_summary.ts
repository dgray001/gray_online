import type { BattlePlayerStats, ReplayEvent } from './replay_event';

export interface BattleMetric {
  label: string;
  value: (stats: BattlePlayerStats) => number;
}

export const BATTLE_METRICS: BattleMetric[] = [
  { label: 'Kills', value: (stats: BattlePlayerStats): number => stats.kills },
  { label: 'Razes', value: (stats: BattlePlayerStats): number => stats.razes },
  { label: 'Damage', value: (stats: BattlePlayerStats): number => stats.unit_damage + stats.building_damage },
];

export function battleSides(stats: readonly BattlePlayerStats[], player_id: number): BattlePlayerStats[][] {
  return [
    stats.filter((entry: BattlePlayerStats): boolean => entry.player_id === player_id),
    stats.filter((entry: BattlePlayerStats): boolean => entry.player_id !== player_id),
  ];
}

export function battleTotals(events: readonly ReplayEvent[]): BattlePlayerStats[] {
  const players = new Map<number, BattlePlayerStats>();
  const participants = new Set<string>();
  for (const event of events) {
    for (const stats of event.battle_stats ?? []) {
      const total = players.get(stats.player_id);
      if (total) {
        for (const key of ['kills', 'deaths', 'razes', 'losses', 'unit_damage', 'building_damage'] as const) {
          total[key] += stats[key];
        }
      } else {
        players.set(stats.player_id, { ...stats, troops: 0, buildings: 0 });
      }
    }
    for (const participant of event.battle_participants ?? []) {
      if (participant.kind === 'resource') {
        continue;
      }
      const key = `${participant.kind}:${participant.internal_id}`;
      if (participants.has(key)) {
        continue;
      }
      participants.add(key);
      const total = players.get(participant.player_id);
      if (total) {
        total[participant.kind === 'unit' ? 'troops' : 'buildings']++;
      }
    }
  }
  return [...players.values()].sort((a: BattlePlayerStats, b: BattlePlayerStats): number => a.player_id - b.player_id);
}
