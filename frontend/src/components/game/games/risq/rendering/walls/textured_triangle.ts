import type { Point2D } from '../../../../util/objects2d';

const SEAM_GROW = 0.6;

function grow(triangle: Point2D[]): Point2D[] {
  const cx = (triangle[0].x + triangle[1].x + triangle[2].x) / 3;
  const cy = (triangle[0].y + triangle[1].y + triangle[2].y) / 3;
  return triangle.map((p) => {
    const length = Math.hypot(p.x - cx, p.y - cy) || 1;
    return { x: p.x + ((p.x - cx) / length) * SEAM_GROW, y: p.y + ((p.y - cy) / length) * SEAM_GROW };
  });
}

export function drawTexturedTriangle(
  ctx: CanvasRenderingContext2D,
  image: CanvasImageSource,
  source: [Point2D, Point2D, Point2D],
  target: [Point2D, Point2D, Point2D],
  light: number
): void {
  const [s0, s1, s2] = source;
  const [d0, d1, d2] = target;
  const determinant = (s1.x - s0.x) * (s2.y - s0.y) - (s2.x - s0.x) * (s1.y - s0.y);
  if (Math.abs(determinant) < 1e-9) {
    return;
  }
  const a = ((d1.x - d0.x) * (s2.y - s0.y) - (d2.x - d0.x) * (s1.y - s0.y)) / determinant;
  const c = ((d2.x - d0.x) * (s1.x - s0.x) - (d1.x - d0.x) * (s2.x - s0.x)) / determinant;
  const b = ((d1.y - d0.y) * (s2.y - s0.y) - (d2.y - d0.y) * (s1.y - s0.y)) / determinant;
  const d = ((d2.y - d0.y) * (s1.x - s0.x) - (d1.y - d0.y) * (s2.x - s0.x)) / determinant;
  const e = d0.x - a * s0.x - c * s0.y;
  const f = d0.y - b * s0.x - d * s0.y;
  const clip = grow([d0, d1, d2]);
  ctx.save();
  ctx.beginPath();
  ctx.moveTo(clip[0].x, clip[0].y);
  ctx.lineTo(clip[1].x, clip[1].y);
  ctx.lineTo(clip[2].x, clip[2].y);
  ctx.closePath();
  ctx.clip();
  ctx.save();
  ctx.transform(a, b, c, d, e, f);
  ctx.drawImage(image, 0, 0);
  ctx.restore();
  if (light < 1) {
    ctx.fillStyle = `rgba(0, 0, 0, ${1 - light})`;
    ctx.fill();
  }
  ctx.restore();
}

export function drawTexturedTrapezoid(
  ctx: CanvasRenderingContext2D,
  image: HTMLImageElement,
  bottom: [Point2D, Point2D],
  top: [Point2D, Point2D],
  light: number
): void {
  const w = image.naturalWidth;
  const h = image.naturalHeight;
  drawTexturedTriangle(
    ctx,
    image,
    [
      { x: 0, y: h },
      { x: w, y: h },
      { x: w, y: 0 },
    ],
    [bottom[0], bottom[1], top[1]],
    light
  );
  drawTexturedTriangle(
    ctx,
    image,
    [
      { x: 0, y: h },
      { x: w, y: 0 },
      { x: 0, y: 0 },
    ],
    [bottom[0], top[1], top[0]],
    light
  );
}
