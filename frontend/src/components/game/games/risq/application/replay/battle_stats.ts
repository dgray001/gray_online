import type { BattlePlayerStats } from '../../model/replay_event';
import type { RisqTickEffect, RisqTickTarget } from '../../model/tick_actions';
import type { RisqTickBuildingState, RisqTurnJournal } from '../../model/turn_journal';

export type BattleSample = { tick: number; effects: RisqTickEffect[]; participants: RisqTickTarget[] };
type EntityState = { health: number; deleted: boolean; construction_health?: number };

function constructionHealth(building: RisqTickBuildingState): number {
  const progress =
    building.construction_stamina_total > 0
      ? Math.min(1, Math.max(0, 1 - building.stamina_remaining / building.construction_stamina_total))
      : 1;
  return Math.round(building.combat_stats.max_health * (0.1 + 0.9 * progress) * 1e9) / 1e9;
}

function settleTarget(
  hits: RisqTickEffect[],
  before: EntityState | undefined,
  after: EntityState | undefined,
  effects: RisqTickEffect[],
  players: Map<number, BattlePlayerStats>
): void {
  const target = hits[0].target!;
  const total = hits.reduce((sum: number, hit: RisqTickEffect): number => sum + hit.amount, 0);
  const healing = effects
    .filter(
      (effect: RisqTickEffect): boolean =>
        effect.kind === 'heal' && !!effect.target && entityKey(effect.target) === entityKey(target)
    )
    .reduce((sum: number, effect: RisqTickEffect): number => sum + effect.amount, 0);
  const construction_healing = (after?.construction_health ?? 0) - (before?.construction_health ?? 0);
  const actual = before ? Math.min(total, Math.max(0, before.health + healing + construction_healing)) : total;
  const contributions = new Map<number, number>();
  for (const hit of hits) {
    if (!hit.actor || total <= 0) {
      continue;
    }
    const stats = players.get(hit.actor.player_id);
    if (stats) {
      stats[target.kind === 'unit' ? 'unit_damage' : 'building_damage'] += (actual * hit.amount) / total;
    }
    contributions.set(hit.actor.player_id, (contributions.get(hit.actor.player_id) ?? 0) + hit.amount);
  }
  if (!before || before.deleted || !after || (!after.deleted && after.health > 0) || total <= 0) {
    return;
  }
  const winner = [...contributions].sort(
    (a: [number, number], b: [number, number]): number => b[1] - a[1] || a[0] - b[0]
  )[0];
  const killer = winner && players.get(winner[0]);
  const victim = players.get(target.player_id);
  if (killer) {
    killer[target.kind === 'unit' ? 'kills' : 'razes']++;
  }
  if (victim) {
    victim[target.kind === 'unit' ? 'deaths' : 'losses']++;
  }
}

function entityKey(target: RisqTickTarget): string {
  return `${target.kind}:${target.internal_id}`;
}

function emptyStats(player_id: number): BattlePlayerStats {
  return {
    player_id,
    troops: 0,
    buildings: 0,
    kills: 0,
    deaths: 0,
    razes: 0,
    losses: 0,
    unit_damage: 0,
    building_damage: 0,
  };
}

export function battleStats(journal: RisqTurnJournal, samples: BattleSample[]): BattlePlayerStats[] {
  const players = new Map<number, BattlePlayerStats>();
  const participants = new Map<string, RisqTickTarget>();
  const effects = new Map<number, RisqTickEffect>();
  for (const sample of samples) {
    for (const target of sample.participants) {
      participants.set(entityKey(target), target);
    }
    for (const effect of sample.effects) {
      effects.set(effect.id, effect);
    }
  }
  for (const target of participants.values()) {
    const stats = players.get(target.player_id) ?? emptyStats(target.player_id);
    if (target.kind === 'unit') {
      stats.troops++;
    }
    if (target.kind === 'building') {
      stats.buildings++;
    }
    players.set(target.player_id, stats);
  }
  accumulateDamage(journal, effects, players);
  return [...players.values()].sort((a: BattlePlayerStats, b: BattlePlayerStats): number => a.player_id - b.player_id);
}

function accumulateDamage(
  journal: RisqTurnJournal,
  effects: Map<number, RisqTickEffect>,
  players: Map<number, BattlePlayerStats>
): void {
  const states = new Map<string, EntityState>();
  for (const frame of [journal.baseline, ...journal.frames]) {
    const next = new Map(states);
    for (const unit of frame.units.values()) {
      next.set(`unit:${unit.internal_id}`, { health: unit.combat_stats.health, deleted: unit.deleted });
    }
    for (const building of frame.buildings.values()) {
      next.set(`building:${building.internal_id}`, {
        health: building.combat_stats.health,
        deleted: building.deleted,
        construction_health: constructionHealth(building),
      });
    }
    const targets = new Map<string, RisqTickEffect[]>();
    for (const effect of effects.values()) {
      if (effect.tick !== frame.tick || effect.kind !== 'damage' || !effect.target) {
        continue;
      }
      const key = entityKey(effect.target);
      const hits = targets.get(key) ?? [];
      hits.push(effect);
      targets.set(key, hits);
    }
    for (const [key, hits] of targets) {
      settleTarget(hits, states.get(key), next.get(key), frame.effects, players);
    }
    for (const [key, state] of next) {
      states.set(key, state);
    }
  }
}
