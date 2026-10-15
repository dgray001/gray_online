import type { Point2D } from '../../../../util/objects2d';
import {
  AXIAL_DIRECTION_VECTORS,
  addPoint2D,
  axialDistance,
  equalsPoint2D,
  subtractPoint2D,
} from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { RisqSelection } from '../../application/selection/selection';
import { cantorPair, invertPair, invertZoneKey } from '../../model/coordinates';
import { orderArrowColor } from '../../application/orders/orders_model';
import type { RisqOrderPlanning } from '../../application/orders/planning';
import type { DwgRisq } from '../../risq';
import {
  rangeDistance,
  RisqOrderType,
  RisqUnitType,
  type RisqFrontendOrder,
  type RisqMovePathStep,
  type RisqRange,
  type RisqSpace,
  type RisqUnit,
  type UnitByTypeData,
} from '../../model/types';
import { hexEdgeForDirectionIndex } from '../space_borders';
import { drawUnitTypeCluster } from '../assets/unit';
import { hexagonVertices, zoneCenterOffset, zoneMercenarySlotOffsets, zoneVertices } from '../zones/geometry';
import type { RisqViewport } from './viewport';

type RangeSubject = Pick<RisqUnit, 'space_coordinate' | 'attack_range'>;

function rangeSpaces(session: RisqSession, space: RisqSpace, radius: number): Map<number, Point2D> {
  const cached = space.range_spaces?.get(radius);
  if (cached) {
    return cached;
  }
  const spaces = new Map<number, Point2D>();
  for (let x = -radius; x <= radius; x++) {
    for (let y = -radius; y <= radius; y++) {
      const coordinate = addPoint2D(space.coordinate, { x, y });
      if (axialDistance(space.coordinate, coordinate) <= radius && session.spaceAt(coordinate)) {
        spaces.set(cantorPair(coordinate.x, coordinate.y), coordinate);
      }
    }
  }
  (space.range_spaces ??= new Map()).set(radius, spaces);
  return spaces;
}

function rangePaths(spaces: Map<number, Point2D>, viewport: RisqViewport): { fill: Path2D; outline: Path2D } {
  const fill = new Path2D();
  const outline = new Path2D();
  for (const coordinate of spaces.values()) {
    const center = viewport.coordinateToCanvas(coordinate);
    const vertices = hexagonVertices(viewport.hexR()).map((point: Point2D): Point2D => addPoint2D(center, point));
    fill.moveTo(vertices[0].x, vertices[0].y);
    for (const point of vertices.slice(1)) {
      fill.lineTo(point.x, point.y);
    }
    fill.closePath();
    for (const [direction, offset] of AXIAL_DIRECTION_VECTORS.entries()) {
      const neighbor = addPoint2D(coordinate, offset);
      if (!spaces.has(cantorPair(neighbor.x, neighbor.y))) {
        const [start, end] = hexEdgeForDirectionIndex(center, viewport.hexR(), direction);
        outline.moveTo(start.x, start.y);
        outline.lineTo(end.x, end.y);
      }
    }
  }
  return { fill, outline };
}

export function drawSelectionAttackRanges(
  ctx: CanvasRenderingContext2D,
  session: RisqSession,
  selection: RisqSelection,
  viewport: RisqViewport
): void {
  const units = [...selection.selectedUnitIds()].flatMap((id: number): RisqUnit[] => {
    const unit = session.findUnitById(id);
    return unit ? [unit] : [];
  });
  const subjects: RangeSubject[] = [
    ...selection.selectedBuildings(),
    ...units.flatMap((unit: RisqUnit): RangeSubject[] => {
      const location = session.unitLocation(unit);
      return location ? [{ space_coordinate: location.space_coordinate, attack_range: unit.attack_range }] : [];
    }),
  ];
  const mixed = new Set(subjects.map((subject: RangeSubject): RisqRange => subject.attack_range)).size > 1;
  const groups = new Map<RisqRange, Map<number, Point2D>>();
  const seen = new Set<string>();
  for (const subject of subjects) {
    const distance = rangeDistance(subject.attack_range);
    const space = session.spaceAt(subject.space_coordinate);
    const subject_key = `${subject.attack_range}:${space?.coordinate_key}`;
    if (distance === null || !space || seen.has(subject_key)) {
      continue;
    }
    seen.add(subject_key);
    const spaces = groups.get(subject.attack_range) ?? new Map<number, Point2D>();
    for (const [key, coordinate] of rangeSpaces(session, space, distance)) {
      spaces.set(key, coordinate);
    }
    groups.set(subject.attack_range, spaces);
  }
  ctx.save();
  ctx.setLineDash([]);
  ctx.globalAlpha = mixed ? 0.35 : 1;
  ctx.fillStyle = 'rgba(255, 255, 255, 0.12)';
  ctx.strokeStyle = 'white';
  ctx.lineWidth = 2 / viewport.lastTransform().scale;
  for (const spaces of groups.values()) {
    const paths = rangePaths(spaces, viewport);
    ctx.fill(paths.fill);
    ctx.stroke(paths.outline);
  }
  ctx.restore();
}

