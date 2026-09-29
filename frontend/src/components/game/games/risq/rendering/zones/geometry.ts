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

/** Multiplier for inner zone relative to whole radius */
export const INNER_ZONE_MULTIPLIER = 0.4;

/** Radius multiplier (of hex_r) for a zone's building/resource circle; same for center and edge zones */
export const BUILDING_CIRCLE_RADIUS_MULTIPLIER = 0.13;

/** Number of generic unit slots around the center zone's building circle (one ring) */
export const CENTER_ZONE_UNIT_SLOTS = 12;
// apothem of the center zone's own hexagon (not the whole space)
export const CENTER_ZONE_APOTHEM_MULTIPLIER = (Math.sqrt(3) / 2) * INNER_ZONE_MULTIPLIER;
// midpoint between the building circle and the center zone's apothem, so both gaps match regardless of unit-circle size
const CENTER_UNIT_RING_RADIUS_MULTIPLIER = (CENTER_ZONE_APOTHEM_MULTIPLIER + BUILDING_CIRCLE_RADIUS_MULTIPLIER) / 2;
/** Radius multiplier (of hex_r) for one unit-slot circle; same for center and edge zones, sized to the center ring */
export const UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER =
  0.85 * CENTER_UNIT_RING_RADIUS_MULTIPLIER * Math.sin(Math.PI / CENTER_ZONE_UNIT_SLOTS);

/** Number of generic unit slots orbiting an edge zone's building, on its inward side */
export const EDGE_ZONE_UNIT_SLOTS = 8;
const EDGE_BUILDING_RADIAL_MULTIPLIER = 0.7; // angle 0 hits an edge midpoint, so the boundary is the apothem (~0.866), not 1.0

export function findOuterZoneIndex(zone_coordinate: Point2D): number {
  const index = coordinateToIndex(1, zone_coordinate);
  return OUTER_ZONE_INDICES.findIndex((dv) => equalsPoint2D(dv, index));
}

/** Clips ctx to one zone's shape (direction -1 for center), in the space's real on-screen frame (center c, radius r) */
export function clipToZone(ctx: CanvasRenderingContext2D, c: Point2D, r: number, zone_coordinate: Point2D) {
  const direction = findOuterZoneIndex(zone_coordinate);
  const inner_r = INNER_ZONE_MULTIPLIER * r;
  ctx.beginPath();
  if (direction === -1) {
    for (let i = 0; i < 6; i++) {
      const angle = (Math.PI / 3) * i + Math.PI / 6;
      ctx.lineTo(c.x + inner_r * Math.cos(angle), c.y + inner_r * Math.sin(angle));
    }
  } else {
    const a = (Math.PI / 3) * direction;
    ctx.lineTo(c.x + inner_r * Math.cos(a + Math.PI / 6), c.y + inner_r * Math.sin(a + Math.PI / 6));
    ctx.lineTo(c.x + inner_r * Math.cos(a + Math.PI / 2), c.y + inner_r * Math.sin(a + Math.PI / 2));
    ctx.lineTo(c.x + r * Math.cos(a + Math.PI / 2), c.y + r * Math.sin(a + Math.PI / 2));
    ctx.lineTo(c.x + r * Math.cos(a + Math.PI / 6), c.y + r * Math.sin(a + Math.PI / 6));
  }
  ctx.closePath();
  ctx.clip();
}

/** Local-frame (pre-space-rotation) offset of a zone's building/resource circle from its space's center */
export function buildingLocalOffset(is_center: boolean, hex_r: number): Point2D {
  return is_center ? { x: 0, y: 0 } : { x: EDGE_BUILDING_RADIAL_MULTIPLIER * hex_r, y: 0 };
}

/** Whether a space-center-relative point is within an edge zone's own footprint */
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

/** Binary-searches the distance from the building, along a given angle, to the zone's boundary */
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

/** Fill order: center-out interior pairs, then corners (each equidistant from building/boundary along its own angle) */
function edgeUnitSlotLocalOffsets(hex_r: number): Point2D[] {
  const building = buildingLocalOffset(false, hex_r);
  const corner = { x: hex_r * Math.cos(Math.PI / 6), y: hex_r * Math.sin(Math.PI / 6) };
  const corner_angle = Math.atan2(corner.y - building.y, corner.x - building.x);
  const sweep_step = (2 * (Math.PI - corner_angle)) / (EDGE_ZONE_UNIT_SLOTS - 1);
  const sweep_angle = (i: number) => corner_angle + i * sweep_step;
  const last_index = EDGE_ZONE_UNIT_SLOTS - 1;
  const center_low = (last_index - 1) / 2;
  const center_high = (last_index + 1) / 2;
  const interior_indices: number[] = [];
  for (let k = 0; center_low - k >= 1; k++) {
    interior_indices.push(center_low - k, center_high + k);
  }
  const interior_positions = interior_indices.map((i) => edgeSlotAtAngle(sweep_angle(i), hex_r));
  const spacing = Math.hypot(
    interior_positions[0].x - interior_positions[2].x,
    interior_positions[0].y - interior_positions[2].y
  );
  // on the corner's interior angle bisector, at `spacing` from the adjacent middle slot
  const corner_slot = (mirror: boolean): Point2D => {
    const q = mirror ? { x: corner.x, y: -corner.y } : corner;
    const m = interior_positions[mirror ? interior_positions.length - 1 : interior_positions.length - 2];
    const edge_a = { x: 0, y: mirror ? 1 : -1 }; // toward the other outer corner
    const edge_b = { x: -Math.cos(Math.PI / 6), y: (mirror ? 1 : -1) * Math.sin(Math.PI / 6) }; // toward the origin
    const bx = edge_a.x + edge_b.x;
    const by = edge_a.y + edge_b.y;
    const b_len = Math.hypot(bx, by);
    const dir = { x: bx / b_len, y: by / b_len };
    const vx = q.x - m.x;
    const vy = q.y - m.y;
    const v_dot_dir = vx * dir.x + vy * dir.y;
    const disc = Math.max(0, v_dot_dir * v_dot_dir - (vx * vx + vy * vy - spacing * spacing));
    const sqrt_disc = Math.sqrt(disc);
    const t1 = -v_dot_dir - sqrt_disc;
    const t = t1 >= 0 ? t1 : -v_dot_dir + sqrt_disc;
    return { x: q.x + t * dir.x, y: q.y + t * dir.y };
  };
  return [...interior_positions, corner_slot(false), corner_slot(true)];
}

