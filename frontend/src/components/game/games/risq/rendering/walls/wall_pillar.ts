import type { Point2D } from '../../../../util/objects2d';
import type { WallMetrics } from './wall_metrics';
import { drawHangingBanner } from './wall_cloth';
import type { SourceRect } from './textured_quad';
import { drawTexturedQuad, drawTiledQuad, lightFor, pathPolygon } from './textured_quad';
import { drawRubbleStone, drawSmoke, seededRandom } from './wall_damage';

const TOP_COLOR = 'rgb(168, 158, 138)';
const TOP_EDGE = 'rgb(100, 95, 83)';
const CORNER_MERLON_SPAN = 0.32;

export declare interface PillarBreak {
  a_side: number;
  atlas: CanvasImageSource;
  ruin: boolean;
  weathered: boolean;
}

export declare interface PillarFootprint {
  corners: [Point2D, Point2D, Point2D, Point2D];
  normals: [Point2D, Point2D, Point2D, Point2D];
  color?: string;
  broken?: PillarBreak;
}

const BREAK_DROP = 0.45;
const RUIN_DROPS = [0.62, 0.82, 0.7, 0.55];
const WEATHERED_DROP = 1.12;
const NO_DROPS = [0, 0, 0, 0];

function lift(point: Point2D, z: number): Point2D {
  return { x: point.x, y: point.y - z };
}

function blend(c: PillarFootprint['corners'], a: number, b: number): Point2D {
  return {
    x: c[0].x + (c[1].x - c[0].x) * a + (c[3].x - c[0].x) * b,
    y: c[0].y + (c[1].y - c[0].y) * a + (c[3].y - c[0].y) * b,
  };
}

function fullRect(image: HTMLImageElement): SourceRect {
  return { x: 0, y: 0, w: image.naturalWidth, h: image.naturalHeight };
}

function drawShaft(
  ctx: CanvasRenderingContext2D,
  stone: HTMLImageElement,
  corners: Point2D[],
  normals: Point2D[],
  z0: number,
  z1: number,
  drops: number[] = NO_DROPS,
  scorched = false
): void {
  const source = fullRect(stone);
  for (let i = 0; i < 4; i++) {
    const p = corners[i];
    const q = corners[(i + 1) % 4];
    if (normals[i].y <= 0.02) {
      continue;
    }
    const height = z1 - z0;
    const [dp, dq] = [drops[i], drops[(i + 1) % 4]];
    const random = seededRandom(`shaft|${i}`);
    const rim = [1, 2, 3].map((k): Point2D => {
      const t = k / 4;
      const jag = dp > 0 || dq > 0 ? 0.2 * height * random() : 0;
      return { x: q.x + (p.x - q.x) * t, y: q.y + (p.y - q.y) * t - (z1 - (dq + (dp - dq) * t)) + jag };
    });
    const clip = [lift(p, z0), lift(q, z0), lift(q, z1 - dq), ...rim, lift(p, z1 - dp)];
    drawTiledQuad(
      ctx,
      stone,
      source,
      lift(p, z1),
      { x: q.x - p.x, y: q.y - p.y },
      { x: 0, y: height },
      (height * source.w) / source.h,
      clip,
      lightFor(normals[i])
    );
    if (scorched) {
      scorchFace(ctx, clip, lift(p, z1).y, lift(p, z0).y);
    }
  }
}

function scorchFace(ctx: CanvasRenderingContext2D, clip: Point2D[], top_y: number, bottom_y: number): void {
  ctx.save();
  pathPolygon(ctx, clip);
  ctx.clip();
  const soot = ctx.createLinearGradient(0, top_y, 0, bottom_y);
  soot.addColorStop(0, 'rgba(18, 12, 8, 0.7)');
  soot.addColorStop(0.7, 'rgba(18, 12, 8, 0.15)');
  soot.addColorStop(1, 'rgba(18, 12, 8, 0)');
  ctx.fillStyle = soot;
  const xs = clip.map((c) => c.x);
  ctx.fillRect(Math.min(...xs) - 2, top_y - 2, Math.max(...xs) - Math.min(...xs) + 4, bottom_y - top_y + 4);
  ctx.restore();
}

