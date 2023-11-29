import {Point2D} from "./objects2d";

/** Draws a hexagon on the canvas */
export function drawHexagon(ctx: CanvasRenderingContext2D, c: Point2D, r: number, o = Math.PI / 6) {
  const a = 2 * Math.PI / 6;
  ctx.beginPath();
  for (var i = 0; i < 6; i++) {
    ctx.lineTo(c.x + r * Math.cos(a * i + o), c.y + r * Math.sin(a * i + o));
  }
  ctx.closePath();
  ctx.stroke();
}
