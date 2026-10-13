import type { Point2D } from '../../../util/objects2d';
import { AXIAL_DIRECTION_VECTORS, addPoint2D } from '../../../util/objects2d';
import { cantorPair, invertPair } from '../model/coordinates';
import type { RisqRegion } from '../model/types';

/** A label point and how many region spaces meet there: 3 at a corner, 2 on an edge, 1 inside a lone space */
export declare interface RegionLabelAnchor {
  point: Point2D;
  shared_spaces: number;
}

function averagePoint(points: Point2D[]): Point2D {
  const sum = points.reduce(addPoint2D);
  return { x: sum.x / points.length, y: sum.y / points.length };
}

function sharedPoints(region: RisqRegion, to_canvas: (coordinate: Point2D) => Point2D): RegionLabelAnchor[] {
  const keys = new Set(region.spaces);
  const in_region = (c: Point2D): boolean => keys.has(cantorPair(c.x, c.y));
  const anchors: RegionLabelAnchor[] = [];
  for (const space of region.spaces.map(invertPair)) {
    const neighbors = AXIAL_DIRECTION_VECTORS.map((direction) => addPoint2D(space, direction));
    for (const [i, neighbor] of neighbors.entries()) {
      const next = neighbors[(i + 1) % neighbors.length];
      if (in_region(neighbor)) {
        anchors.push({ point: averagePoint([space, neighbor].map((c) => to_canvas(c))), shared_spaces: 2 });
      }
      if (in_region(neighbor) && in_region(next)) {
        anchors.push({ point: averagePoint([space, neighbor, next].map((c) => to_canvas(c))), shared_spaces: 3 });
      }
    }
  }
  return anchors;
}

/** The anchor closest to `near`, preferring a shared corner, then a shared edge, then the nearest lone space */
export function regionLabelAnchor(
  region: RisqRegion,
  to_canvas: (coordinate: Point2D) => Point2D,
  near: Point2D
): RegionLabelAnchor {
  const anchors = sharedPoints(region, to_canvas);
  const most_shared = Math.max(1, ...anchors.map((anchor) => anchor.shared_spaces));
  const pool =
    most_shared > 1
      ? anchors.filter((anchor) => anchor.shared_spaces === most_shared)
      : region.spaces.map((key) => ({ point: to_canvas(invertPair(key)), shared_spaces: 1 }));
  const distance = (anchor: RegionLabelAnchor): number => Math.hypot(anchor.point.x - near.x, anchor.point.y - near.y);
  return pool.reduce((best, anchor) => (distance(anchor) < distance(best) ? anchor : best));
}
