import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { DrawTextConfig } from '../../canvas_util';
import { drawRect, drawText } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import type { DrawConfig } from '../canvas_component';
import { configDraw } from '../canvas_component';
import { queueTooltipDraw } from '../tooltip';
import type { DropdownConfig } from './dropdown';
import { DwgDropdown } from './dropdown';
import type { WithRequired } from '../../../../../scripts/types';

type RectDropdownTextConfig = WithRequired<
  Omit<DrawTextConfig, 'p' | 'w' | 'align' | 'baseline'>,
  'font' | 'fill_style'
>;

export declare interface RectDropdownConfig {
  dropdown_config: DropdownConfig;
  p: Point2D;
  w: number;
  h: number;
  draw_config: DrawConfig;
  popover_draw_config: DrawConfig;
  text_config: RectDropdownTextConfig;
  option_hover_fill_style: string;
  placeholder?: string;
  popover_w?: number;
  row_h?: number;
  max_rows?: number;
  padding?: number;
  r?: number;
}

const DEFAULT_ROW_H = 20;
const DEFAULT_MAX_ROWS = 10;
const DEFAULT_PADDING = 4;
const CARET_HALF_W = 4;
const CARET_HALF_H = 2;
const THUMB_W = 3;
const WHEEL_ROWS = 2;

interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export abstract class DwgRectDropdown extends DwgDropdown {
  private rect_config: RectDropdownConfig;
  private scroll_offset = 0;

  constructor(config: RectDropdownConfig) {
    super(config.dropdown_config);
    this.rect_config = config;
  }

  setPosition(p: Point2D): void {
    this.rect_config.p = { ...p };
  }

  setSize(w: number, h: number): void {
    this.rect_config.w = w;
    this.rect_config.h = h;
  }

  xi(): number {
    return this.rect_config.p.x;
  }
  yi(): number {
    return this.rect_config.p.y;
  }
  xf(): number {
    return this.rect_config.p.x + this.rect_config.w;
  }
  yf(): number {
    return this.rect_config.p.y + this.rect_config.h;
  }
  xc(): number {
    return this.xi() + 0.5 * this.rect_config.w;
  }
  yc(): number {
    return this.yi() + 0.5 * this.rect_config.h;
  }
  w(): number {
    return this.rect_config.w;
  }
  h(): number {
    return this.rect_config.h;
  }

  private rowH(): number {
    return this.rect_config.row_h ?? DEFAULT_ROW_H;
  }

  private padding(): number {
    return this.rect_config.padding ?? DEFAULT_PADDING;
  }

  private contentH(): number {
    return this.options().length * this.rowH();
  }

  private popoverRect(): Rect {
    const rows = Math.min(this.options().length, this.rect_config.max_rows ?? DEFAULT_MAX_ROWS);
    return { x: this.xi(), y: this.yf(), w: this.rect_config.popover_w ?? this.rect_config.w, h: rows * this.rowH() };
  }

  private maxScroll(): number {
    return Math.max(0, this.contentH() - this.popoverRect().h);
  }

  private mousePoint(canvas: Point2D, screen: Point2D): Point2D {
    return this.rect_config.draw_config.fixed_position ? screen : canvas;
  }

  override mouseOver(canvas: Point2D, screen: Point2D): boolean {
    const m = this.mousePoint(canvas, screen);
    return m.x >= this.xi() && m.x <= this.xf() && m.y >= this.yi() && m.y <= this.yf();
  }

  protected override overPopover(canvas: Point2D, screen: Point2D): boolean {
    const m = this.mousePoint(canvas, screen);
    const pop = this.popoverRect();
    return m.x >= pop.x && m.x <= pop.x + pop.w && m.y >= pop.y && m.y <= pop.y + pop.h;
  }

  protected override optionAt(canvas: Point2D, screen: Point2D): number | undefined {
    const m = this.mousePoint(canvas, screen);
    const index = Math.floor((m.y - this.popoverRect().y + this.scroll_offset) / this.rowH());
    return index >= 0 && index < this.options().length ? index : undefined;
  }

  protected override scrolled(dy: number): boolean {
    const max_scroll = this.maxScroll();
    this.scroll_offset = Math.min(
      Math.max(this.scroll_offset + Math.sign(dy) * WHEEL_ROWS * this.rowH(), 0),
      max_scroll
    );
    return max_scroll > 0;
  }

  protected override opened(): void {
    const selected = this.options().findIndex((option) => option.value === this.value());
    this.scroll_offset = Math.min(Math.max(selected, 0) * this.rowH(), this.maxScroll());
  }

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    const config = this.rect_config;
    configDraw(ctx, transform, config.draw_config, this.isButtonHovered(), this.isOpen(), () => {
      drawRect(ctx, config.p, config.w, config.h, config.r);
      this.drawLabel(ctx);
      this.drawCaret(ctx);
    });
    if (this.isOpen()) {
      queueTooltipDraw(() => this.drawPopover(ctx, transform));
    }
  }

  private drawLabel(ctx: CanvasRenderingContext2D): void {
    const padding = this.padding();
    drawText(ctx, this.selectedOption()?.label ?? this.rect_config.placeholder ?? '', {
      ...this.rect_config.text_config,
      p: { x: this.xi() + padding, y: this.yc() },
      w: this.rect_config.w - 3 * padding - 2 * CARET_HALF_W,
      align: 'left',
      baseline: 'middle',
    });
  }

  private drawCaret(ctx: CanvasRenderingContext2D): void {
    const x = this.xf() - this.padding() - CARET_HALF_W;
    const dy = this.isOpen() ? -CARET_HALF_H : CARET_HALF_H;
    ctx.fillStyle = this.rect_config.text_config.fill_style;
    ctx.beginPath();
    ctx.moveTo(x - CARET_HALF_W, this.yc() - dy);
    ctx.lineTo(x + CARET_HALF_W, this.yc() - dy);
    ctx.lineTo(x, this.yc() + dy);
    ctx.closePath();
    ctx.fill();
  }

  private drawPopover(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    const pop = this.popoverRect();
    configDraw(ctx, transform, this.rect_config.popover_draw_config, false, false, () => {
      drawRect(ctx, pop, pop.w, pop.h, this.rect_config.r);
      ctx.save();
      ctx.beginPath();
      ctx.rect(pop.x, pop.y, pop.w, pop.h);
      ctx.clip();
      this.options().forEach((option, i) => this.drawRow(ctx, pop, i, option.label));
      ctx.restore();
      this.drawThumb(ctx, pop);
    });
  }

  private drawRow(ctx: CanvasRenderingContext2D, pop: Rect, index: number, label: string): void {
    const y = pop.y + index * this.rowH() - this.scroll_offset;
    if (this.hoveredOption() === index) {
      ctx.fillStyle = this.rect_config.option_hover_fill_style;
      ctx.strokeStyle = 'transparent';
      drawRect(ctx, { x: pop.x, y }, pop.w, this.rowH());
    }
    drawText(ctx, label, {
      ...this.rect_config.text_config,
      p: { x: pop.x + this.padding(), y: y + 0.5 * this.rowH() },
      w: pop.w - 2 * this.padding() - THUMB_W,
      align: 'left',
      baseline: 'middle',
    });
  }

  private drawThumb(ctx: CanvasRenderingContext2D, pop: Rect): void {
    const max_scroll = this.maxScroll();
    if (max_scroll === 0) {
      return;
    }
    const thumb_h = (pop.h * pop.h) / this.contentH();
    ctx.globalAlpha = 0.4;
    ctx.fillStyle = this.rect_config.text_config.fill_style;
    ctx.fillRect(
      pop.x + pop.w - THUMB_W - 1,
      pop.y + (this.scroll_offset / max_scroll) * (pop.h - thumb_h),
      THUMB_W,
      thumb_h
    );
    ctx.globalAlpha = 1;
  }
}
