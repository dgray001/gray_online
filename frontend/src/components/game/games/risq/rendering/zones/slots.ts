import type { RisqZone, UnitByTypeData } from '../../model/types';
import type { Point2D } from '../../../../util/objects2d';
import {
  BUILDING_CIRCLE_RADIUS_MULTIPLIER,
  CENTER_ZONE_APOTHEM_MULTIPLIER,
  MAX_UNIT_RINGS,
  buildingLocalOffset,
  findOuterZoneIndex,
  unitSlotOffsets,
  zoneUnitSlotOffsets,
} from './geometry';
import type { RisqViewMode } from '../terrain';
import { unitVisibleInViewMode } from '../terrain';
import type { RisqDrawHost } from '../board/board_renderer';

export function getZoneUnitSlots(
  zone: RisqZone,
  active_player_id: number,
  center_max_slots: number,
  edge_max_slots: number,
  view_mode: RisqViewMode
): UnitByTypeData[][] {
  const capacity = findOuterZoneIndex(zone.coordinate) === -1 ? center_max_slots : edge_max_slots;
  return zone.unit_slots ?? (zone.unit_slots = buildZoneUnitSlots(zone, active_player_id, capacity, view_mode));
}

export function unitSlotWorldPosition(
  zone: RisqZone,
  zone_coordinate: Point2D,
  hex_r: number,
  active_player_id: number,
  internal_id: number,
  center_max_slots: number,
  unit_r: number,
  edge_max_slots: number,
  view_mode: RisqViewMode
): Point2D | undefined {
  const filled_slots = getZoneUnitSlots(zone, active_player_id, center_max_slots, edge_max_slots, view_mode);
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

export function buildZoneUnitSlots(
  zone: RisqZone,
  active_player_id: number,
  num_slots: number,
  view_mode: RisqViewMode
): UnitByTypeData[][] {
  const visible_by_type = new Map(
    [...zone.units_by_type]
      .map(([player_id, groups]): [number, UnitByTypeData[]] => [
        player_id,
        [...groups.values()].filter((g) => unitVisibleInViewMode(g.unit_type, view_mode)),
      ])
      .filter(([, groups]) => groups.length > 0)
  );
  const active_groups = [...(visible_by_type.get(active_player_id) ?? [])].sort((a, b) => a.unit_id - b.unit_id);
  const other_by_player = new Map(
    [...visible_by_type.entries()].filter(([player_id]) => player_id !== active_player_id)
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
  return [[...active_groups, ...other_flat]];
}

export function drawControlGroupLabel(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  slot: UnitByTypeData[] | undefined,
  r: Point2D
): void {
  const player_id = game.getPlayerId();
  const ids = slot
    ? slot
        .filter((group) => group.player_id === player_id)
        .flatMap((group) => [...group.units].filter((id) => game.selection.isUnitSelected(id)))
    : zone.building?.player_id === player_id
      ? [zone.building.internal_id]
      : [];
  const groups = game.control_groups?.memberships(slot ? 'unit' : 'building', ids);
  if (!groups?.length) {
    return;
  }
  const transform = ctx.getTransform();
  const scale = Math.hypot(transform.a, transform.b);
  const gap = 2 / scale;
  ctx.fillStyle = 'white';
  ctx.font = `bold ${10 / scale}px serif`;
  ctx.textAlign = 'right';
  ctx.textBaseline = 'bottom';
  ctx.fillText(groups.map((number) => number % 10).join(','), -Math.SQRT1_2 * r.x - gap, -Math.SQRT1_2 * r.y - gap);
}

const MAX_LAYOUT_RINGS = 8;
const REFERENCE_UNIT_DIAMETER = 44;
const UNIT_OVERLAP_FRACTION = 0.1;
const UNIT_GAP_FRACTION = 0.1;
const UNIT_SPACING_SEARCH_STEPS = 40;

/** Rings needed at max zoom so circles smaller than the reference size don't shrink the zoom range */
export function maxZoomRings(unit_diameter: number): number {
  const reference_gap = REFERENCE_UNIT_DIAMETER * (MAX_UNIT_RINGS + (MAX_UNIT_RINGS + 1) * UNIT_GAP_FRACTION);
  const rings = Math.ceil((reference_gap / unit_diameter - UNIT_GAP_FRACTION) / (1 + UNIT_GAP_FRACTION));
  return Math.min(MAX_LAYOUT_RINGS, Math.max(MAX_UNIT_RINGS, rings));
}

export function centerUnitMaxZoom(hex_r: number, unit_diameter: number, rings = MAX_UNIT_RINGS): number {
  const required_gap = unit_diameter * (rings + (rings + 1) * UNIT_GAP_FRACTION);
  const gap = (CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER) * hex_r;
  return required_gap / gap;
}

export function corpseUnitRadius(hex_r: number): number {
  return hex_r / (2 * centerUnitMaxZoom(1, 1));
}

function unitRingRadius(inner_radius: number, gap: number, unit_r: number, rings: number, ring: number): number {
  const diameter = 2 * unit_r;
  const clearance = (gap - rings * diameter) / (rings + 1);
  return inner_radius + unit_r + clearance + ring * (diameter + clearance);
}

function unitRingLimit(gap: number, unit_r: number): number {
  const diameter = 2 * unit_r;
  const limit = Math.floor((gap + diameter * UNIT_OVERLAP_FRACTION) / (diameter * (1 - UNIT_OVERLAP_FRACTION)));
  return Math.max(1, Math.min(MAX_LAYOUT_RINGS, limit));
}

interface CenterUnitRing {
  radius: number;
  capacity: number;
}

function centerUnitRings(
  hex_r: number,
  unit_r: number,
  rings: number,
  spacing_multiplier = 1 - UNIT_OVERLAP_FRACTION
): CenterUnitRing[] {
  const gap = (CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER) * hex_r;
  return Array.from({ length: rings }, (_, ring) => {
    const radius = unitRingRadius(BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r, gap, unit_r, rings, ring);
    const ratio = (unit_r * spacing_multiplier) / radius;
    return { radius, capacity: ratio >= 1 ? 1 : Math.floor(Math.PI / Math.asin(ratio)) };
  });
}

function centerUnitRingLayout(hex_r: number, unit_r: number, count: number): CenterUnitRing[] {
  const gap = (CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER) * hex_r;
  const max_rings = unitRingLimit(gap, unit_r);
  for (let rings = 1; rings <= max_rings; rings++) {
    const layout = centerUnitRings(hex_r, unit_r, rings);
    if (count <= layout.reduce((total, ring) => total + ring.capacity, 0) || rings === max_rings) {
      return layout;
    }
  }
  return centerUnitRings(hex_r, unit_r, 1);
}

export function centerUnitSlotCapacity(hex_r: number, unit_r: number): number {
  return centerUnitRingLayout(hex_r, unit_r, Infinity).reduce((total, ring) => total + ring.capacity, 0);
}

function centerRingSlotOffsets(layout: CenterUnitRing[], count: number): Point2D[] {
  let remaining = count;
  return layout.flatMap(({ radius, capacity }, ring) => {
    const occupied = Math.min(capacity, remaining);
    remaining -= occupied;
    const offset = (ring % 2) * (Math.PI / layout[0].capacity);
    return Array.from({ length: occupied }, (_, i) => {
      const angle = offset - (2 * Math.PI * i) / occupied;
      return {
        x: radius * Math.cos(angle),
        y: radius * Math.sin(angle),
      };
    });
  });
}

export function centerUnitSlotLocalOffsets(hex_r: number, count: number, unit_r: number): Point2D[] {
  return centerRingSlotOffsets(centerUnitRingLayout(hex_r, unit_r, count), count);
}

interface EdgeUnitPath {
  corner: Point2D;
  end: Point2D;
  length: number;
}

function edgeUnitGap(hex_r: number): number {
  const building = buildingLocalOffset(false, hex_r);
  return (
    Math.min(building.x - CENTER_ZONE_APOTHEM_MULTIPLIER * hex_r, building.x / 2) -
    BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r
  );
}

function edgeUnitPath(hex_r: number, unit_r: number, rings: number, ring: number): EdgeUnitPath {
  const inset = unitRingRadius(0, edgeUnitGap(hex_r), unit_r, rings, rings - ring - 1);
  const slope_normal = Math.sqrt(3) / 2;
  const inner_x = CENTER_ZONE_APOTHEM_MULTIPLIER * hex_r + inset;
  const outer_x = Math.max(inner_x, slope_normal * hex_r - unit_r);
  const corner = { x: inner_x, y: (inner_x / 2 - inset) / slope_normal };
  const end = { x: outer_x, y: (outer_x / 2 - inset) / slope_normal };
  return { corner, end, length: corner.y + Math.hypot(end.x - corner.x, end.y - corner.y) };
}

function edgePathPoint(path: EdgeUnitPath, distance: number): Point2D {
  if (distance <= path.corner.y) {
    return { x: path.corner.x, y: distance };
  }
  const fraction = (distance - path.corner.y) / (path.length - path.corner.y);
  return {
    x: path.corner.x + fraction * (path.end.x - path.corner.x),
    y: path.corner.y + fraction * (path.end.y - path.corner.y),
  };
}

function nextEdgePathDistance(path: EdgeUnitPath, distance: number, spacing: number): number {
  if (distance + spacing <= path.corner.y) {
    return distance + spacing;
  }
  let next = distance + spacing;
  if (distance < path.corner.y) {
    const vertical_remaining = path.corner.y - distance;
    next =
      path.corner.y -
      vertical_remaining / 2 +
      Math.sqrt(spacing * spacing - (3 * vertical_remaining * vertical_remaining) / 4);
  }
  return next <= path.length ? next : Infinity;
}

function packEdgeUnitPath(path: EdgeUnitPath, spacing: number, phase: number): Point2D[] {
  const positions: Point2D[] = [];
  let distance = phase * spacing;
  while (distance <= path.length) {
    const point = edgePathPoint(path, distance);
    positions.push(point);
    if (point.y !== 0) {
      positions.push({ x: point.x, y: -point.y });
    }
    distance = nextEdgePathDistance(path, distance, spacing);
  }
  return positions;
}

function occupiedEdgeRowOffsets(path: EdgeUnitPath, unit_r: number, count: number): Point2D[] {
  if (count === 0) {
    return [];
  }
  const phase = count % 2 === 0 ? 0.5 : 0;
  let low = 2 * unit_r * (1 - UNIT_OVERLAP_FRACTION);
  let high = 2 * unit_r * (1 + UNIT_GAP_FRACTION);
  const roomy = packEdgeUnitPath(path, high, phase);
  if (roomy.length >= count) {
    return roomy.slice(0, count);
  }
  for (let step = 0; step < UNIT_SPACING_SEARCH_STEPS; step++) {
    const middle = (low + high) / 2;
    if (packEdgeUnitPath(path, middle, phase).length >= count) {
      low = middle;
    } else {
      high = middle;
    }
  }
  return packEdgeUnitPath(path, low, phase).slice(0, count);
}

export function buildEdgeUnitRows(
  hex_r: number,
  unit_r: number,
  rings: number,
  spacing_multiplier = 1 - UNIT_OVERLAP_FRACTION
): Point2D[][] {
  const rows: Point2D[][] = [];
  const spacing = 2 * unit_r * spacing_multiplier;
  for (let ring = 0; ring < rings; ring++) {
    const path = edgeUnitPath(hex_r, unit_r, rings, ring);
    const phase = ring === 0 ? 0 : ((rows[0].length % 2 === 0 ? 0.5 : 0) + (ring % 2) * 0.5) % 1;
    let positions = packEdgeUnitPath(path, spacing, phase);
    if (ring === 0) {
      const alternate = packEdgeUnitPath(path, spacing, 0.5);
      if (alternate.length > positions.length) {
        positions = alternate;
      }
    }

    rows.push(positions);
  }
  return rows;
}

function edgeUnitRowLayout(hex_r: number, unit_r: number, count: number): Point2D[][] {
  const cache = unitSlotOffsets(hex_r, unit_r);
  const gap = edgeUnitGap(hex_r);
  let best: Point2D[][] = [];
  let best_capacity = 0;
  for (let rings = 1; rings <= unitRingLimit(gap, unit_r); rings++) {
    const rows = cache.edge_rows.get(rings) ?? buildEdgeUnitRows(hex_r, unit_r, rings);
    cache.edge_rows.set(rings, rows);
    const capacity = rows.reduce((total, row) => total + row.length, 0);
    if (count <= capacity) {
      return rows;
    }
    if (capacity > best_capacity) {
      best = rows;
      best_capacity = capacity;
    }
  }
  return best;
}

export function edgeUnitSlotCapacity(hex_r: number, unit_r: number): number {
  return edgeUnitRowLayout(hex_r, unit_r, Infinity).reduce((total, row) => total + row.length, 0);
}

export function edgeOccupiedSlotOffsets(hex_r: number, count: number, unit_r: number): Point2D[] {
  const rows = edgeUnitRowLayout(hex_r, unit_r, count);
  let remaining = count;
  return rows.flatMap((row, ring) => {
    const occupied = Math.min(row.length, remaining);
    remaining -= occupied;
    return occupiedEdgeRowOffsets(edgeUnitPath(hex_r, unit_r, rows.length, ring), unit_r, occupied);
  });
}

export function zoneCorpseSlots(zone_coordinate: Point2D): Point2D[] {
  const radius = corpseUnitRadius(1);
  if (findOuterZoneIndex(zone_coordinate) !== -1) {
    return buildEdgeUnitRows(1, radius, MAX_UNIT_RINGS, 1).flat();
  }
  const rings = centerUnitRings(1, radius, MAX_UNIT_RINGS, 1);
  return centerRingSlotOffsets(
    rings,
    rings.reduce((total, ring) => total + ring.capacity, 0)
  );
}

export function zoneCorpsePoint(zone_coordinate: Point2D, ring: number, fraction: number): Point2D {
  const unit_r = corpseUnitRadius(1);
  if (findOuterZoneIndex(zone_coordinate) === -1) {
    const radius = unitRingRadius(
      BUILDING_CIRCLE_RADIUS_MULTIPLIER,
      CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER,
      unit_r,
      MAX_UNIT_RINGS,
      ring
    );
    const angle = 2 * Math.PI * fraction;
    return { x: radius * Math.cos(angle), y: radius * Math.sin(angle) };
  }
  const path = edgeUnitPath(1, unit_r, MAX_UNIT_RINGS, ring);
  const distance = (2 * fraction - 1) * path.length;
  const point = edgePathPoint(path, Math.abs(distance));
  return { x: point.x, y: Math.sign(distance) * point.y };
}
