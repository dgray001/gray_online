import type { RisqZone, UnitByTypeData } from '../../model/types';
import type { Point2D } from '../../../../util/objects2d';
import { findOuterZoneIndex, CENTER_ZONE_UNIT_SLOTS, EDGE_ZONE_UNIT_SLOTS, zoneUnitSlotOffsets } from './geometry';
/** Pixel offset of the specific unit-slot circle a given unit currently occupies, or undefined if not found */
export function unitSlotWorldPosition(
  zone: RisqZone,
  zone_coordinate: Point2D,
  hex_r: number,
  active_player_id: number,
  internal_id: number
): Point2D | undefined {
  const is_center = findOuterZoneIndex(zone_coordinate) === -1;
  const num_slots = is_center ? CENTER_ZONE_UNIT_SLOTS : EDGE_ZONE_UNIT_SLOTS;
  const filled_slots = buildZoneUnitSlots(zone, active_player_id, num_slots);
  const slot_index = filled_slots.findIndex((groups) => groups.some((g) => g.units.has(internal_id)));
  if (slot_index === -1) {
    return undefined;
  }
  return zoneUnitSlotOffsets(zone_coordinate, hex_r)[slot_index];
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

/** Assigns a zone's units to at most `num_slots` slots, active player first by unit id, coarsening others then active until it fits */
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
