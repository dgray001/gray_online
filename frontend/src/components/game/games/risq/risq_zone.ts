import { ColorRGB } from '../../../../scripts/color_rgb';
import { err } from '../../../../scripts/log';
import { atangent } from '../../../../scripts/math';
import { drawEllipse } from '../../util/canvas_util';
import type { Point2D } from '../../util/objects2d';
import { equalsPoint2D, pointInHexagon, rotatePoint, subtractPoint2D } from '../../util/objects2d';
import type { DwgRisq } from './risq';
import { buildingImage } from './risq_buildings';
import type { RisqSpace, RisqUnit, RisqZone, UnitByTypeData } from './risq_data';
import { RisqUnitType, RisqVisibilityLevel } from './risq_data';
import { resourceImage } from './risq_resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon, unitImage } from './risq_unit';
import { RisqViewMode, terrainImage } from './risq_terrain';
import { coordinateToIndex } from './risq_coordinates';

/** space.zones[][] array indices for the six outer zones, in draw-loop order (index i -> rotation (PI/3)*(i+1)) */
export const OUTER_ZONE_INDICES: Point2D[] = [
  { x: 2, y: 1 },
  { x: 2, y: 0 },
  { x: 1, y: 0 },
  { x: 0, y: 0 },
  { x: 0, y: 1 },
  { x: 1, y: 2 },
];

/** Multiplier for inner zone relative to whole radius */
export const INNER_ZONE_MULTIPLIER = 0.4;

/** Radius multiplier (of hex_r) for a zone's building/resource circle; same for center and edge zones */
export const BUILDING_CIRCLE_RADIUS_MULTIPLIER = 0.13;

/** Number of generic unit slots around the center zone's building circle (one ring) */
export const CENTER_ZONE_UNIT_SLOTS = 12;
// apothem of the center zone's own hexagon (not the whole space)
const CENTER_ZONE_APOTHEM_MULTIPLIER = (Math.sqrt(3) / 2) * INNER_ZONE_MULTIPLIER;
// midpoint between the building circle and the center zone's apothem, so both gaps match regardless of unit-circle size
const CENTER_UNIT_RING_RADIUS_MULTIPLIER = (CENTER_ZONE_APOTHEM_MULTIPLIER + BUILDING_CIRCLE_RADIUS_MULTIPLIER) / 2;
/** Radius multiplier (of hex_r) for one unit-slot circle; same for center and edge zones, sized to the center ring */
export const UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER =
  0.85 * CENTER_UNIT_RING_RADIUS_MULTIPLIER * Math.sin(Math.PI / CENTER_ZONE_UNIT_SLOTS);

/** Number of generic unit slots orbiting an edge zone's building, on its inward side */
export const EDGE_ZONE_UNIT_SLOTS = 8;
const EDGE_BUILDING_RADIAL_MULTIPLIER = 0.7; // angle 0 hits an edge midpoint, so the boundary is the apothem (~0.866), not 1.0

