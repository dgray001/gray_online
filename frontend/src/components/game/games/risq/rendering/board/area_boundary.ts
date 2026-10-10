import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, subtractPoint2D } from '../../../../util/objects2d';

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
