import type { Point2D } from '../../../../util/objects2d';
import { lightFor } from './textured_quad';
import { drawFlag, drawHangingBanner } from './wall_cloth';
import type { WallDamage } from './wall_damage';
import { isRuin, seededRandom } from './wall_damage';
import type { WallMetrics } from './wall_metrics';
import { FLAG_SCALE, HEIGHT_SCALE, WALL_MERLONS } from './wall_metrics';

const BANNER_AT = [2, 6, 10, 14, 18];
const FLAG_AT = [0.2, 0.8];
const TORN_CHANCE = [0, 0, 0.4, 0.85];

export declare interface DecorContext {
  ctx: CanvasRenderingContext2D;
  m: WallMetrics;
  color: string;
  damage: WallDamage;
  span: [number, number];
  seed: string;
}

function along(a: Point2D, b: Point2D, u: number, shift?: Point2D): Point2D {
  return { x: a.x + (b.x - a.x) * u + (shift?.x ?? 0), y: a.y + (b.y - a.y) * u + (shift?.y ?? 0) };
}

function lift(point: Point2D, z: number): Point2D {
  return { x: point.x, y: point.y - z };
}

function inSpan(span: [number, number], u: number, half = 0): boolean {
  return u - half >= span[0] - 1e-9 && u + half <= span[1] + 1e-9;
}

export function drawWallBanners(c: DecorContext, [a, b]: [Point2D, Point2D], normal: Point2D): void {
  if (isRuin(c.damage)) {
    return;
  }
  const { wall_h: h, unit } = c.m;
  const torn = seededRandom(`${c.seed}|banners`);
  for (const i of BANNER_AT) {
    const torn_off = c.damage >= 2 && torn() < TORN_CHANCE[c.damage];
    if (torn_off || !inSpan(c.span, i / WALL_MERLONS, 0.25 / WALL_MERLONS)) {
      continue;
    }
    const left = along(a, b, (i - 0.25) / WALL_MERLONS);
    const right = along(a, b, (i + 0.25) / WALL_MERLONS);
    drawHangingBanner(
      c.ctx,
      c.color,
      lift(left, h - 0.12 * unit),
      { x: right.x - left.x, y: right.y - left.y },
      (1.6 - 0.35 * Math.min(c.damage, 2)) * unit * HEIGHT_SCALE,
      lightFor(normal)
    );
  }
}

export function drawWallFlags(c: DecorContext, [a, b]: [Point2D, Point2D], inward: Point2D): void {
  if (isRuin(c.damage)) {
    return;
  }
  const { wall_h: h, merlon_h: mh, crenel_h: cren, unit } = c.m;
  const k = FLAG_SCALE * unit;
  const shift = { x: inward.x * c.m.parapet_depth * 0.5, y: inward.y * c.m.parapet_depth * 0.5 };
  for (const u of FLAG_AT.filter((at) => inSpan(c.span, at))) {
    drawFlag(c.ctx, c.color, lift(along(a, b, u, shift), h + cren), mh - cren + 1.4 * k, 1.5 * k, 0.9 * k, 0.12 * k);
  }
}
