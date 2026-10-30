import type { Point2D } from '../../../../util/objects2d';
import { drawRubbleStone, drawSmoke, seededRandom } from './wall_damage';

const SMOKE_SCALES = [1.7, 1.2, 1];
export const [WEATHERED_HEIGHT, WEATHERED_LIGHT] = [0.4, 1.15];
export const RUIN_COUNTS = { beams: 5, surface: 7, ground: 22, smoke: 3 };
export const WEATHERED_COUNTS = { beams: 0, surface: 16, ground: 12, smoke: 0 };

export declare interface RuinPoints {
  beams: Point2D[];
  surface: Point2D[];
  ground: Point2D[];
  smoke: Point2D[];
}

function strokeBeam(ctx: CanvasRenderingContext2D, from: Point2D, to: Point2D, width: number, color: string): void {
  ctx.strokeStyle = color;
  ctx.lineWidth = Math.max(1, width);
  ctx.beginPath();
  ctx.moveTo(from.x, from.y);
  ctx.lineTo(to.x, to.y);
  ctx.stroke();
}

export function drawRafters(
  ctx: CanvasRenderingContext2D,
  r: number,
  apex: Point2D,
  bases: Point2D[],
  rings: Point2D[][]
): void {
  ctx.lineCap = 'round';
  for (const base of bases) {
    strokeBeam(ctx, base, apex, 0.01 * r, 'rgb(48, 38, 30)');
    strokeBeam(
      ctx,
      { x: base.x - 0.003 * r, y: base.y },
      { x: apex.x - 0.003 * r, y: apex.y },
      0.003 * r,
      'rgba(170, 150, 125, 0.6)'
    );
  }
  for (const ring of rings) {
    ctx.beginPath();
    ring.forEach((p, i) => (i === 0 ? ctx.moveTo(p.x, p.y) : ctx.lineTo(p.x, p.y)));
    ctx.closePath();
    ctx.strokeStyle = 'rgb(62, 50, 40)';
    ctx.lineWidth = Math.max(1, 0.012 * r);
    ctx.stroke();
  }
}

export function drawRuinProps(
  ctx: CanvasRenderingContext2D,
  atlas: CanvasImageSource,
  r: number,
  smoke_size: number,
  points: RuinPoints
): void {
  const random = seededRandom('tower-ruin-props');
  ctx.lineCap = 'round';
  for (const from of points.beams) {
    const to = { x: from.x + (random() - 0.5) * 0.09 * r, y: from.y - (0.04 + 0.06 * random()) * r };
    strokeBeam(ctx, from, to, 0.012 * r, 'rgb(40, 31, 24)');
    strokeBeam(ctx, from, to, 0.003 * r, 'rgba(150, 130, 108, 0.55)');
  }
  const stones = [
    ...points.surface.map((at) => ({ at, size: 0.05 + 0.025 * random() })),
    ...points.ground.map((at) => ({ at, size: 0.05 + 0.05 * random() })),
  ].sort((p, q) => p.at.y - q.at.y);
  for (const { at, size } of stones) {
    const stone = { u: 0, across: 0, cell: Math.floor(random() * 8), size: size * r, mirrored: random() < 0.5 };
    drawRubbleStone(ctx, atlas, at, stone);
  }
  points.smoke.forEach((at, i) => drawSmoke(ctx, at, smoke_size * SMOKE_SCALES[i], seededRandom(`ruin-smoke-${i}`)));
}
