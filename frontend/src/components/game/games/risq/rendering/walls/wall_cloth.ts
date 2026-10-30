import type { Point2D } from '../../../../util/objects2d';

const POLE_COLOR = 'rgb(58, 42, 30)';
const ROD_COLOR = 'rgb(58, 42, 30)';
const FOLD_COUNT = 3;
const WAVE_STEPS = 12;

export function drawHangingBanner(
  ctx: CanvasRenderingContext2D,
  color: string,
  top_left: Point2D,
  along: Point2D,
  drop: number,
  light: number
): void {
  ctx.save();
  ctx.transform(along.x, along.y, 0, drop, top_left.x, top_left.y);
  ctx.beginPath();
  ctx.moveTo(0, 0);
  ctx.lineTo(1, 0);
  ctx.lineTo(1, 1);
  ctx.lineTo(0.5, 0.78);
  ctx.lineTo(0, 1);
  ctx.closePath();
  ctx.fillStyle = color;
  ctx.fill();
  ctx.clip();
  const fade = ctx.createLinearGradient(0, 0, 0, 1);
  fade.addColorStop(0, 'rgba(0, 0, 0, 0)');
  fade.addColorStop(1, 'rgba(0, 0, 0, 0.25)');
  ctx.fillStyle = fade;
  ctx.fillRect(0, 0, 1, 1);
  ctx.fillStyle = 'rgba(0, 0, 0, 0.1)';
  for (let k = 0; k < FOLD_COUNT; k++) {
    ctx.fillRect((k + 0.5) / FOLD_COUNT - 0.06, 0, 0.12, 1);
  }
  if (light < 1) {
    ctx.fillStyle = `rgba(0, 0, 0, ${1 - light})`;
    ctx.fillRect(0, 0, 1, 1);
  }
  ctx.restore();
  ctx.save();
  ctx.transform(along.x, along.y, 0, drop, top_left.x, top_left.y);
  ctx.fillStyle = ROD_COLOR;
  ctx.fillRect(-0.04, -0.015, 1.08, 0.05);
  ctx.restore();
}

export function drawFlag(
  ctx: CanvasRenderingContext2D,
  color: string,
  base: Point2D,
  pole_height: number,
  cloth_w: number,
  cloth_h: number,
  wave: number
): void {
  const pole_w = Math.max(1, cloth_h * 0.12);
  ctx.fillStyle = POLE_COLOR;
  ctx.fillRect(base.x - pole_w / 2, base.y - pole_height, pole_w, pole_height);
  const top = base.y - pole_height;
  const edge = (t: number, bottom: boolean): Point2D => {
    const sway = wave * Math.sin(t * Math.PI * 2 * 1.3) * Math.min(1, t * 1.6);
    const height = cloth_h * (1 - 0.55 * t);
    return { x: base.x + cloth_w * t, y: top + sway + (bottom ? height : 0) };
  };
  ctx.beginPath();
  ctx.moveTo(base.x, top);
  for (let i = 1; i <= WAVE_STEPS; i++) {
    const p = edge(i / WAVE_STEPS, false);
    ctx.lineTo(p.x, p.y);
  }
  for (let i = WAVE_STEPS; i >= 0; i--) {
    const t = i / WAVE_STEPS;
    const p = edge(t, true);
    ctx.lineTo(p.x, t > 0.82 ? p.y - (t - 0.82) * 1.2 * cloth_h : p.y);
  }
  ctx.closePath();
  ctx.fillStyle = color;
  ctx.fill();
  const shade = ctx.createLinearGradient(0, top, 0, top + cloth_h);
  shade.addColorStop(0, 'rgba(0, 0, 0, 0)');
  shade.addColorStop(1, 'rgba(0, 0, 0, 0.25)');
  ctx.fillStyle = shade;
  ctx.fill();
}
