import type { RisqZone, UnitByTypeData } from '../../model/types';
import type { Point2D } from '../../../../util/objects2d';
import { findOuterZoneIndex, zoneUnitSlotOffsets } from './geometry';

export function getZoneUnitSlots(
  zone: RisqZone,
  active_player_id: number,
  center_max_slots: number,
  edge_max_slots: number
): UnitByTypeData[][] {
  const capacity = findOuterZoneIndex(zone.coordinate) === -1 ? center_max_slots : edge_max_slots;
  return zone.unit_slots ?? (zone.unit_slots = buildZoneUnitSlots(zone, active_player_id, capacity));
}

export function unitSlotWorldPosition(
  zone: RisqZone,
  zone_coordinate: Point2D,
  hex_r: number,
  active_player_id: number,
  internal_id: number,
  center_max_slots: number,
  unit_r: number,
  edge_max_slots: number
): Point2D | undefined {
  const filled_slots = getZoneUnitSlots(zone, active_player_id, center_max_slots, edge_max_slots);
  const slot_index = filled_slots.findIndex((groups) => groups.some((g) => g.units.has(internal_id)));
  if (slot_index === -1) {
    return undefined;
  }
  return zoneUnitSlotOffsets(zone_coordinate, hex_r, filled_slots.length, unit_r)[slot_index];
}

function bandGroupsByUnitId(groups: UnitByTypeData[], band_size: number): UnitByTypeData[][] {
  const bands = new Map<number, UnitByTypeData[]>();
  for (const g of groups) {
    const band_index = Math.floor((g.unit_id - 1) / band_size);
    if (!bands.has(band_index)) {
      bands.set(band_index, []);
    }
    bands.get(band_index)!.push(g);
  }
  return [...bands.entries()].sort(([a], [b]) => a - b).map(([, g]) => g);
}

export function buildZoneUnitSlots(zone: RisqZone, active_player_id: number, num_slots: number): UnitByTypeData[][] {
  const active_groups = [...(zone.units_by_type.get(active_player_id)?.values() ?? [])].sort(
    (a, b) => a.unit_id - b.unit_id
  );
  const other_by_player = new Map(
    [...zone.units_by_type.entries()].filter(([player_id]) => player_id !== active_player_id)
  );
  const other_flat = [...other_by_player.values()]
    .flatMap((m) => [...m.values()])
    .sort((a, b) => a.unit_id - b.unit_id);
  const to_individuals = (groups: UnitByTypeData[]): UnitByTypeData[] =>
    groups.flatMap((g) => [...g.units].sort((a, b) => a - b).map((uid) => ({ ...g, units: new Set([uid]) })));

  let slots: UnitByTypeData[][] = [
    ...to_individuals(active_groups).map((g) => [g]),
    ...to_individuals(other_flat).map((g) => [g]),
  ];
  if (slots.length <= num_slots) {
    return slots;
  }
  slots = [...active_groups.map((g) => [g]), ...other_flat.map((g) => [g])];
  if (slots.length <= num_slots) {
    return slots;
  }
  slots = [...active_groups.map((g) => [g]), ...[...other_by_player.values()].map((m) => [...m.values()])];
  if (slots.length <= num_slots) {
    return slots;
  }
  slots = [...active_groups.map((g) => [g]), ...(other_flat.length > 0 ? [other_flat] : [])];
  if (slots.length <= num_slots) {
    return slots;
  }
  for (let band = 10; band <= 100; band += 10) {
    slots = [...bandGroupsByUnitId(active_groups, band), ...(other_flat.length > 0 ? [other_flat] : [])];
    if (slots.length <= num_slots) {
      return slots;
    }
  }
  return slots;
}
