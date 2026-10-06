import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';

const PADDING = 6;
const GAP = 4;
const ROW_H = 24;
const INK = 'rgb(59, 36, 19)';
const FONT = '12px "Fira Sans", sans-serif';
const BOLD_FONT = 'bold 12px "Fira Sans", sans-serif';

function contains(rect: PanelBounds, p: Point2D): boolean {
  return p.x >= rect.x && p.x <= rect.x + rect.w && p.y >= rect.y && p.y <= rect.y + rect.h;
}
export class SlotsPanel implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private mouse: Point2D = { x: 0, y: 0 };
  private active_slot = 0;

  constructor(
    private context: EditorToolContext,
    private on_change_slots: (delta: number) => void,
    private on_change_bank: (resource: 'food' | 'wood' | 'stone' | 'gold', delta: number) => void
  ) {}

  selectedSlot(): number {
    return this.active_slot;
  }

  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
  }
  mousemove(_canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.mouse = screen;
    return contains(this.bounds, screen);
  }

  isHovering(): boolean {
    return contains(this.bounds, this.mouse);
  }

  setHovering(_hovering: boolean): void {}
  isClicking(): boolean {
    return false;
  }
  setClicking(_clicking: boolean): void {}
  scroll(_dy: number): boolean {
    return contains(this.bounds, this.mouse);
  }
  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0 || !contains(this.bounds, this.mouse)) {
      return false;
    }
    const x = this.bounds.x + PADDING;
    let y = this.bounds.y + PADDING + ROW_H + GAP;
    if (contains({ x: x + 80, y, w: 28, h: ROW_H }, this.mouse)) {
      this.on_change_slots(-1);
      return true;
    }
    if (contains({ x: x + 114, y, w: 28, h: ROW_H }, this.mouse)) {
      this.on_change_slots(1);
      return true;
    }
    y += (ROW_H + GAP) * 2;
    const resources: ('food' | 'wood' | 'stone' | 'gold')[] = ['food', 'wood', 'stone', 'gold'];
    for (const res of resources) {
      if (contains({ x: x + 70, y, w: 24, h: ROW_H }, this.mouse)) {
        this.on_change_bank(res, -50);
        return true;
      }
      if (contains({ x: x + 130, y, w: 24, h: ROW_H }, this.mouse)) {
        this.on_change_bank(res, 50);
        return true;
      }
      y += ROW_H + GAP;
    }
    return false;
  }

  mouseup(_e: MouseEvent): void {}
  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    if (this.bounds.w <= 0 || this.bounds.h <= 0) {
      return;
    }
    const draw_config = { fill_style: 'transparent', stroke_width: 1, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      const doc = this.context.doc();
      const players = doc?.players ?? 1;
      const x = this.bounds.x + PADDING;
      let y = this.bounds.y + PADDING;
      drawText(ctx, 'Player Slots', { p: { x, y: y + 10 }, w: this.bounds.w, fill_style: INK, font: BOLD_FONT });
      y += ROW_H + GAP;
      drawText(ctx, `Slots: ${players}`, { p: { x, y: y + 10 }, w: 80, fill_style: INK, font: FONT });
      this.drawButton(ctx, '-', { x: x + 80, y, w: 28, h: ROW_H });
      this.drawButton(ctx, '+', { x: x + 114, y, w: 28, h: ROW_H });
      y += ROW_H + GAP * 2;
      this.drawBankControls(ctx, doc, x, y);
    });
  }
  private drawBankControls(
    ctx: CanvasRenderingContext2D,
    doc: ReturnType<EditorToolContext['doc']>,
    x: number,
    start_y: number
  ): void {
    let y = start_y;
    drawText(ctx, 'Starting Bank', { p: { x, y: y + 10 }, w: this.bounds.w, fill_style: INK, font: BOLD_FONT });
    y += ROW_H + GAP;
    const bank = doc?.starting_bank ?? { food: 0, wood: 0, stone: 0, gold: 0 };
    const resources: ('food' | 'wood' | 'stone' | 'gold')[] = ['food', 'wood', 'stone', 'gold'];
    for (const res of resources) {
      drawText(ctx, `${res}: ${bank[res]}`, { p: { x, y: y + 10 }, w: 70, fill_style: INK, font: FONT });
      this.drawButton(ctx, '-', { x: x + 70, y, w: 24, h: ROW_H });
      this.drawButton(ctx, '+', { x: x + 130, y, w: 24, h: ROW_H });
      y += ROW_H + GAP;
    }
  }

  private drawButton(ctx: CanvasRenderingContext2D, label: string, rect: PanelBounds): void {
    ctx.fillStyle = contains(rect, this.mouse) ? 'rgb(220, 200, 180)' : 'rgb(240, 230, 215)';
    drawRect(ctx, rect, rect.w, rect.h, 3);
    drawText(ctx, label, {
      p: { x: rect.x + rect.w / 2, y: rect.y + rect.h / 2 },
      w: rect.w,
      fill_style: INK,
      font: FONT,
      align: 'center',
      baseline: 'middle',
    });
  }

  xi(): number {
    return this.bounds.x;
  }
  yi(): number {
    return this.bounds.y;
  }
  xf(): number {
    return this.bounds.x + this.bounds.w;
  }
  yf(): number {
    return this.bounds.y + this.bounds.h;
  }
  w(): number {
    return this.bounds.w;
  }
  h(): number {
    return this.bounds.h;
  }
}
