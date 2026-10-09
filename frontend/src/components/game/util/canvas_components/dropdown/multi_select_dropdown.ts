import { clampNumber } from '../../../../../scripts/math';
import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawRect, drawText } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';
import { configDraw } from '../canvas_component';
import type { RectScrollbarConfig } from '../scrollbar/rect_scrollbar';
import { DwgRectScrollbar } from '../scrollbar/rect_scrollbar';
import { queueTooltipDraw } from '../tooltip';

export declare interface MultiSelectOption {
  id: number;
  label: string;
  icon?: string;
}

export declare interface DropdownBounds {
  x: number;
  y: number;
  w: number;
  h: number;
}

export declare interface MultiSelectDropdownConfig {
  title: string;
  options: MultiSelectOption[];
  /** Edited in place; onChange fires after every edit */
  selected: Set<number>;
  onChange: () => void;
  getIcon: (path: string) => HTMLImageElement;
  /** The popover stays inside this rect, scrolling its option rows if they don't fit */
  bounds: () => DropdownBounds;
}

/** The button, its clear badge, or an option by index */
type DropdownTarget = 'button' | 'clear' | number;

type PopoverRect = DropdownBounds;

const ROW_H = 18;
const POPOVER_W = 200;
const POPOVER_OVERLAP = 8;
const MIN_POPOVER_ROWS = 1.5;
const CLEAR_BADGE = 12;
const PADDING = 4;
const CHECKBOX = 12;
const ICON = 14;
const SCROLLBAR_W = 10;

function popoverScrollbarConfig(value: number, max_scroll: number): RectScrollbarConfig {
  const fill = (alpha: number): { fill_style: string; stroke_width: number; fixed_position: boolean } => ({
    fill_style: `rgba(59, 36, 19, ${alpha})`,
    stroke_width: 0,
    fixed_position: true,
  });
  return {
    scrollbar_config: {
      value: { value, value_min: 0, value_max: max_scroll },
      step_size: ROW_H,
      scroll_pixel_constant: 1,
    },
    p: { x: 0, y: 0 },
    w: SCROLLBAR_W,
    h: 0,
    scrollbar_size: SCROLLBAR_W,
    vertical: true,
    draw_config: { ...fill(0.55), hover_fill_style: fill(0.75).fill_style, click_fill_style: fill(0.9).fill_style },
    space_draw_config: { ...fill(0.12), stroke_matches_fill_style: true },
    background_color: 'rgba(59, 36, 19, 0.08)',
    min_bar_size: 12,
    arrow_scroll_amount: ROW_H,
    space_scroll_amount: 3 * ROW_H,
  };
}

class PopoverScrollbar extends DwgRectScrollbar {
  constructor(value: number, max_scroll: number) {
    super(popoverScrollbarConfig(value, max_scroll));
  }

  scrollCallback(_value: number): void {}
}

export class DwgMultiSelectDropdown implements CanvasComponent {
  private p: Point2D = { x: 0, y: 0 };
  private size: Point2D = { x: 0, y: 0 };
  private open = false;
  private hover?: DropdownTarget;
  private pressed?: DropdownTarget;
  private over_popover = false;
  private scrollbar?: PopoverScrollbar;

  constructor(private config: MultiSelectDropdownConfig) {}

  setPosition(p: Point2D): void {
    this.p = p;
  }

  setSize(w: number, h: number): void {
    this.size = { x: w, y: h };
  }

  isOpen(): boolean {
    return this.open;
  }

  close(): void {
    this.open = false;
    this.hover = undefined;
    this.pressed = undefined;
    this.over_popover = false;
    this.scrollbar = undefined;
  }

  private popoverRect(): PopoverRect {
    const bounds = this.config.bounds();
    const w = Math.min(POPOVER_W, bounds.w);
    const y = this.yf() - POPOVER_OVERLAP;
    const full_h = this.config.options.length * ROW_H;
    const room_h = Math.max(MIN_POPOVER_ROWS * ROW_H, bounds.y + bounds.h - y);
    return { x: clampNumber(this.xi(), bounds.x, bounds.x + bounds.w - w), y, w, h: Math.min(full_h, room_h) };
  }

  private scrollOffset(): number {
    return this.scrollbar?.value() ?? 0;
  }

  private syncScrollbar(): void {
    const pop = this.popoverRect();
    const max_scroll = Math.max(0, this.config.options.length * ROW_H - pop.h);
    if (max_scroll === 0) {
      this.scrollbar = undefined;
      return;
    }
    if (this.scrollbar?.maxValue() !== max_scroll) {
      this.scrollbar = new PopoverScrollbar(Math.min(this.scrollOffset(), max_scroll), max_scroll);
    }
    this.scrollbar.setAllSizes(SCROLLBAR_W, { x: pop.x + pop.w - SCROLLBAR_W - 2, y: pop.y }, SCROLLBAR_W, pop.h);
  }