function widestVisibleFace(footprint: PillarFootprint): number {
  let best = -1;
  let best_width = 0;
  for (let i = 0; i < 4; i++) {
    const width = Math.abs(footprint.corners[(i + 1) % 4].x - footprint.corners[i].x);
    if (footprint.normals[i].y > 0.02 && width > best_width) {
      best = i;
      best_width = width;
    }
  }
  return best;
}

export function drawPillar(
  ctx: CanvasRenderingContext2D,
  stone: HTMLImageElement,
  m: WallMetrics,
  footprint: PillarFootprint
): void {
  const z1 = m.wall_h + m.merlon_h;
  const { corners, normals, broken } = footprint;
  const on_break = (i: number): boolean => !!broken && (i === 1 || i === 2) === (broken.a_side === 1);
  const worn = broken?.weathered ? WEATHERED_DROP : 1;
  const drops = corners.map((_, i) => (broken?.ruin ? RUIN_DROPS[i] * worn : on_break(i) ? BREAK_DROP : 0) * z1);
  drawShaft(ctx, stone, corners, normals, 0, z1, drops, !!broken && !broken.weathered);
  const face = widestVisibleFace(footprint);
  if (face >= 0 && footprint.color) {
    const p = corners[face];
    const q = corners[(face + 1) % 4];
    const span = Math.min(0.6 * m.period, 0.5 * Math.hypot(q.x - p.x, q.y - p.y));
    const length = Math.hypot(q.x - p.x, q.y - p.y);
    const half = span / length / 2;
    const lerp = (t: number): Point2D => ({ x: p.x + (q.x - p.x) * t, y: p.y + (q.y - p.y) * t });
    const left = lerp(0.5 - half);
    const right = lerp(0.5 + half);
    drawHangingBanner(
      ctx,
      footprint.color,
      lift(left, z1 - 0.25 * m.unit - Math.max(drops[face], drops[(face + 1) % 4])),
      { x: right.x - left.x, y: right.y - left.y },
      1.8 * m.unit,
      lightFor(normals[face])
    );
  }
  if (broken) {
    drawBrokenTop(ctx, stone, m, corners, drops, { z1, soot: !broken.weathered });
  } else {
    pathPolygon(
      ctx,
      corners.map((c) => lift(c, z1))
    );
    ctx.fillStyle = TOP_COLOR;
    ctx.fill();
    ctx.strokeStyle = TOP_EDGE;
    ctx.stroke();
  }
  drawCornerMerlons(ctx, stone, m, footprint, z1, drops);
  if (broken) {
    drawPillarDebris(ctx, m, footprint, z1 - Math.max(...drops));
  }
}

function drawBrokenTop(
  ctx: CanvasRenderingContext2D,
  stone: HTMLImageElement,
  m: WallMetrics,
  corners: Point2D[],
  drops: number[],
  { z1, soot }: { z1: number; soot: boolean }
): void {
  const random = seededRandom('pillar-top');
  const top = corners.map((c, i) => lift(c, z1 - drops[i]));
  const outline = top.flatMap((p, i) => {
    const next = top[(i + 1) % 4];
    if (drops[i] === 0 || drops[(i + 1) % 4] === 0) {
      return [p];
    }
    const jag = (t: number): Point2D => ({
      x: p.x + (next.x - p.x) * t,
      y: p.y + (next.y - p.y) * t - m.wall_h * 0.18 * random(),
    });
    return [p, jag(1 / 3), jag(2 / 3)];
  });
  const along = { x: top[1].x - top[0].x, y: top[1].y - top[0].y };
  const across = { x: top[3].x - top[0].x, y: top[3].y - top[0].y };
  const density = stone.naturalHeight / m.wall_h;
  const w = Math.min(stone.naturalWidth, Math.max(8, Math.hypot(along.x, along.y) * density));
  const h = Math.min(stone.naturalHeight, Math.max(8, Math.hypot(across.x, across.y) * density));
  const source = { x: random() * (stone.naturalWidth - w), y: random() * (stone.naturalHeight - h), w, h };
  drawTexturedQuad(ctx, stone, source, top[0], along, across, outline, 1);
  if (soot) {
    ctx.fillStyle = 'rgba(24, 18, 12, 0.3)';
    pathPolygon(ctx, outline);
    ctx.fill();
  }
}