function findOuterZoneIndex(zone_coordinate: Point2D): number {
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

/** Returns the space's terrain image, or (cached, at the base terrain's own native resolution) a composite with any zone terrain overrides painted on top */
export function getSpaceTerrainImage(
  game: DwgRisq,
  base_terrain_id: number,
  space_zones: RisqZone[]
): CanvasImageSource {
  const base_icon = game.getIcon(terrainImage(base_terrain_id));
  const overrides = space_zones.filter((z) => !!z.terrain_override);
  if (overrides.length === 0) {
    return base_icon;
  }
  const key = `space_terrain:${base_terrain_id}:${overrides
    .map((z) => `${findOuterZoneIndex(z.coordinate)}=${z.terrain_override}`)
    .sort()
    .join(',')}`;
  const w = base_icon.naturalWidth;
  const h = base_icon.naturalHeight;
  const c = { x: w / 2, y: h / 2 };
  const r = h / 2;
  const override_icons = overrides.map((z) => game.getIcon(terrainImage(z.terrain_override)));
  const composite = game.getImageCache().getRectImage(key, w, h, [base_icon, ...override_icons], (ctx) => {
    ctx.drawImage(base_icon, 0, 0, w, h);
    overrides.forEach((zone, i) => {
      ctx.save();
      clipToZone(ctx, c, r, zone.coordinate);
      ctx.drawImage(override_icons[i], 0, 0, w, h);
      ctx.restore();
    });
  });
  return composite ?? base_icon;
}

/** Local-frame (pre-space-rotation) offset of a zone's building/resource circle from its space's center */
function buildingLocalOffset(is_center: boolean, hex_r: number): Point2D {
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

/** Fill order: center-out interior pairs, then corners (each equidistant from building/boundary along its own angle) */
function edgeUnitSlotLocalOffsets(hex_r: number): Point2D[] {
  const building = buildingLocalOffset(false, hex_r);
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const corner = { x: hex_r * Math.cos(Math.PI / 6), y: hex_r * Math.sin(Math.PI / 6) };
  const corner_angle = Math.atan2(corner.y - building.y, corner.x - building.x);
  const sweep_step = (2 * (Math.PI - corner_angle)) / (EDGE_ZONE_UNIT_SLOTS - 1);
  const sweep_angle = (i: number) => corner_angle + i * sweep_step;
  const slot_at_angle = (angle: number): Point2D => {
    const d = distanceToEdgeZoneBoundary(building, angle, hex_r);
    const r = (d + building_r) / 2;
    return { x: building.x + r * Math.cos(angle), y: building.y + r * Math.sin(angle) };
  };
  const last_index = EDGE_ZONE_UNIT_SLOTS - 1;
  const center_low = (last_index - 1) / 2;
  const center_high = (last_index + 1) / 2;
  const interior_indices: number[] = [];
  for (let k = 0; center_low - k >= 1; k++) {
    interior_indices.push(center_low - k, center_high + k);
  }
  const interior_positions = interior_indices.map((i) => slot_at_angle(sweep_angle(i)));
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

/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(units: Map<number, RisqUnit>): Map<number, Map<number, UnitByTypeData>> {
  const units_by_type = new Map<number, Map<number, UnitByTypeData>>();
  for (const unit of units.values()) {
    if (!units_by_type.has(unit.player_id)) {
      units_by_type.set(unit.player_id, new Map<number, UnitByTypeData>());
    }
    const units_by_type_data = units_by_type.get(unit.player_id)?.get(unit.unit_id);
    if (units_by_type_data) {
      units_by_type_data.units.add(unit.internal_id);
    } else {
      units_by_type.get(unit.player_id)!.set(unit.unit_id, {
        unit_id: unit.unit_id,
        unit_type: unit.unit_type,
        player_id: unit.player_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
  return units_by_type;
}

/** Groups the input units by player, keeping only economic or military types */
export function unitsByPlayerFiltered(units: Map<number, RisqUnit>, economic: boolean): Map<number, UnitByTypeData[]> {
  const units_by_type = organizeZoneUnits(units);
  const result = new Map<number, UnitByTypeData[]>();
  for (const [player_id, player_units] of units_by_type.entries()) {
    const filtered = [...player_units.values()].filter((u) => (u.unit_type === RisqUnitType.ECONOMIC) === economic);
    if (filtered.length > 0) {
      result.set(player_id, filtered);
    }
  }
  return result;
}

/** Groups an arbitrary, possibly multi-space list of a single player's unit ids by unit type */
export function groupUnitsByType(units: Map<number, RisqUnit>, ids: number[]): UnitByTypeData[] {
  const by_type = new Map<number, UnitByTypeData>();
  for (const id of ids) {
    const unit = units.get(id);
    if (!unit) {
      continue;
    }
    const existing = by_type.get(unit.unit_id);
    if (existing) {
      existing.units.add(id);
    } else {
      by_type.set(unit.unit_id, {
        player_id: unit.player_id,
        unit_id: unit.unit_id,
        unit_type: unit.unit_type,
        units: new Set([id]),
      });
    }
  }
  return [...by_type.values()];
}

export const UNIT_CLUSTER_ICON_SIZE = 64;

/** Cache key for a multi-unit-type cluster icon, sensitive to the types, their counts, and the shown total */
export function unitClusterIconKey(units_by_type: UnitByTypeData[], total: number): string {
  return 'unit_cluster:' + units_by_type.map((t) => `${t.unit_id}x${t.units.size}`).join(',') + `:${total}`;
}

/** Draws one player's unit types centered at the origin, arranging 1/2/3/4/many types each with its count */
export function drawUnitTypeCluster(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  units_by_type: UnitByTypeData[],
  r: Point2D,
  total: number,
  primary_color: string,
  secondary_color: string
) {
  const draw_count = (s: string, ts: number, x: number, y: number, w: number, fill_primary = true) => {
    const fs = ctx.fillStyle;
    ctx.fillStyle = fill_primary ? primary_color : secondary_color;
    ctx.font = `bold ${ts}px serif`;
    ctx.fillText(s, x, y, w);
    ctx.fillStyle = fs;
  };
  const icon = (t: UnitByTypeData) => {
    const color = game.getGame()?.players[t.player_id]?.color;
    return color ? game.getPlayerColoredIcon(unitImage(t.unit_id), color) : game.getIcon(unitImage(t.unit_id));
  };
  if (units_by_type.length === 1) {
    const t = units_by_type[0];
    ctx.drawImage(icon(t), -r.x, -r.y, 2 * r.x, 2 * r.y);
    draw_count(t.units.size.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  } else if (units_by_type.length === 2) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 1.5 * r.x, 1.5 * r.y);
      draw_count(t.units.size.toString(), r.y, (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 2 * r.x);
    }
  } else if (units_by_type.length === 3) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.9 * r.y, 1.5 * r.x);
    }
    const t = units_by_type[2];
    ctx.drawImage(icon(t), -0.5 * r.x, -0.1 * r.y, r.x, r.y);
    draw_count(t.units.size.toString(), 0.75 * r.y, -0.5 * r.x, -0.1 * r.y, 1.5 * r.x);
  } else if (units_by_type.length === 4) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.9 * r.y, 1.5 * r.x);
    }
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[2 + j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.1 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.1 * r.y, 1.5 * r.x);
    }
  } else {
    for (let j = 0; j < 2; j++) {
      ctx.drawImage(icon(units_by_type[j]), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
    }
    ctx.drawImage(icon(units_by_type[2]), -0.9 * r.x, -0.1 * r.y, r.x, r.y);
    draw_count('...', 0.8 * r.y, 0.1 * r.x, -0.1 * r.y, r.x, false);
    draw_count(total.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  }
}

/** Gets zone fill for the input zone */
export function getZoneFill(
  zone: RisqZone,
  view_mode: RisqViewMode = RisqViewMode.ALL,
  owner_color: ColorRGB | undefined = undefined,
  check_hover = true,
  alpha_multiplier = 1
): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (view_mode === RisqViewMode.OWNERSHIP) {
    if (owner_color) {
      color.setColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), 0.85);
    } else {
      color.setColor(90, 90, 90, 0.85);
    }
  } else if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
    color.addColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), alpha_multiplier * 0.06);
  }
  if (check_hover && zone.hovered && !zone.hovered_data.some((p) => p.hovered)) {
    if (zone.clicked) {
      color.addColor(210, 210, 210, alpha_multiplier * 0.06);
    } else {
      color.addColor(190, 190, 190, alpha_multiplier * 0.03);
    }
  }
  return color;
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

