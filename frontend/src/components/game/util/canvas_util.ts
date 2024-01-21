import {Point2D} from "./objects2d";

/** Draws a hexagon on the canvas */
export function drawHexagon(ctx: CanvasRenderingContext2D, c: Point2D, r: number, o = Math.PI / 6) {
  drawNgon(ctx, 6, c, r, o);
}

/** Draws a circle on the canvas */
export function drawCircle(ctx: CanvasRenderingContext2D, c: Point2D, r: number) {
  ctx.beginPath();
  ctx.ellipse(c.x, c.y, r, r, 0, 0, 2 * Math.PI);
  ctx.closePath();
  ctx.stroke();
  ctx.fill();
}

/** Draws a ellipse on the canvas */
export function drawEllipse(ctx: CanvasRenderingContext2D, c: Point2D, r: Point2D) {
  ctx.beginPath();
  ctx.ellipse(c.x, c.y, r.x, r.y, 0, 0, 2 * Math.PI);
  ctx.closePath();
  ctx.stroke();
  ctx.fill();
}

/** Draws a rectangle on the canvas */
export function drawRect(ctx: CanvasRenderingContext2D, pi: Point2D, w: number, h: number, r = 0) {
  ctx.beginPath();
  ctx.roundRect(pi.x, pi.y, w, h, r);
  ctx.closePath();
  ctx.stroke();
  ctx.fill();
}

/** Draws a regular n-gon on the canvas */
export function drawNgon(ctx: CanvasRenderingContext2D, n: number, c: Point2D, r: number, o = 0) {
  if (n < 3) {
    console.error('n must be 3+');
  }
  const a = 2 * Math.PI / n;
  ctx.beginPath();
  for (var i = 0; i < n; i++) {
    ctx.lineTo(c.x + r * Math.cos(a * i + o), c.y + r * Math.sin(a * i + o));
  }
  ctx.closePath();
  ctx.stroke();
  ctx.fill();
}

/** Config data for drawing text */
export declare interface DrawTextConfig {
  p: Point2D;
  w: number;
  fill_style?: string;
  stroke_style?: string;
  stroke_width?: number;
  font?: string;
  align?: CanvasTextAlign;
  baseline?: CanvasTextBaseline;
  direction?: CanvasDirection;
}

/** Draws text from config */
export function drawText(ctx: CanvasRenderingContext2D, s: string, config: DrawTextConfig) {
  ctx.beginPath();
  if (!!ctx.font) {
    ctx.font = config.font;
  }
  ctx.textAlign = config.align ?? 'left';
  ctx.textBaseline = config.baseline ?? 'top';
  ctx.direction = config.direction ?? 'ltr';
  if (!!config.fill_style) {
    ctx.fillStyle = config.fill_style;
    ctx.fillText(s, config.p.x, config.p.y, config.w);
  }
  if (!!config.stroke_style) {
    ctx.strokeStyle = config.stroke_style;
    ctx.lineWidth = config.stroke_width;
    ctx.strokeText(s, config.p.x, config.p.y, config.w);
  }
  ctx.closePath();
}
