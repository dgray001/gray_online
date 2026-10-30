import type { Point2D } from '../../../../util/objects2d';
import type { WallMetrics } from './wall_metrics';
import { HEIGHT_SCALE, WALL_MERLONS } from './wall_metrics';
import { pathPolygon } from './textured_quad';

const WOOD = 'rgb(110, 82, 52)';
const WOOD_EDGE = 'rgb(52, 38, 26)';
const BRACKET_EVERY = 4;

function lift(point: Point2D, z: number): Point2D {
  return { x: point.x, y: point.y - z };
}

function along(a: Point2D, b: Point2D, u: number): Point2D {
  return { x: a.x + (b.x - a.x) * u, y: a.y + (b.y - a.y) * u };
}

export function drawSupports(
  ctx: CanvasRenderingContext2D,
  m: WallMetrics,
  [a, b]: [Point2D, Point2D],
  [from, to]: [number, number]
): void {
  const { wall_h: h, period } = m;
  const tall = m.unit * HEIGHT_SCALE;
  const beam_h = 0.18 * tall;
  pathPolygon(ctx, [
    lift(along(a, b, from), h - beam_h),
    lift(along(a, b, to), h - beam_h),
    lift(along(a, b, to), h),
    lift(along(a, b, from), h),
  ]);
  ctx.fillStyle = WOOD;
  ctx.fill();
  const length = Math.hypot(b.x - a.x, b.y - a.y);
  const unit = { x: (b.x - a.x) / length, y: (b.y - a.y) / length };
  const post = (center: Point2D, w: number, z_top: number, z_bottom: number, narrow: number): void => {
    pathPolygon(ctx, [
      lift({ x: center.x - unit.x * w, y: center.y - unit.y * w }, z_top),
      lift({ x: center.x + unit.x * w, y: center.y + unit.y * w }, z_top),
      lift({ x: center.x + unit.x * w * narrow, y: center.y + unit.y * w * narrow }, z_bottom),
      lift({ x: center.x - unit.x * w * narrow, y: center.y - unit.y * w * narrow }, z_bottom),
    ]);
    ctx.fillStyle = WOOD;
    ctx.fill();
    ctx.strokeStyle = WOOD_EDGE;
    ctx.stroke();
  };
  for (let i = 0; i < WALL_MERLONS; i++) {
    const u = (i + 0.5) / WALL_MERLONS;
    if (u < from || u > to) {
      continue;
    }
    const center = along(a, b, u);
    post(center, 0.26 * period, h - beam_h, h - beam_h - 0.6 * tall, 0.15);
    if (i % BRACKET_EVERY === 0) {
      post(center, 0.2 * period, h, 0, 1);
    }
  }
}
