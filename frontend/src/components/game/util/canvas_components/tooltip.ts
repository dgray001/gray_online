import type { BoardTransformData } from '../canvas_board/canvas_board';
import { drawRect, drawText } from '../canvas_util';
import type { Point2D } from '../objects2d';
import { configDraw } from './canvas_component';

export interface CanvasSize {
  width: number;
  height: number;
}

export type TooltipDraw<T> = (
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  canvas_size: CanvasSize,
  p: Point2D,
  data: T
) => void;

export interface TooltipConfig<T> {
  delay_ms: number;
  draw: TooltipDraw<T>;
}

export const DEFAULT_TOOLTIP_CONFIG: TooltipConfig<string> = {
  delay_ms: 250,
  draw: defaultTooltipDraw,
};

const MAX_TOOLTIP_TEXT_WIDTH = 220;

export function wrapText(ctx: CanvasRenderingContext2D, text: string, max_width: number): string[] {
  const words = text.split(' ');
  const lines: string[] = [];
  let current = '';
  for (const word of words) {
    const attempt = current ? `${current} ${word}` : word;
    if (current && ctx.measureText(attempt).width > max_width) {
      lines.push(current);
      current = word;
    } else {
      current = attempt;
    }
  }
  if (current) {
    lines.push(current);
  }
  return lines;
}

// anchors the box on whichever side of p has more room, then clamps it fully on screen
export function anchorTooltipBox(p: Point2D, w: number, h: number, canvas_size: CanvasSize): Point2D {
  const x = p.x > canvas_size.width / 2 ? p.x - w : p.x;
  return {
    x: Math.max(0, Math.min(x, canvas_size.width - w)),
    y: Math.max(0, p.y - 2 - h),
  };
}

export function defaultTooltipDraw(
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  canvas_size: CanvasSize,
  p: Point2D,
  text: string
) {
  const padding = 4;
  const row_height = 16;
  configDraw(ctx, transform, { fill_style: 'transparent', stroke_width: 0, fixed_position: true }, false, false, () => {
    ctx.font = '12px serif';
    const lines = wrapText(ctx, text, MAX_TOOLTIP_TEXT_WIDTH);
    const w =
      Math.min(MAX_TOOLTIP_TEXT_WIDTH, Math.max(...lines.map((line) => ctx.measureText(line).width))) + 2 * padding;
    const h = lines.length * row_height + 2 * padding;
    const box_p = anchorTooltipBox(p, w, h, canvas_size);
    ctx.fillStyle = 'rgba(0, 0, 0, 0.85)';
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    ctx.lineWidth = 1;
    drawRect(ctx, box_p, w, h, 3);
    lines.forEach((line, i) => {
      drawText(ctx, line, {
        p: { x: box_p.x + padding, y: box_p.y + padding + (i + 0.5) * row_height },
        w: w - 2 * padding,
        fill_style: 'white',
        align: 'left',
        baseline: 'middle',
        font: '12px serif',
      });
    });
  });
}

export interface TooltipState<T = string> {
  hover_ms: number;
  config: TooltipConfig<T>;
}

export function createTooltipState<T = string>(config?: Partial<TooltipConfig<T>>): TooltipState<T> {
  return { hover_ms: 0, config: { ...(DEFAULT_TOOLTIP_CONFIG as unknown as TooltipConfig<T>), ...config } };
}

export function shouldShowTooltip(
  state: { hover_ms: number; config: { delay_ms: number } },
  hovering: boolean,
  clicked: boolean,
  dt: number
): boolean {
  if (!hovering) {
    state.hover_ms = 0;
    return false;
  }
  state.hover_ms = clicked ? state.config.delay_ms : state.hover_ms + dt;
  return state.hover_ms >= state.config.delay_ms;
}

export function drawTooltip<T>(
  state: TooltipState<T>,
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  canvas_size: CanvasSize,
  p: Point2D,
  data: T
) {
  state.config.draw(ctx, transform, canvas_size, p, data);
}