function drawPillarDebris(
  ctx: CanvasRenderingContext2D,
  m: WallMetrics,
  { corners, broken }: PillarFootprint,
  top: number
): void {
  if (!broken) {
    return;
  }
  const random = seededRandom('pillar-debris');
  const outward = broken.a_side === 1 ? 1.5 : -0.5;
  for (let i = 0; i < (broken.weathered ? 8 : broken.ruin ? 14 : 7); i++) {
    const spread = broken.ruin ? -0.7 + 2.4 * random() : outward + 0.3 * random() * (broken.a_side === 1 ? 1 : -1);
    const at = blend(corners, spread, broken.ruin ? -0.5 + 2 * random() : random());
    const stone = {
      u: 0,
      across: 0,
      cell: Math.floor(random() * 8),
      size: m.wall_h * (0.4 + 0.3 * random()),
      mirrored: random() < 0.5,
    };
    drawRubbleStone(ctx, broken.atlas, at, stone);
  }
  for (let i = 0; i < 3; i++) {
    const on_top = lift(
      blend(corners, broken.a_side === 1 ? 0.55 + 0.3 * random() : 0.15 + 0.3 * random(), random()),
      top
    );
    drawRubbleStone(ctx, broken.atlas, on_top, {
      u: 0,
      across: 0,
      cell: Math.floor(random() * 8),
      size: 0.4 * m.wall_h,
      mirrored: random() < 0.5,
    });
  }
  if (broken.weathered) {
    return;
  }
  const mouth = blend(corners, broken.a_side === 1 ? 0.8 : 0.2, 0.5);
  drawSmoke(ctx, lift(mouth, top), 1.1 * m.unit, seededRandom('pillar-smoke'));
  if (broken.ruin) {
    drawSmoke(ctx, lift(blend(corners, 0.35, 0.7), top), 0.9 * m.unit, seededRandom('pillar-smoke-2'));
  }
}

function drawCornerMerlons(
  ctx: CanvasRenderingContext2D,
  stone: HTMLImageElement,
  m: WallMetrics,
  footprint: PillarFootprint,
  z1: number,
  drops: number[]
): void {
  const spans: [number, number][] = [
    [0, CORNER_MERLON_SPAN],
    [1 - CORNER_MERLON_SPAN, 1],
  ];
  const boxes: { points: Point2D[]; floor: number; height: number }[] = [];
  spans.forEach(([a0, a1], a_index) => {
    spans.forEach(([b0, b1], b_index) => {
      const on_break = footprint.broken && footprint.broken.a_side === a_index;
      boxes.push({
        points: [
          blend(footprint.corners, a0, b0),
          blend(footprint.corners, a1, b0),
          blend(footprint.corners, a1, b1),
          blend(footprint.corners, a0, b1),
        ],
        floor: on_break ? z1 - Math.max(...drops) : z1,
        height: footprint.broken?.ruin ? 0 : on_break ? (b_index === 0 ? 0.2 * m.unit : 0) : 0.5 * m.unit,
      });
    });
  });
  boxes.sort((p, q) => p.points[0].y + p.points[2].y - (q.points[0].y + q.points[2].y));
  for (const { points: box, floor, height } of boxes) {
    if (height === 0) {
      continue;
    }
    const top = floor + height;
    drawShaft(ctx, stone, box, footprint.normals, floor, top);
    pathPolygon(
      ctx,
      box.map((c) => lift(c, top))
    );
    ctx.fillStyle = TOP_COLOR;
    ctx.fill();
    ctx.strokeStyle = TOP_EDGE;
    ctx.stroke();
  }
}
