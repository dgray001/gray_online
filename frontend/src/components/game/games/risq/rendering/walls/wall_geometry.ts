import type { Point2D } from '../../../../util/objects2d';
import { WALL_THICKNESS_SCALE } from './wall_metrics';
import type { WallDamage } from './wall_damage';

export declare interface WallSegmentInput {
  a: Point2D;
  b: Point2D;
  normal: Point2D;
  color: string;
  alpha: number;
  damage: WallDamage;
  face: number;
  garrison_icon?: CanvasImageSource;
  start_neighbor_normal?: Point2D;
  end_neighbor_normal?: Point2D;
}

export declare interface WallGeometry {
  outer: [Point2D, Point2D];
  parapet: [Point2D, Point2D];
  inner: [Point2D, Point2D];
  inward: Point2D;
}

function cross(p: Point2D, q: Point2D): number {
  return p.x * q.y - p.y * q.x;
}

function inwardShift(normal: Point2D, depth: number): Point2D {
  return { x: -normal.x * depth * WALL_THICKNESS_SCALE, y: -normal.y * depth * WALL_THICKNESS_SCALE };
}

function offsetCorner(vertex: Point2D, normal: Point2D, neighbor_normal: Point2D | undefined, depth: number): Point2D {
  const own = inwardShift(normal, depth);
  const simple = { x: vertex.x + own.x, y: vertex.y + own.y };
  if (!neighbor_normal) {
    return simple;
  }
  const other = inwardShift(neighbor_normal, depth);
  const own_dir = { x: -normal.y, y: normal.x };
  const other_dir = { x: -neighbor_normal.y, y: neighbor_normal.x };
  const denominator = cross(own_dir, other_dir);
  if (Math.abs(denominator) < 1e-6) {
    return simple;
  }
  const other_point = { x: vertex.x + other.x, y: vertex.y + other.y };
  const t = cross({ x: other_point.x - simple.x, y: other_point.y - simple.y }, other_dir) / denominator;
  return { x: simple.x + own_dir.x * t, y: simple.y + own_dir.y * t };
}

export function wallGeometry(segment: WallSegmentInput, parapet_depth: number, deck_depth: number): WallGeometry {
  const { a, b, normal } = segment;
  const line = (depth: number): [Point2D, Point2D] => [
    offsetCorner(a, normal, segment.start_neighbor_normal, depth),
    offsetCorner(b, normal, segment.end_neighbor_normal, depth),
  ];
  return {
    outer: [a, b],
    parapet: line(parapet_depth),
    inner: line(parapet_depth + deck_depth),
    inward: { x: -normal.x * WALL_THICKNESS_SCALE, y: -normal.y * WALL_THICKNESS_SCALE },
  };
}