function unitVisibleInViewMode(unit_type: RisqUnitType, view_mode: RisqViewMode): boolean {
  if (view_mode === RisqViewMode.OWNERSHIP) {
    return false;
  }
  if (view_mode === RisqViewMode.MILITARY) {
    return unit_type !== RisqUnitType.ECONOMIC;
  }
  if (view_mode === RisqViewMode.RESOURCE) {
    return unit_type === RisqUnitType.ECONOMIC;
  }
  return true;
}

/** Draws the input risq zone */
export function drawRisqZone(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  zone: RisqZone,
  visibility: number,
  view_mode: RisqViewMode,
  black_text: boolean,
  hex_r: number,
  rotation: number,
  building_pos: Point2D,
  unit_slot_positions: Point2D[],
  active_player_id: number
) {
  const primary_color = black_text ? 'rgb(0, 0, 0)' : 'rgb(255, 255, 255)';
  const secondary_color = black_text ? 'rgba(40, 40, 40, 0.4)' : 'rgba(210, 210, 210, 0.4)';
  const tertiary_color = black_text ? 'rgb(60, 60, 60, 0.2)' : 'rgba(190, 190, 190, 0.2)';

  function drawText(
    ctx: CanvasRenderingContext2D,
    s: string,
    ts: number,
    x: number,
    y: number,
    w: number,
    fill_primary = true
  ) {
    const fs = ctx.fillStyle;
    ctx.fillStyle = fill_primary ? primary_color : secondary_color;
    ctx.font = `bold ${ts}px serif`;
    ctx.fillText(s, x, y, w);
    ctx.fillStyle = fs;
  }

  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const unit_r = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const filled_slots = buildZoneUnitSlots(zone, active_player_id, unit_slot_positions.length);
  zone.unit_slots = filled_slots;

  const target_len = 1 + filled_slots.length;
  if (zone.hovered_data.length !== target_len || zone.reset_hovered_data) {
    zone.hovered_data = [
      { c: building_pos, r: { x: building_r, y: building_r } },
      ...filled_slots.map((_, i) => ({ c: unit_slot_positions[i], r: { x: unit_r, y: unit_r } })),
    ];
    zone.reset_hovered_data = false;
  } else {
    zone.hovered_data[0].c = building_pos;
    for (let i = 0; i < filled_slots.length; i++) {
      zone.hovered_data[i + 1].c = unit_slot_positions[i];
    }
  }
  for (const [i, part] of zone.hovered_data.entries()) {
    const selected =
      i === 0
        ? (!!zone.resource && game.isResourceSelected(zone.resource.internal_id)) ||
          (!!zone.building && game.isBuildingSelected(zone.building.internal_id))
        : filled_slots[i - 1].some((t) => [...t.units].some((uid) => game.isUnitSelected(uid)));
    ctx.strokeStyle = 'transparent';
    if (part.hovered) {
      if (part.clicked) {
        ctx.fillStyle = secondary_color;
      } else {
        ctx.fillStyle = tertiary_color;
      }
    } else {
      ctx.fillStyle = 'transparent';
    }
    ctx.translate(part.c.x, part.c.y);
    ctx.rotate(-rotation);
    if (i === 0) {
      // resources / building
      if (!!zone.resource && view_mode !== RisqViewMode.MILITARY && view_mode !== RisqViewMode.OWNERSHIP) {
        ctx.drawImage(game.getIcon(resourceImage(zone.resource)), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
      } else {
        let building_image: string;
        let building_color: ColorRGB | undefined;

        const local_foundation = game.getLocalFoundation(zone.coordinate_key);
        const server_foundation = game.getPlayer()?.planned_foundations?.get(zone.coordinate_key);

        if (zone.building) {
          building_image = buildingImage(zone.building.building_id, zone.building.under_construction);
          building_color = game.getGame()?.players[zone.building.player_id]?.color;
        } else if (local_foundation || server_foundation) {
          building_image = 'risq/buildings/construction';
          building_color = game.getPlayer()?.color;
        } else {
          building_image = buildingImage(undefined);
          building_color = undefined;
        }

        const building_icon = building_color
          ? game.getPlayerColoredIcon(building_image, building_color)
          : game.getIcon(building_image);
        ctx.drawImage(building_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        if (!!zone.building && zone.building.has_garrisoned_units) {
          const flag_icon = building_color
            ? game.getPlayerColoredIcon('risq/icons/garrison_flag', building_color)
            : game.getIcon('risq/icons/garrison_flag');
          const flag_size = 0.9 * part.r.x;
          ctx.drawImage(flag_icon, 0.5 * part.r.x, -1.4 * part.r.y, flag_size, flag_size);
        }
      }
    } else if (visibility === RisqVisibilityLevel.POOR) {
      if (i === 1 && !!zone.unit_count && view_mode !== RisqViewMode.OWNERSHIP) {
        const villager_img = game.getIcon('icons/villager64');
        const unit_img = game.getIcon('icons/unit64');
        const combo_icon = game
          .getImageCache()
          .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
            drawComboUnitIcon(combo_ctx, villager_img, unit_img)
          );
        if (combo_icon) {
          ctx.drawImage(combo_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        }
        drawText(ctx, zone.unit_count.toString(), 1.4 * part.r.y, -part.r.x, -0.7 * part.r.y, 2 * part.r.x);
      } else {
        ctx.strokeStyle = secondary_color;
      }
    } else {
      const slot = filled_slots[i - 1].filter((t) => unitVisibleInViewMode(t.unit_type, view_mode));
      if (slot.length === 0) {
        ctx.strokeStyle = secondary_color;
      } else {
        const slot_total = slot.reduce((sum, t) => sum + t.units.size, 0);
        drawUnitTypeCluster(ctx, game, slot, part.r, slot_total, primary_color, secondary_color);
      }
    }
    ctx.rotate(rotation);
    ctx.translate(-part.c.x, -part.c.y);
    if (selected) {
      const prev_line_width = ctx.lineWidth;
      ctx.strokeStyle = 'white';
      ctx.lineWidth = 0.4;
      drawEllipse(ctx, part.c, part.r);
      ctx.lineWidth = prev_line_width;
    } else {
      drawEllipse(ctx, part.c, part.r);
    }
  }
}

/** Resolves hover logic for the zones of a risq space */
export function resolveHoveredZones(
  p: Point2D,
  space: RisqSpace | undefined,
  r: number,
  override_center?: Point2D,
  ignore_parts?: boolean
): RisqZone | undefined {
  if (!space?.zones) {
    return undefined;
  }

  const resolve_zone_dependencies = (m: Point2D, zone: RisqZone, rotate: number) => {
    zone.hovered = true;
    if (ignore_parts) {
      return;
    }
    const p = rotatePoint(m, rotate);
    for (const part of zone.hovered_data) {
      const dx = p.x - part.c.x;
      const dy = p.y - part.c.y;
      if ((dx * dx) / (part.r.x * part.r.x) + (dy * dy) / (part.r.y * part.r.y) <= 1) {
        part.hovered = true;
      } else {
        part.hovered = false;
      }
    }
  };

  const m = subtractPoint2D({ x: p.x, y: p.y }, override_center ?? space.center);
  let new_hovered_zone: RisqZone | undefined = undefined;
  if (pointInHexagon(m, INNER_ZONE_MULTIPLIER * r)) {
    new_hovered_zone = space.zones[1][1];
    resolve_zone_dependencies(m, new_hovered_zone, 0);
  } else if (pointInHexagon(m, r)) {
    const angle = atangent(m.y, m.x);
    let index = Math.floor((angle + Math.PI / 6) / (Math.PI / 3));
    let direction_vector: Point2D = { x: 0, y: 0 };
    switch (index) {
      case 6:
        index = 0;
      case 0:
        direction_vector = { x: 1, y: 2 };
        break;
      case 1:
        direction_vector = { x: 0, y: 1 };
        break;
      case 2:
        direction_vector = { x: 0, y: 0 };
        break;
      case 3:
        direction_vector = { x: 1, y: 0 };
        break;
      case 4:
        direction_vector = { x: 2, y: 0 };
        break;
      case 5:
        direction_vector = { x: 2, y: 1 };
        break;
      default:
        err('Unknown zone hovered', angle);
        return;
    }
    new_hovered_zone = space.zones[direction_vector.x][direction_vector.y];
    resolve_zone_dependencies(m, new_hovered_zone, -(Math.PI / 3) * (1 + 5 - index));
  }
  return new_hovered_zone;
}

export type HoveredZoneObject =
  | { kind: 'building' }
  | { kind: 'resource' }
  | { kind: 'unit'; groups: UnitByTypeData[] };

/** Returns whatever slot (building, resource, or unit group) is under the cursor in this zone, if any */
export function hoveredZoneObject(zone: RisqZone): HoveredZoneObject | undefined {
  for (const [i, part] of zone.hovered_data.entries()) {
    if (!part.hovered) {
      continue;
    }
    if (i === 0) {
      if (zone.building) {
        return { kind: 'building' };
      }
      if (zone.resource) {
        return { kind: 'resource' };
      }
      return undefined;
    }
    const groups = zone.unit_slots?.[i - 1];
    if (groups) {
      return { kind: 'unit', groups };
    }
  }
  return undefined;
}

/** Removes all hovered flags from the risq zone */
export function unhoverRisqZone(zone: RisqZone) {
  if (!zone) {
    return;
  }
  zone.clicked = false;
  zone.hovered = false;
  for (const part of zone.hovered_data) {
    part.clicked = false;
    part.hovered = false;
  }
}
