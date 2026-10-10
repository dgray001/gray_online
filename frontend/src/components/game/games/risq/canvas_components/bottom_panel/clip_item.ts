import { screenToCanvas, type BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { BottomPanelItem } from './bottom_panel';

export function clipBottomPanelItem(
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  item: Pick<BottomPanelItem, 'xi' | 'xf' | 'yi' | 'yf'>
): void {
  const corners = [
    { x: item.xi(), y: item.yi() },
    { x: item.xf(), y: item.yi() },
    { x: item.xf(), y: item.yf() },
    { x: item.xi(), y: item.yf() },
  ].map((p) => screenToCanvas(p, transform));
  ctx.beginPath();
  corners.forEach((p, i) => (i ? ctx.lineTo(p.x, p.y) : ctx.moveTo(p.x, p.y)));
  ctx.closePath();
  ctx.clip();
}
