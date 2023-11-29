
/** Draws a hexagon on the canvas */
export function drawHexagon(ctx: CanvasRenderingContext2D, x: number, y: number, r: number) {
  const a = 2 * Math.PI / 6;
  ctx.beginPath();
  for (var i = 0; i < 6; i++) {
    ctx.lineTo(x + r * Math.cos(a * i), y + r * Math.sin(a * i));
  }
  ctx.closePath();
  ctx.stroke();
}