  private overPopover(m: Point2D): boolean {
    const pop = this.popoverRect();
    return this.open && m.x >= pop.x && m.x <= pop.x + pop.w && m.y >= pop.y && m.y <= pop.y + pop.h;
  }

  private overButton(m: Point2D): boolean {
    return m.x >= this.xi() && m.x <= this.xf() && m.y >= this.yi() && m.y <= this.yf();
  }

  private hitTarget(m: Point2D): DropdownTarget | undefined {
    if (!this.overPopover(m)) {
      if (this.overClearBadge(m)) {
        return 'clear';
      }
      return this.overButton(m) ? 'button' : undefined;
    }
    const pop = this.popoverRect();
    const index = Math.floor((m.y - pop.y + this.scrollOffset()) / ROW_H);
    const over_scrollbar = !!this.scrollbar && m.x >= pop.x + pop.w - SCROLLBAR_W - 2;
    return !over_scrollbar && index < this.config.options.length ? index : undefined;
  }

  private clearBadgePosition(): Point2D {
    return { x: this.xf() - CLEAR_BADGE - 2, y: this.yi() + 2 };
  }

  private overClearBadge(m: Point2D): boolean {
    const p = this.clearBadgePosition();
    const inside = m.x >= p.x && m.x <= p.x + CLEAR_BADGE && m.y >= p.y && m.y <= p.y + CLEAR_BADGE;
    return this.config.selected.size > 0 && inside;
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.syncScrollbar();
    this.scrollbar?.mousemove(canvas, screen, transform);
    this.hover = this.hitTarget(screen);
    this.over_popover = this.overPopover(screen);
    return this.hover !== undefined || this.over_popover;
  }

