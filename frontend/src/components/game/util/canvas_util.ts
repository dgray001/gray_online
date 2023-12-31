import {Point2D} from "./objects2d";

/** Draws a hexagon on the canvas */
export function drawHexagon(ctx: CanvasRenderingContext2D, c: Point2D, r: number, o = Math.PI / 6) {
  drawNgon(ctx, 6, c, r, o);
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