export function areaBoundary(vertices: readonly Point2D[], from: Point2D): { point: Point2D; inside: boolean } {
  const center = vertices.reduce((sum: Point2D, point: Point2D): Point2D => addPoint2D(sum, point), { x: 0, y: 0 });
  center.x /= vertices.length;
  center.y /= vertices.length;
  const offset = subtractPoint2D(from, center);
  const direction = offset.x || offset.y ? offset : { x: 1, y: 0 };
  let scale = Infinity;
  for (let i = 0; i < vertices.length; i++) {
    const start = vertices[i];
    const edge = subtractPoint2D(vertices[(i + 1) % vertices.length], start);
    const denominator = direction.x * edge.y - direction.y * edge.x;
    if (denominator === 0) {
      continue;
    }
    const relative = subtractPoint2D(start, center);
    const intersection = (relative.x * edge.y - relative.y * edge.x) / denominator;
    if (intersection > 0) {
      scale = Math.min(scale, intersection);
    }
  }
  return {
    point: { x: center.x + direction.x * scale, y: center.y + direction.y * scale },
    inside: (offset.x === 0 && offset.y === 0) || scale >= 1,
  };
}

export function drawAttackArea(
  ctx: CanvasRenderingContext2D,
  order: RisqFrontendOrder,
  from: Point2D,
  selected: boolean,
  path: RisqMovePathStep[],
  viewport: RisqViewport,
  tinted_attack_areas: Set<string>,
  stroke_segment: (ctx: CanvasRenderingContext2D, from: Point2D, to: Point2D) => void
): Point2D {
  const zone = order.order_type === RisqOrderType.OrderType_UnitAttackZone ? invertZoneKey(order.target_id) : undefined;
  const center = viewport.coordinateToCanvas(zone?.space ?? invertPair(order.target_id));
  const hex_r = viewport.hexR();
  const vertices = (zone ? zoneVertices(zone.zone, hex_r) : hexagonVertices(hex_r)).map(
    (point: Point2D): Point2D => addPoint2D(center, point)
  );
  ctx.save();
  ctx.strokeStyle = orderArrowColor(order.order_type);
  const key = `${order.order_type}:${order.target_id}`;
  if (selected && !tinted_attack_areas.has(key)) {
    ctx.beginPath();
    for (const point of vertices) {
      ctx.lineTo(point.x, point.y);
    }
    ctx.closePath();
    ctx.fillStyle = 'rgba(220, 30, 30, 0.18)';
    ctx.fill();
    tinted_attack_areas.add(key);
  }
  let current = from;
  for (const step of path) {
    const point = viewport.orderPoint(step.space, zoneCenterOffset(step.zone, hex_r), true);
    if (areaBoundary(vertices, point).inside) {
      break;
    }
    if (!equalsPoint2D(current, point)) {
      stroke_segment(ctx, current, point);
    }
    current = point;
  }
  const to = areaBoundary(vertices, current).point;
  if (!equalsPoint2D(current, to)) {
    stroke_segment(ctx, current, to);
  }
  ctx.restore();
  return to;
}

const GHOST_ALPHA = 0.6;

/** Collapses pending hires into at most `capacity` slots: one per hire, then one per unit type, then one for all */
function mercenaryGhostSlots(player_id: number, unit_ids: number[], capacity: number): UnitByTypeData[][] {
  const group = (unit_id: number, indices: number[]): UnitByTypeData => ({
    player_id,
    unit_id,
    unit_type: RisqUnitType.NONE,
    units: new Set(indices),
  });
  if (unit_ids.length <= capacity) {
    return unit_ids.map((unit_id, i) => [group(unit_id, [i])]);
  }
  const indices_by_type = new Map<number, number[]>();
  unit_ids.forEach((unit_id, i) => indices_by_type.set(unit_id, [...(indices_by_type.get(unit_id) ?? []), i]));
  const by_type = [...indices_by_type.entries()].map(([unit_id, indices]) => group(unit_id, indices));
  return by_type.length <= capacity ? by_type.map((g) => [g]) : [by_type];
}

function drawMercenaryGhosts(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  slots: UnitByTypeData[][],
  positions: Point2D[],
  unit_r: number,
  rotation: number
): void {
  ctx.globalAlpha = GHOST_ALPHA;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  for (const [i, slot] of slots.entries()) {
    ctx.save();
    ctx.translate(positions[i].x, positions[i].y);
    ctx.rotate(-rotation);
    const total = slot.reduce((sum, t) => sum + t.units.size, 0);
    drawUnitTypeCluster(ctx, risq, slot, { x: unit_r, y: unit_r }, total, 'white', 'rgba(210, 210, 210, 0.4)', true);
    ctx.restore();
  }
  ctx.globalAlpha = 1;
}

export function drawPendingMercenaries(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  session: RisqSession,
  viewport: RisqViewport,
  planning: RisqOrderPlanning
): void {
  const zone_view = viewport.zoneView();
  const hex_r = viewport.hexR();
  const groups = new Map<number, { positions: Point2D[]; unit_ids: number[] }>();
  for (const order of planning.pendingMercenaryOrders()) {
    const { x: unit_id, y: zone_key } = invertPair(order.target_id);
    const { space, zone } = invertZoneKey(zone_key);
    const key = zone_view ? zone_key : cantorPair(space.x, space.y);
    const space_center = viewport.coordinateToCanvas(space);
    const group = groups.get(key) ?? {
      positions: zoneMercenarySlotOffsets(zone_view ? zone : { x: 0, y: 0 }, hex_r).map((offset) =>
        addPoint2D(space_center, offset)
      ),
      unit_ids: [],
    };
    group.unit_ids.push(unit_id);
    groups.set(key, group);
  }
  const rotation = viewport.lastTransform().rotation;
  for (const { positions, unit_ids } of groups.values()) {
    const slots = mercenaryGhostSlots(session.getPlayerId(), unit_ids, positions.length);
    drawMercenaryGhosts(ctx, risq, slots, positions, viewport.unitRadius(), rotation);
  }
}