  scroll(dy: number): boolean {
    if (this.over_popover) {
      this.scrollbar?._scroll(Math.sign(dy) * 2 * ROW_H);
    }
    return this.over_popover;
  }

  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0) {
      return false;
    }
    if (this.scrollbar?.mousedown(e)) {
      return true;
    }
    if (this.hover !== undefined) {
      this.pressed = this.hover;
      return true;
    }
    if (this.over_popover) {
      return true;
    }
    const captured = this.open;
    this.close();
    return captured;
  }

  mouseup(e: MouseEvent): void {
    this.scrollbar?.mouseup(e);
    const target = this.pressed;
    this.pressed = undefined;
    if (e.button !== 0 || target === undefined || target !== this.hover) {
      return;
    }
    if (target === 'button') {
      this.toggleOpen();
    } else if (target === 'clear') {
      this.config.selected.clear();
      this.config.onChange();
    } else {
      this.toggle(this.config.options[target].id);
    }
  }

  private toggleOpen(): void {
    if (this.open) {
      this.close();
    } else {
      this.open = true;
    }
  }

  private label(): string {
    const count = this.config.selected.size;
    return count > 0 ? `${this.config.title} (${count})` : this.config.title;
  }

  private drawButton(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const fill_style = this.config.selected.size > 0 ? 'rgb(255, 232, 160)' : 'rgb(241, 226, 196)';
    const draw_config = { fill_style, stroke_style: 'rgb(59, 36, 19)', stroke_width: 0.6, fixed_position: true };
    configDraw(ctx, transform, draw_config, this.hover === 'button', this.pressed === 'button', () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h(), 3);
      drawText(ctx, this.label(), {
        p: { x: this.xi() + PADDING + 2, y: this.yc() },
        w: this.w() - 2 * PADDING - 14,
        fill_style: 'rgb(59, 36, 19)',
        baseline: 'middle',
        font: 'bold 11px serif',
      });
      this.drawCaret(ctx);
      this.drawClearBadge(ctx);
    });
  }

  private drawCaret(ctx: CanvasRenderingContext2D): void {
    const x = this.xf() - PADDING - 6;
    const y = this.config.selected.size > 0 ? this.yi() + 17 : this.yc();
    const dy = this.open ? -2 : 2;
    ctx.beginPath();
    ctx.moveTo(x - 4, y - dy);
    ctx.lineTo(x + 4, y - dy);
    ctx.lineTo(x, y + dy);
    ctx.closePath();
    ctx.fill();
  }

  private drawClearBadge(ctx: CanvasRenderingContext2D): void {
    if (this.config.selected.size === 0) {
      return;
    }
    const p = this.clearBadgePosition();
    const hovered = this.hover === 'clear';
    const inset = 3;
    ctx.strokeStyle = hovered ? 'rgb(122, 46, 27)' : 'rgba(59, 36, 19, 0.6)';
    ctx.lineWidth = hovered ? 2 : 1.5;
    ctx.beginPath();
    ctx.moveTo(p.x + inset, p.y + inset);
    ctx.lineTo(p.x + CLEAR_BADGE - inset, p.y + CLEAR_BADGE - inset);
    ctx.moveTo(p.x + CLEAR_BADGE - inset, p.y + inset);
    ctx.lineTo(p.x + inset, p.y + CLEAR_BADGE - inset);
    ctx.stroke();
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.drawButton(ctx, transform);
    if (this.open) {
      queueTooltipDraw(() => this.drawPopover(ctx, transform, dt));
    }
  }

  private drawPopover(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.syncScrollbar();
    const pop = this.popoverRect();
    const draw_config = {
      fill_style: 'rgb(241, 226, 196)',
      stroke_style: 'rgb(59, 36, 19)',
      stroke_width: 1,
      fixed_position: true,
    };
    configDraw(ctx, transform, draw_config, false, false, () => {
      drawRect(ctx, { x: pop.x, y: pop.y }, pop.w, pop.h, 3);
      ctx.save();
      ctx.beginPath();
      ctx.rect(pop.x, pop.y, pop.w, pop.h);
      ctx.clip();
      for (let i = 0; i < this.config.options.length; i++) {
        this.drawOptionRow(ctx, pop, i, pop.y + i * ROW_H - this.scrollOffset());
      }
      ctx.restore();
    });
    this.scrollbar?.draw(ctx, transform, dt);
  }

  private drawRowHighlight(ctx: CanvasRenderingContext2D, pop: PopoverRect, y: number, hovered: boolean): void {
    if (hovered) {
      ctx.fillStyle = this.pressed === undefined ? 'rgba(59, 36, 19, 0.12)' : 'rgba(59, 36, 19, 0.22)';
      ctx.strokeStyle = 'transparent';
      drawRect(ctx, { x: pop.x + 2, y }, pop.w - 4, ROW_H, 2);
    }
  }

  private drawCheckbox(ctx: CanvasRenderingContext2D, p: Point2D, checked: boolean): void {
    ctx.fillStyle = 'transparent';
    ctx.strokeStyle = 'rgb(59, 36, 19)';
    ctx.lineWidth = 1;
    drawRect(ctx, p, CHECKBOX, CHECKBOX, 2);
    if (checked) {
      ctx.fillStyle = 'rgb(59, 36, 19)';
      ctx.strokeStyle = 'transparent';
      drawRect(ctx, { x: p.x + 3, y: p.y + 3 }, CHECKBOX - 6, CHECKBOX - 6, 1);
    }
  }

  private drawOptionRow(ctx: CanvasRenderingContext2D, pop: PopoverRect, index: number, y: number): void {
    const option = this.config.options[index];
    this.drawRowHighlight(ctx, pop, y, this.hover === index);
    const checkbox_p = { x: pop.x + PADDING + 4, y: y + 0.5 * (ROW_H - CHECKBOX) };
    this.drawCheckbox(ctx, checkbox_p, this.config.selected.has(option.id));
    let x = checkbox_p.x + CHECKBOX + 6;
    if (option.icon) {
      ctx.drawImage(this.config.getIcon(option.icon), x, y + 0.5 * (ROW_H - ICON), ICON, ICON);
      x += ICON + 6;
    }
    drawText(ctx, option.label, {
      p: { x, y: y + 0.5 * ROW_H },
      w: pop.x + pop.w - PADDING - (this.scrollbar ? SCROLLBAR_W + 2 : 0) - x,
      fill_style: 'rgb(59, 36, 19)',
      baseline: 'middle',
      font: '11px serif',
    });
  }

  private toggle(id: number): void {
    if (!this.config.selected.delete(id)) {
      this.config.selected.add(id);
    }
    this.config.onChange();
  }

  isHovering(): boolean {
    return this.hover !== undefined || this.over_popover;
  }

  setHovering(hovering: boolean): void {
    if (!hovering) {
      this.hover = undefined;
      this.over_popover = false;
    }
  }

  isClicking(): boolean {
    return this.pressed !== undefined;
  }

  setClicking(clicking: boolean): void {
    if (!clicking) {
      this.pressed = undefined;
    }
  }

  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.p.x + this.size.x;
  }
  yf(): number {
    return this.p.y + this.size.y;
  }
  yc(): number {
    return this.p.y + 0.5 * this.size.y;
  }
  w(): number {
    return this.size.x;
  }
  h(): number {
    return this.size.y;
  }
}
