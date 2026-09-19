import { ColorRGB } from '../../../../../scripts/color_rgb';
import type { BoardTransformData } from '../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import type { DwgRisq } from '../risq';

const FADE_MS = 300;
const BASE_HOLD_MS = 1200;
const HOLD_MS_PER_CHAR = 20;
const MAX_HOLD_EXTRA_MS = 4000;
const MAX_VISIBLE = 3;
const ROW_H = 30;
const ROW_GAP = 8;
const ROW_PADDING_X = 14;
const TOP_MARGIN = 12;
const FONT = 'bold 14px serif';

export const RISQ_MESSAGE_WARNING_COLOR = new ColorRGB(255, 200, 60);

interface RisqQueuedMessage {
  text: string;
  color: ColorRGB;
  activated_at?: number;
}

/** Transient top-of-screen system messages (e.g. "Not enough resources"), queued FIFO with a fixed number visible at once */
export class RisqMessageQueue implements CanvasComponent {
  private risq: DwgRisq;
  private queue: RisqQueuedMessage[] = [];
  private active: RisqQueuedMessage[] = [];
  private bounds: { w: number; h: number } = { w: 0, h: 0 };
  private clock = 0;

  constructor(risq: DwgRisq) {
    this.risq = risq;
  }

  isHovering(): boolean {
    return false;
  }
  setHovering(_hovering: boolean): void {}
  isClicking(): boolean {
    return false;
  }
  setClicking(_clicking: boolean): void {}
  mousemove(_canvas: Point2D, _screen: Point2D, _transform: BoardTransformData): boolean {
    return false;
  }
  mousedown(_e: MouseEvent): boolean {
    return false;
  }
  mouseup(_e: MouseEvent): void {}

  enqueue(text: string, color: ColorRGB) {
    this.queue.push({ text, color });
    this.promote();
  }

  private promote() {
    while (this.active.length < MAX_VISIBLE && this.queue.length > 0) {
      const msg = this.queue.shift()!;
      msg.activated_at = this.clock;
      this.active.unshift(msg);
    }
  }

  private holdDuration(text: string): number {
    return BASE_HOLD_MS + Math.min(text.length * HOLD_MS_PER_CHAR, MAX_HOLD_EXTRA_MS);
  }

  private ageOf(msg: RisqQueuedMessage): number {
    return this.clock - msg.activated_at!;
  }

  private alphaFor(msg: RisqQueuedMessage): number {
    const age = this.ageOf(msg);
    const hold = this.holdDuration(msg.text);
    if (age < FADE_MS) {
      return age / FADE_MS;
    }
    if (age < FADE_MS + hold) {
      return 1;
    }
    return Math.max(0, 1 - (age - FADE_MS - hold) / FADE_MS);
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    this.clock += dt;
    this.active = this.active.filter((msg) => this.ageOf(msg) < 2 * FADE_MS + this.holdDuration(msg.text));
    this.promote();
    if (this.active.length === 0) {
      this.bounds = { w: 0, h: 0 };
      return;
    }
    const cx = 0.5 * this.risq.canvasSize().width;
    ctx.font = FONT;
    let max_w = 0;
    for (const [i, msg] of this.active.entries()) {
      const alpha = this.alphaFor(msg);
      if (alpha <= 0) {
        continue;
      }
      const w = ctx.measureText(msg.text).width + 2 * ROW_PADDING_X;
      max_w = Math.max(max_w, w);
      const y = TOP_MARGIN + i * (ROW_H + ROW_GAP);
      const bg_style = msg.color.getBrightness() > 0.5 ? 'rgba(0, 0, 0, 0.65)' : 'rgba(255, 255, 255, 0.65)';
      const prev_alpha = ctx.globalAlpha;
      ctx.globalAlpha = alpha;
      configDraw(
        ctx,
        transform,
        {
          fill_style: bg_style,
          stroke_width: 0,
          fixed_position: true,
        },
        false,
        false,
        () => {
          drawRect(ctx, { x: cx - 0.5 * w, y }, w, ROW_H, 5);
          drawText(ctx, msg.text, {
            p: { x: cx, y: y + 0.5 * ROW_H },
            w,
            fill_style: msg.color.getString(),
            align: 'center',
            baseline: 'middle',
            font: FONT,
          });
        }
      );
      ctx.globalAlpha = prev_alpha;
    }
    this.bounds = { w: max_w, h: this.active.length * ROW_H + (this.active.length - 1) * ROW_GAP };
  }

  xi(): number {
    return 0.5 * (this.risq.canvasSize().width - this.bounds.w);
  }
  xf(): number {
    return this.xi() + this.bounds.w;
  }
  yi(): number {
    return TOP_MARGIN;
  }
  yf(): number {
    return this.yi() + this.bounds.h;
  }
  w(): number {
    return this.bounds.w;
  }
  h(): number {
    return this.bounds.h;
  }
}
