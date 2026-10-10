import type { Point2D } from '../../../../util/objects2d';
import { coordinateToIndex } from '../../model/coordinates';
import { equalsPoint2D, rotatePoint } from '../../../../util/objects2d';
import type { RisqSpace, RisqZone } from '../../model/types';
/** space.zones[][] array indices for the six outer zones, in draw-loop order (index i -> rotation (PI/3)*(i+1)) */
export const OUTER_ZONE_INDICES: Point2D[] = [
  { x: 2, y: 1 },
  { x: 2, y: 0 },
  { x: 1, y: 0 },
  { x: 0, y: 0 },
  { x: 0, y: 1 },
  { x: 1, y: 2 },
];

export const MERCENARY_RING_OFFSET_MULTIPLIER = 0.6;

export const MERCENARY_SLOT_SPACING_MULTIPLIER = 0.9;

// Equalizes building-circle clearance to center-zone sides and the three unit-placement edges of edge zones.
export const INNER_ZONE_MULTIPLIER = 0.7 / Math.sqrt(3);

export const BUILDING_CIRCLE_RADIUS_MULTIPLIER = 0.13;

export const CENTER_ZONE_APOTHEM_MULTIPLIER = (Math.sqrt(3) / 2) * INNER_ZONE_MULTIPLIER;
// midpoint between the building circle and the center zone's apothem, so both gaps match regardless of unit-circle size
export const CENTER_UNIT_RING_RADIUS_MULTIPLIER =
  (CENTER_ZONE_APOTHEM_MULTIPLIER + BUILDING_CIRCLE_RADIUS_MULTIPLIER) / 2;
const MERCENARY_SLOT_RADIUS_MULTIPLIER = 0.85 * CENTER_UNIT_RING_RADIUS_MULTIPLIER * Math.sin(Math.PI / 12);

const EDGE_BUILDING_RADIAL_MULTIPLIER = 0.7; // angle 0 hits an edge midpoint, so the boundary is the apothem (~0.866), not 1.0

export function findOuterZoneIndex(zone_coordinate: Point2D): number {
  const index = coordinateToIndex(1, zone_coordinate);
  return OUTER_ZONE_INDICES.findIndex((dv) => equalsPoint2D(dv, index));
}

export function hexagonVertices(hex_r: number): Point2D[] {
  return Array.from({ length: 6 }, (_: unknown, i: number): Point2D => {
    const angle = (Math.PI / 3) * i + Math.PI / 6;
    return { x: hex_r * Math.cos(angle), y: hex_r * Math.sin(angle) };
  });
}

export function zoneVertices(zone_coordinate: Point2D, hex_r: number): Point2D[] {
  const direction = findOuterZoneIndex(zone_coordinate);
  const inner_r = INNER_ZONE_MULTIPLIER * hex_r;
  if (direction === -1) {
    return hexagonVertices(inner_r);
  }
  const angle = (Math.PI / 3) * direction;
  return [
    [inner_r, angle + Math.PI / 6],
    [inner_r, angle + Math.PI / 2],
    [hex_r, angle + Math.PI / 2],
    [hex_r, angle + Math.PI / 6],
  ].map(([radius, a]: number[]): Point2D => ({ x: radius * Math.cos(a), y: radius * Math.sin(a) }));
}

/** Clips ctx to one zone's shape (direction -1 for center), in the space's real on-screen frame (center c, radius r) */
export function clipToZone(ctx: CanvasRenderingContext2D, c: Point2D, r: number, zone_coordinate: Point2D): void {
  ctx.beginPath();
  for (const point of zoneVertices(zone_coordinate, r)) {
    ctx.lineTo(c.x + point.x, c.y + point.y);
  }
  ctx.closePath();
  ctx.clip();
}

/** Local-frame (pre-space-rotation) offset of a zone's building/resource circle from its space's center */
export function buildingLocalOffset(is_center: boolean, hex_r: number): Point2D {
  return is_center ? { x: 0, y: 0 } : { x: EDGE_BUILDING_RADIAL_MULTIPLIER * hex_r, y: 0 };
}

function isInsideEdgeZoneFootprint(p: Point2D, hex_r: number): boolean {
  const radius = Math.hypot(p.x, p.y);
  if (radius < INNER_ZONE_MULTIPLIER * hex_r) {
    return false;
  }
  const angle = Math.atan2(p.y, p.x);
  if (Math.abs(angle) > Math.PI / 6) {
    return false;
  }
  const boundary = ((Math.sqrt(3) / 2) * hex_r) / Math.cos(angle);
  return radius <= boundary;
}

function distanceToEdgeZoneBoundary(building: Point2D, angle: number, hex_r: number): number {
  const dir = { x: Math.cos(angle), y: Math.sin(angle) };
  let lo = 0;
  let hi = 2 * hex_r;
  for (let i = 0; i < 30; i++) {
    const mid = (lo + hi) / 2;
    const p = { x: building.x + mid * dir.x, y: building.y + mid * dir.y };
    if (isInsideEdgeZoneFootprint(p, hex_r)) {
      lo = mid;
    } else {
      hi = mid;
    }
  }
  return lo;
}

