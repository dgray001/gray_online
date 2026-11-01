import { AXIAL_DIRECTION_VECTORS } from '../../../../util/objects2d';
import type { ReplayEvent } from '../../model/replay_event';
import type { RisqTickLocation, RisqTickEffect, RisqTickTarget } from '../../model/tick_actions';
import type { RisqTurnJournal } from '../../model/turn_journal';
import type { RisqSpaceLink } from '../../model/types';
import { battleStats } from './battle_stats';
import type { BattleSample } from './battle_stats';

type BattleAttack = BattleSample & { locations: RisqTickLocation[] };

function seamNeighbors(links: RisqSpaceLink[]): Map<string, RisqTickLocation> {
  const neighbors = new Map<string, RisqTickLocation>();
  for (const link of links) {
    const from = { space: link.from, zone: AXIAL_DIRECTION_VECTORS[link.direction] };
    const to = { space: link.to, zone: AXIAL_DIRECTION_VECTORS[link.to_direction] };
    neighbors.set(zoneKey(from), to);
    neighbors.set(zoneKey(to), from);
  }
  return neighbors;
}

function acrossBorder(location: RisqTickLocation, seams: Map<string, RisqTickLocation>): RisqTickLocation | undefined {
  if (seams.has(zoneKey(location))) {
    return seams.get(zoneKey(location));
  }
  if (location.zone.x === 0 && location.zone.y === 0) {
    return undefined;
  }
  const neighbor = {
    space: { x: location.space.x + location.zone.x, y: location.space.y + location.zone.y },
    zone: { x: -location.zone.x, y: -location.zone.y },
  };
  return seams.has(zoneKey(neighbor)) ? undefined : neighbor;
}

function zoneKey(location: RisqTickLocation): string {
  return `zone:${location.space.x},${location.space.y},${location.zone.x},${location.zone.y}`;
}

function root(parents: number[], index: number): number {
  while (parents[index] !== index) {
    parents[index] = parents[parents[index]];
    index = parents[index];
  }
  return index;
}

export function battleEvents(journal: RisqTurnJournal, player_id: number, links: RisqSpaceLink[]): ReplayEvent[] {
  const attacks = attackSamples(journal, player_id);
  const seams = seamNeighbors(links);
  const parents = attacks.map((_attack: BattleAttack, index: number): number => index);
  const buckets = new Map<string, number[]>();
  const groups = new Map<number, BattleAttack[]>();
  attacks.forEach((attack: BattleAttack, index: number): void => {
    const keys = new Set(
      attack.locations.flatMap((location: RisqTickLocation): string[] => [
        `space:${location.space.x},${location.space.y}`,
        zoneKey(location),
      ])
    );
    const nearby = new Set(keys);
    for (const location of attack.locations) {
      const neighbor = acrossBorder(location, seams);
      if (neighbor) {
        nearby.add(zoneKey(neighbor));
      }
    }
    for (const key of nearby) {
      for (let gap = 0; gap <= 2; gap++) {
        for (const other of buckets.get(`${key}/${attack.tick - gap}`) ?? []) {
          parents[root(parents, index)] = root(parents, other);
        }
      }
    }
    for (const key of keys) {
      const at_tick = `${key}/${attack.tick}`;
      const bucket = buckets.get(at_tick) ?? [];
      bucket.push(index);
      buckets.set(at_tick, bucket);
    }
  });
  attacks.forEach((attack: BattleAttack, index: number): void => {
    const id = root(parents, index);
    const group = groups.get(id) ?? [];
    group.push(attack);
    groups.set(id, group);
  });
  return [...groups.values()].map(
    (group: BattleAttack[]): ReplayEvent => ({
      kind: 'battle',
      tick: group[0].tick,
      end_tick: group[group.length - 1].tick,
      battle_stats: battleStats(journal, group),
      player_id,
      title: 'Battle',
      icon: 'icons/swords128',
      locations: [
        ...new Map(
          group
            .flatMap((attack: BattleAttack): RisqTickLocation[] => attack.locations)
            .map((location: RisqTickLocation): [string, RisqTickLocation] => [zoneKey(location), location])
        ).values(),
      ],
    })
  );
}

function attackSamples(journal: RisqTurnJournal, player_id: number): BattleAttack[] {
  const attacks: BattleAttack[] = [];
  for (const actor of [...journal.units.values(), ...journal.buildings.values()]) {
    for (const action of actor.tick_actions) {
      if (action.execute.outcome !== 'executed' || !['attack', 'receive_damage'].includes(action.execute.kind)) {
        continue;
      }
      if (actor.player_id !== player_id && action.execute.target?.player_id !== player_id) {
        continue;
      }
      attacks.push({
        tick: action.tick,
        effects: action.execute.effects,
        participants: [
          {
            kind: journal.units.get(actor.internal_id) === actor ? 'unit' : 'building',
            internal_id: actor.internal_id,
            player_id: actor.player_id,
          },
          ...action.execute.effects.flatMap((effect: RisqTickEffect): RisqTickTarget[] => [
            ...(effect.actor ? [effect.actor] : []),
            ...(effect.target ? [effect.target] : []),
          ]),
        ],
        locations: [
          action.intent.location,
          ...(action.execute.target_location ? [action.execute.target_location] : []),
        ],
      });
    }
  }
  return attacks.sort((a: BattleAttack, b: BattleAttack): number => a.tick - b.tick);
}
