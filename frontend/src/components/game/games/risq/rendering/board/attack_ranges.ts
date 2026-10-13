import type { Point2D } from '../../../../util/objects2d';
import { AXIAL_DIRECTION_VECTORS, addPoint2D, axialDistance } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { RisqSelection } from '../../application/selection/selection';
import { cantorPair } from '../../model/coordinates';
import { rangeDistance, type RisqRange, type RisqSpace, type RisqUnit } from '../../model/types';
import { hexEdgeForDirectionIndex } from '../space_borders';
import { hexagonVertices } from '../zones/geometry';
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