/** Edge-zone point along an angle from the building, `fraction` of the way from the building circle to the zone boundary */
export function edgeSlotAtAngle(angle: number, hex_r: number, fraction = 0.5): Point2D {
  const building = buildingLocalOffset(false, hex_r);
  const d = distanceToEdgeZoneBoundary(building, angle, hex_r);
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const r = building_r + fraction * (d - building_r);
  return { x: building.x + r * Math.cos(angle), y: building.y + r * Math.sin(angle) };
}

export const MAX_UNIT_RINGS = 3;
const UNIT_OVERLAP_FRACTION = 0.1;
const UNIT_GAP_FRACTION = 0.1;
const UNIT_SPACING_SEARCH_STEPS = 40;

export function centerUnitMaxZoom(hex_r: number, unit_diameter: number): number {
  const required_gap = unit_diameter * (MAX_UNIT_RINGS + (MAX_UNIT_RINGS + 1) * UNIT_GAP_FRACTION);
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
  return Math.max(1, Math.min(MAX_UNIT_RINGS, limit));
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

function centerUnitSlotLocalOffsets(hex_r: number, count: number, unit_r: number): Point2D[] {
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

function edgeOuterApproachPoint(hex_r: number): Point2D {
  const apothem = Math.sqrt(3) / 2;
  const corner_x = apothem;
  const corner_y = corner_x / Math.sqrt(3);
  const direction = { x: -0.5, y: -apothem };
  const clearance_ratio = 0.5;
  const offset_x = corner_x - EDGE_BUILDING_RADIAL_MULTIPLIER;
  const quadratic = 1 - clearance_ratio * clearance_ratio;
  const linear = offset_x * direction.x + corner_y * direction.y - BUILDING_CIRCLE_RADIUS_MULTIPLIER * clearance_ratio;
  const constant = offset_x * offset_x + corner_y * corner_y - BUILDING_CIRCLE_RADIUS_MULTIPLIER ** 2;
  const distance = (-linear - Math.sqrt(linear * linear - quadratic * constant)) / quadratic;
  return { x: (corner_x + distance * direction.x) * hex_r, y: (corner_y + distance * direction.y) * hex_r };
}

function edgeInnerApproachPoint(hex_r: number, from: Point2D): Point2D {
  const building = buildingLocalOffset(false, hex_r);
  const angle = Math.atan2(from.y, from.x - building.x);
  const dx = Math.cos(angle);
  const dy = Math.sin(angle);
  const boundary_distance = Math.min(
    (building.x - CENTER_ZONE_APOTHEM_MULTIPLIER * hex_r) / -dx,
    building.x / 2 / (-dx / 2 + (Math.sqrt(3) / 2) * Math.abs(dy))
  );
  const distance = (boundary_distance + BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r) / 2;
  return { x: building.x + distance * dx, y: distance * dy };
}

function closestEdgeApproachPoint(points: Point2D[], point: Point2D): Point2D {
  let closest = points[0];
  let closest_distance = Infinity;
  for (const candidate of points) {
    const distance = Math.hypot(candidate.x - point.x, candidate.y - point.y);
    if (distance < closest_distance) {
      closest = candidate;
      closest_distance = distance;
    }
  }
  return closest;
}

function buildEdgeUnitRows(
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

interface UnitSlotOffsetCache {
  hex_r: number;
  unit_r?: number;
  center: Map<number, Point2D[]>;
  edge_rows: Map<number, Point2D[][]>;
  edge: Map<number, Point2D[]>;
  rotated_edge: Map<number, Point2D[][]>;

  approach_outer: Point2D[];
}

// single entry, since hex_r only changes on resize; callers must treat the returned points as read-only
let unit_slot_offset_cache: UnitSlotOffsetCache | undefined;

function unitSlotOffsets(hex_r: number, unit_r?: number): UnitSlotOffsetCache {
  if (unit_slot_offset_cache?.hex_r !== hex_r) {
    const outer = edgeOuterApproachPoint(hex_r);
    unit_slot_offset_cache = {
      hex_r,
      center: new Map(),
      edge_rows: new Map(),
      edge: new Map(),
      rotated_edge: new Map(),

      approach_outer: [outer, { x: outer.x, y: -outer.y }],
    };
  }
  if (unit_r !== undefined && unit_slot_offset_cache.unit_r !== unit_r) {
    unit_slot_offset_cache.center.clear();
    unit_slot_offset_cache.edge_rows.clear();
    unit_slot_offset_cache.edge.clear();
    unit_slot_offset_cache.rotated_edge.clear();
    unit_slot_offset_cache.unit_r = unit_r;
  }
  return unit_slot_offset_cache;
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

function edgeOccupiedSlotOffsets(hex_r: number, count: number, unit_r: number): Point2D[] {
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

function unitSlotLocalOffsets(is_center: boolean, hex_r: number, count: number, unit_r: number): Point2D[] {
  const cache = unitSlotOffsets(hex_r, unit_r);
  const offsets = is_center ? cache.center : cache.edge;
  let positions = offsets.get(count);
  if (!positions) {
    positions = is_center
      ? centerUnitSlotLocalOffsets(hex_r, count, unit_r)
      : edgeOccupiedSlotOffsets(hex_r, count, unit_r);
    offsets.set(count, positions);
  }
  return positions;
}

export function zoneBuildingOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  const i = findOuterZoneIndex(zone_coordinate);
  const local = buildingLocalOffset(i === -1, hex_r);
  return i === -1 ? local : rotatePoint(local, (Math.PI / 3) * (i + 1));
}

export function zoneCenterOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  return zoneBuildingOffset(zone_coordinate, hex_r);
}

export function zoneUnitSlotLocalOffsets(
  zone_coordinate: Point2D,
  hex_r: number,
  center_count = 0,
  unit_r = hex_r
): Point2D[] {
  return unitSlotLocalOffsets(findOuterZoneIndex(zone_coordinate) === -1, hex_r, center_count, unit_r);
}

export function zoneBuildingLocalOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  return buildingLocalOffset(findOuterZoneIndex(zone_coordinate) === -1, hex_r);
}

export function zoneUnitSlotOffsets(
  zone_coordinate: Point2D,
  hex_r: number,
  center_count = 0,
  unit_r = hex_r
): Point2D[] {
  const i = findOuterZoneIndex(zone_coordinate);
  const positions = unitSlotLocalOffsets(i === -1, hex_r, center_count, unit_r);
  if (i === -1) {
    return positions;
  }
  const cache = unitSlotOffsets(hex_r, unit_r);
  let rotated = cache.rotated_edge.get(center_count);
  if (!rotated) {
    rotated = OUTER_ZONE_INDICES.map((_, direction) =>
      positions.map((p) => rotatePoint(p, (Math.PI / 3) * (direction + 1)))
    );
    cache.rotated_edge.set(center_count, rotated);
  }
  return rotated[i];
}

export const CENTER_ZONE_MERCENARY_SLOTS = 8;
export const EDGE_ZONE_MERCENARY_SLOTS = 6;

/** Pixel offsets from the space center of a zone's pending-mercenary slots, hugging its building circle */
export function zoneMercenarySlotOffsets(zone_coordinate: Point2D, hex_r: number): Point2D[] {
  const i = findOuterZoneIndex(zone_coordinate);
  const building = buildingLocalOffset(i === -1, hex_r);
  const unit_r = MERCENARY_SLOT_RADIUS_MULTIPLIER * hex_r;
  const ring_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r + MERCENARY_RING_OFFSET_MULTIPLIER * unit_r;
  return mercenarySlotAngles(i === -1, ring_r, unit_r).map((angle) => {
    const local = { x: building.x + ring_r * Math.cos(angle), y: building.y + ring_r * Math.sin(angle) };
    return i === -1 ? local : rotatePoint(local, (Math.PI / 3) * (i + 1));
  });
}

/** Angles around the building: a full ring for the center zone, a center-out inward arc for an edge zone */
function mercenarySlotAngles(is_center: boolean, ring_r: number, unit_r: number): number[] {
  if (is_center) {
    return Array.from(
      { length: CENTER_ZONE_MERCENARY_SLOTS },
      (_, k) => (2 * Math.PI * k) / CENTER_ZONE_MERCENARY_SLOTS
    );
  }
  const step = 2 * Math.asin((MERCENARY_SLOT_SPACING_MULTIPLIER * unit_r) / ring_r);
  return Array.from(
    { length: EDGE_ZONE_MERCENARY_SLOTS },
    (_, k) => Math.PI + (k % 2 === 0 ? 1 : -1) * (Math.floor(k / 2) + 0.5) * step
  );
}

export function getRisqZone(space: RisqSpace | undefined, zone_coordinate: Point2D): RisqZone | undefined {
  if (!space?.zones) {
    return undefined;
  }
  const i = findOuterZoneIndex(zone_coordinate);
  if (i === -1) {
    return space.zones[1][1];
  }
  const dv = OUTER_ZONE_INDICES[i];
  return space.zones[dv.x][dv.y];
}

/** Pixel offset to aim an order arrow at when its target is a whole zone rather than a specific unit/building */
export function zoneApproachPoint(zone_coordinate: Point2D, hex_r: number, from: Point2D): Point2D {
  const i = findOuterZoneIndex(zone_coordinate);
  if (i === -1) {
    const angle = Math.atan2(from.y, from.x);
    return {
      x: CENTER_UNIT_RING_RADIUS_MULTIPLIER * hex_r * Math.cos(angle),
      y: CENTER_UNIT_RING_RADIUS_MULTIPLIER * hex_r * Math.sin(angle),
    };
  }
  const rotation = (Math.PI / 3) * (i + 1);
  const local_from = rotatePoint(from, -rotation);
  const local_point =
    local_from.x >= EDGE_BUILDING_RADIAL_MULTIPLIER * hex_r
      ? closestEdgeApproachPoint(unitSlotOffsets(hex_r).approach_outer, local_from)
      : edgeInnerApproachPoint(hex_r, local_from);
  return rotatePoint(local_point, rotation);
}
