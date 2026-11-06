import type { ReplayEvent, ReplayHiredUnit } from '../../model/replay_event';
import { hiredUnitLabel } from '../../model/replay_event';
import type { RisqTickLocation } from '../../model/tick_actions';
import type { RisqTickFrame, RisqTickUnitState, RisqTurnJournal } from '../../model/turn_journal';
import type { GameRisq, RisqSpaceLink } from '../../model/types';
import { RisqProducibleKind } from '../../model/types';
import { unitImage } from '../../rendering/assets/unit';
import { acrossBorder, seamNeighbors, zoneKey } from './event_adjacency';

export function hiredEvents(journal: RisqTurnJournal, game: GameRisq, player_id: number): ReplayEvent[] {
  return connectedHires(arrivals(journal, player_id), game.space_links).map(
    (group: RisqTickUnitState[]): ReplayEvent => {
      const units = composition(group);
      return {
        kind: 'production',
        player_id,
        tick: journal.tick_count,
        end_of_turn: true,
        title: `${units.map(hiredUnitLabel).join(', ')} hired`,
        icon: units.length === 1 ? unitImage(units[0].unit_id, true) : 'icons/unit64',
        production: { kind: RisqProducibleKind.UNIT, item_id: units[0].unit_id, source: 'hired' },
        hired_units: units,
        locations: [
          ...new Map(
            group.map((unit: RisqTickUnitState): [string, RisqTickLocation] => [
              zoneKey(location(unit)),
              location(unit),
            ])
          ).values(),
        ],
      };
    }
  );
}

function location(unit: RisqTickUnitState): RisqTickLocation {
  return { space: unit.space_coordinate, zone: unit.zone_coordinate };
}

function spaceKey(unit: RisqTickUnitState): string {
  return `space:${unit.space_coordinate.x},${unit.space_coordinate.y}`;
}

function arrivals(journal: RisqTurnJournal, player_id: number): RisqTickUnitState[] {
  const known = new Set(
    [journal.baseline, ...journal.frames].flatMap((frame: RisqTickFrame): number[] => [...frame.units.keys()])
  );
  const units: RisqTickUnitState[] = [];
  for (const frame of journal.boundary_events) {
    for (const unit of frame.units.values()) {
      if (known.has(unit.internal_id)) {
        continue;
      }
      known.add(unit.internal_id);
      if (unit.player_id === player_id) {
        units.push(unit);
      }
    }
  }
  return units;
}

function hireBuckets(units: RisqTickUnitState[]): Map<string, RisqTickUnitState[]> {
  const buckets = new Map<string, RisqTickUnitState[]>();
  for (const unit of units) {
    for (const key of [spaceKey(unit), zoneKey(location(unit))]) {
      const bucket = buckets.get(key) ?? [];
      bucket.push(unit);
      buckets.set(key, bucket);
    }
  }
  return buckets;
}

function connectedHires(units: RisqTickUnitState[], links: RisqSpaceLink[]): RisqTickUnitState[][] {
  const buckets = hireBuckets(units);
  const seams = seamNeighbors(links);
  const seen = new Set<number>();
  const groups: RisqTickUnitState[][] = [];
  for (const unit of units) {
    if (seen.has(unit.internal_id)) {
      continue;
    }
    const group = [unit];
    seen.add(unit.internal_id);
    for (const current of group) {
      const keys = [spaceKey(current), zoneKey(location(current))];
      const across = acrossBorder(location(current), seams);
      if (across) {
        keys.push(zoneKey(across));
      }
      for (const key of keys) {
        const neighbors = buckets.get(key) ?? [];
        buckets.delete(key);
        for (const neighbor of neighbors) {
          if (seen.has(neighbor.internal_id)) {
            continue;
          }
          seen.add(neighbor.internal_id);
          group.push(neighbor);
        }
      }
    }
    groups.push(group);
  }
  return groups;
}

function composition(units: RisqTickUnitState[]): ReplayHiredUnit[] {
  const counts = new Map<number, ReplayHiredUnit>();
  for (const unit of units) {
    const entry = counts.get(unit.unit_id) ?? { unit_id: unit.unit_id, display_name: unit.display_name, count: 0 };
    entry.count++;
    counts.set(unit.unit_id, entry);
  }
  return [...counts.values()].sort((a: ReplayHiredUnit, b: ReplayHiredUnit): number => a.unit_id - b.unit_id);
}
