import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { DrawTextConfig } from '../../canvas_util';
import { drawRect, drawText } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import type { DrawConfig } from '../canvas_component';
import { configDraw } from '../canvas_component';
import type { InputConfig } from './input';
import { DwgInput } from './input';
import type { WithRequired } from '../../../../../scripts/types';

type RectInputTextConfig = WithRequired<Omit<DrawTextConfig, 'p' | 'w' | 'align' | 'baseline'>, 'font' | 'fill_style'>;

export declare interface RectInputConfig {
  input_config: InputConfig;
  p: Point2D;
  w: number;
  h: number;
  draw_config: DrawConfig;
  text_config: RectInputTextConfig;
  placeholder?: string;
  placeholder_fill_style?: string;
  padding?: number;
  r?: number;
}

const DEFAULT_PADDING = 4;

export abstract class DwgRectInput extends DwgInput {
  private rect_config: RectInputConfig;

  constructor(config: RectInputConfig) {
    super(config.input_config);
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

  protected override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    const config = this.rect_config;
    configDraw(ctx, transform, config.draw_config, this.isHovering(), this.isFocused(), () => {
      drawRect(ctx, config.p, config.w, config.h, config.r);
      this.drawContent(ctx);
    });
  }

  private drawContent(ctx: CanvasRenderingContext2D): void {
    const config = this.rect_config;
    const padding = config.padding ?? DEFAULT_PADDING;
    const placeholder_shown = this.value() === '' && !this.isFocused();
    drawText(ctx, placeholder_shown ? (config.placeholder ?? '') : this.value(), {
      ...config.text_config,
      fill_style: placeholder_shown
        ? (config.placeholder_fill_style ?? config.text_config.fill_style)
        : config.text_config.fill_style,
      p: { x: this.xi() + padding, y: this.yc() },
      w: config.w - 2 * padding,
      align: 'left',
      baseline: 'middle',
    });
    if (this.caretVisible()) {
      const caret_x = this.xi() + padding + ctx.measureText(this.value().slice(0, this.caretIndex())).width;
      ctx.fillStyle = config.text_config.fill_style;
      ctx.fillRect(caret_x, this.yi() + padding, 1, config.h - 2 * padding);
    }
  }

  override mouseOver(canvas: Point2D, screen: Point2D): boolean {
    const m = this.rect_config.draw_config.fixed_position ? screen : canvas;
    return m.x >= this.xi() && m.x <= this.xf() && m.y >= this.yi() && m.y <= this.yf();
  }
}
