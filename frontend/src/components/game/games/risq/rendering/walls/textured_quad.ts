import type { Point2D } from '../../../../util/objects2d';

export declare interface SourceRect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export function pathPolygon(ctx: CanvasRenderingContext2D, points: Point2D[]): void {
  ctx.beginPath();
  ctx.moveTo(points[0].x, points[0].y);
  for (const point of points.slice(1)) {
    ctx.lineTo(point.x, point.y);
  }
  ctx.closePath();
}

export function lightFor(normal: Point2D): number {
  return 0.8 + 0.22 * normal.y - 0.08 * Math.abs(normal.x);
}

function shadeFill(ctx: CanvasRenderingContext2D, light: number): void {
  if (light < 1) {
    ctx.fillStyle = `rgba(0, 0, 0, ${1 - light})`;
    ctx.fill();
  } else if (light > 1) {
    ctx.fillStyle = `rgba(255, 255, 255, ${Math.min(1, (light - 1) * 2)})`;
    ctx.fill();
  }
}

export function drawTexturedQuad(
  ctx: CanvasRenderingContext2D,
  image: CanvasImageSource,
  source: SourceRect,
  origin: Point2D,
  along: Point2D,
  across: Point2D,
  clip: Point2D[],
  light: number
): void {
  ctx.save();
  pathPolygon(ctx, clip);
  ctx.clip();
  ctx.save();
  ctx.transform(along.x / source.w, along.y / source.w, across.x / source.h, across.y / source.h, origin.x, origin.y);
  ctx.drawImage(image, source.x, source.y, source.w, source.h, 0, 0, source.w, source.h);
  ctx.restore();
  shadeFill(ctx, light);
  ctx.restore();
}

export function drawTiledQuad(
  ctx: CanvasRenderingContext2D,
  image: CanvasImageSource,
  source: SourceRect,
  origin: Point2D,
  along: Point2D,
  across: Point2D,
  tile_length: number,
  clip: Point2D[],
  light: number
): void {
  const length = Math.hypot(along.x, along.y);
  const tile_count = Math.max(1, Math.ceil(length / tile_length));
  ctx.save();
  pathPolygon(ctx, clip);
  ctx.clip();
  for (let i = 0; i < tile_count; i++) {
    const start = (i * tile_length) / length;
    const step = tile_length / length;
    const mirrored = i % 2 === 1;
    const tile_origin = {
      x: origin.x + along.x * (mirrored ? start + step : start),
      y: origin.y + along.y * (mirrored ? start + step : start),
    };
    const tile_along = { x: along.x * step * (mirrored ? -1 : 1), y: along.y * step * (mirrored ? -1 : 1) };
    ctx.save();
    ctx.transform(
      tile_along.x / source.w,
      tile_along.y / source.w,
      across.x / source.h,
      across.y / source.h,
      tile_origin.x,
      tile_origin.y
    );
    ctx.drawImage(image, source.x, source.y, source.w, source.h, 0, 0, source.w, source.h);
    ctx.restore();
  }
  pathPolygon(ctx, clip);
  shadeFill(ctx, light);
  ctx.restore();
}
