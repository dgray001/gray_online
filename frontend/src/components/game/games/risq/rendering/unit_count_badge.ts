import { drawRect, drawText } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';

export function drawUnitCountBadge(ctx: CanvasRenderingContext2D, count: number, p: Point2D, s: number): void {
  const text = `×${count}`;
  const font_size = Math.max(10, 0.22 * s);
  ctx.font = `bold ${font_size}px serif`;
  const badge_w = ctx.measureText(text).width + 6;
  const badge_h = font_size + 4;
  const badge_p = { x: p.x + s - badge_w, y: p.y + s - badge_h };
  ctx.fillStyle = 'rgba(0, 0, 0, 0.75)';
  ctx.strokeStyle = 'transparent';
  drawRect(ctx, badge_p, badge_w, badge_h, 3);
  drawText(ctx, text, {
    p: { x: badge_p.x + 0.5 * badge_w, y: badge_p.y + 0.5 * badge_h },
    w: badge_w,
    fill_style: 'white',
    align: 'center',
    baseline: 'middle',
    font: `bold ${font_size}px serif`,
  });
}