function centerUnitSlotLocalOffsets(hex_r: number): Point2D[] {
  return Array.from({ length: CENTER_ZONE_UNIT_SLOTS }, (_, i) => {
    const angle = -((2 * Math.PI * i) / CENTER_ZONE_UNIT_SLOTS); // start right, go counterclockwise on screen
    return {
      x: CENTER_UNIT_RING_RADIUS_MULTIPLIER * hex_r * Math.cos(angle),
      y: CENTER_UNIT_RING_RADIUS_MULTIPLIER * hex_r * Math.sin(angle),
    };
  });
}

interface UnitSlotOffsetCache {
  hex_r: number;
  center: Point2D[];
  edge: Point2D[];
  rotated_edge: Point2D[][]; // by outer zone index
}

// single entry, since hex_r only changes on resize; callers must treat the returned points as read-only
let unit_slot_offset_cache: UnitSlotOffsetCache | undefined;

function unitSlotOffsets(hex_r: number): UnitSlotOffsetCache {
  if (unit_slot_offset_cache?.hex_r !== hex_r) {
    const edge = edgeUnitSlotLocalOffsets(hex_r);
    unit_slot_offset_cache = {
      hex_r,
      center: centerUnitSlotLocalOffsets(hex_r),
      edge,
      rotated_edge: OUTER_ZONE_INDICES.map((_, i) => edge.map((p) => rotatePoint(p, (Math.PI / 3) * (i + 1)))),
    };
  }
  return unit_slot_offset_cache;
}

/** Local-frame (pre-space-rotation) offsets of a zone's unit-slot circles from its space's center */
function unitSlotLocalOffsets(is_center: boolean, hex_r: number): Point2D[] {
  const cache = unitSlotOffsets(hex_r);
  return is_center ? cache.center : cache.edge;
}

/** Pixel offset of a zone's building/resource circle from its space's center, matching how drawRisqSpace positions it */
export function zoneBuildingOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  const i = findOuterZoneIndex(zone_coordinate);
  const local = buildingLocalOffset(i === -1, hex_r);
  return i === -1 ? local : rotatePoint(local, (Math.PI / 3) * (i + 1));
}

/** Pixel offset of a zone's center from its space's center, matching how drawSpaceContent positions it */
export function zoneCenterOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  return zoneBuildingOffset(zone_coordinate, hex_r);
}

/** Local-frame (pre-space-rotation) unit-slot offsets for a zone, for use by drawRisqSpace while already inside its own rotation */
export function zoneUnitSlotLocalOffsets(zone_coordinate: Point2D, hex_r: number): Point2D[] {
  return unitSlotLocalOffsets(findOuterZoneIndex(zone_coordinate) === -1, hex_r);
}

/** Local-frame (pre-space-rotation) building/resource offset for a zone, for use by drawRisqSpace while already inside its own rotation */
export function zoneBuildingLocalOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  return buildingLocalOffset(findOuterZoneIndex(zone_coordinate) === -1, hex_r);
}

/** Pixel offsets of all of a zone's unit-slot circles from its space's center */
export function zoneUnitSlotOffsets(zone_coordinate: Point2D, hex_r: number): Point2D[] {
  const i = findOuterZoneIndex(zone_coordinate);
  const cache = unitSlotOffsets(hex_r);
  return i === -1 ? cache.center : cache.rotated_edge[i];
}

export const CENTER_ZONE_MERCENARY_SLOTS = 8;
export const EDGE_ZONE_MERCENARY_SLOTS = 6;

/** Pixel offsets from the space center of a zone's pending-mercenary slots, hugging its building circle */
export function zoneMercenarySlotOffsets(zone_coordinate: Point2D, hex_r: number): Point2D[] {
  const i = findOuterZoneIndex(zone_coordinate);
  const building = buildingLocalOffset(i === -1, hex_r);
  const unit_r = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * hex_r;
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

/** Finds the zone object at the given coordinate within a space */
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
  const local_building = buildingLocalOffset(false, hex_r);
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const angle = Math.atan2(local_from.y - local_building.y, local_from.x - local_building.x);
  // shift so 0 deg sits perpendicular to the outward axis, making the inward-centered half exactly [0, 180)
  const deg = ((angle * 180) / Math.PI - 90 + 360) % 360;
  if (deg >= 180) {
    const slots = zoneUnitSlotOffsets(zone_coordinate, hex_r);
    return deg < 270 ? slots[5] : slots[4];
  }
  // short of the building itself, landing among the little circles instead
  const d = distanceToEdgeZoneBoundary(local_building, angle, hex_r);
  const r = (d + building_r) / 2;
  const local_point = { x: local_building.x + r * Math.cos(angle), y: local_building.y + r * Math.sin(angle) };
  return rotatePoint(local_point, rotation);
}
