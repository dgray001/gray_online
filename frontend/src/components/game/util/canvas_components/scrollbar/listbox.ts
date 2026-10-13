import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawRect, drawText } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import { subtractPoint2D } from '../../objects2d';
import { configDraw, xc, type CanvasComponent, type DrawConfig } from '../canvas_component';
import type { DwgRectScrollbar } from './rect_scrollbar';

/** Data describing a listbox config */
export declare interface ListboxConfig<T extends CanvasComponent, R extends DwgRectScrollbar = DwgRectScrollbar> {
  list: T[];
  scrollbar: R;
  draw_config: DrawConfig;
  gap?: number;
  padding?: number;
  title?: {
    text: string;
    size: number;
    font_color: string;
  };
}

export class DwgListbox<
  T extends CanvasComponent = CanvasComponent,
  R extends DwgRectScrollbar = DwgRectScrollbar,
> implements CanvasComponent {
  protected config: ListboxConfig<T, R>;
  // Index of the first list item shown
  private draw_start = 0;
  // Index of the last list item shown
  private draw_end = 0;
  // Maintain a list of coordinates for mouse move events
  private list_coordinates: Point2D[] = [];
  private disabled = false;

  constructor(config: ListboxConfig<T, R>) {
    config.scrollbar.setValue({ value: 0, value_min: 0, value_max: 0 });
    const scroll_callback = config.scrollbar.scrollCallback;
    config.scrollbar.scrollCallback = (value: number) => {
      if (scroll_callback) {
        scroll_callback(value);
      }
      this.layoutRows();
      this.refreshMouse();
    };
    this.config = config;
    this.setList(config.list);
  }

  setList(list: T[]): void {
    this.config.list = list;
    this.layoutRows();
    this.refreshMouse();
  }

  protected getList(): T[] {
    return this.config.list;
  }

  isHovering(): boolean {
    if (this.disabled) {
      return false;
    }
    return this.config.scrollbar.isHovering();
  }

  setHovering(hovering: boolean): void {
    this.config.scrollbar.setHovering(hovering);
    if (!hovering) {
      for (const item of this.config.list) {
        item.setHovering(false);
      }
    }
  }

  isClicking(): boolean {
    if (this.disabled) {
      return false;
    }
    return this.config.scrollbar.isClicking();
  }

  setClicking(clicking: boolean): void {
    this.config.scrollbar.setClicking(clicking);
    if (!clicking) {
      for (const item of this.config.list) {
        item.setClicking(false);
        item.setHovering(false);
      }
    }
  }

  setAllSizes(size: number, p: Point2D, w: number, h: number): void {
    const title_size = this.config.title?.size ?? 0;
    this.config.scrollbar.setAllSizes(size, { x: p.x, y: p.y + title_size }, w, h - title_size);
    this.layoutRows();
    this.refreshMouse();
  }

  getPadding(): number {
    return this.config.padding ?? 0;
  }

  getGap(): number {
    return this.config.gap ?? 0;
  }

  disable(): void {
    this.disabled = true;
    this.config.scrollbar.disable();
  }

  enable(): void {
    this.disabled = false;
    this.config.scrollbar.enable();
  }

  private layoutRows(): void {
    const scrollbar = this.config.scrollbar;
    const padding = this.getPadding();
    const total =
      this.config.list.reduce((sum: number, el: T): number => sum + el.h(), 0) +
      Math.max(0, this.config.list.length - 1) * this.getGap();
    scrollbar.setValue({
      value: scrollbar.value(),
      value_min: 0,
      value_max: Math.max(0, total - scrollbar.h() + 2 * padding),
    });
    this.draw_start = this.config.list.length;
    this.draw_end = -1;
    let y = scrollbar.yi() + padding - scrollbar.value();
    this.list_coordinates = this.config.list.map((el: T, i: number): Point2D => {
      const p = { x: this.xi() + padding, y };
      if (y + el.h() >= scrollbar.yi() + padding && y <= scrollbar.yf() - padding) {
        this.draw_start = Math.min(this.draw_start, i);
        this.draw_end = i;
      }
      y += el.h() + this.getGap();
      return p;
    });
  }

  private refreshMouse(): void {
    const canvas = this.config.scrollbar.getLastMousemoveCanvas();
    const screen = this.config.scrollbar.getLastMousemoveScreen();
    const transform = this.config.scrollbar.getLastMousemoveTransform();
    if (canvas && screen && transform) {
      this.mousemove(canvas, screen, transform);
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(ctx, transform, this.config.draw_config, this.isHovering(), this.isClicking(), () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
      if (this.config.title) {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.config.title.size);
        drawText(ctx, this.config.title.text, {
          p: { x: xc(this), y: this.yi() + this.config.title.size - this.getPadding() },
          w: this.w() - 2 * this.getPadding(),
          font: `${this.config.title.size - 2 * this.getPadding()}px serif`,
          fill_style: this.config.title.font_color,
          align: 'center',
          baseline: 'ideographic',
        });
      }
      const padding = this.getPadding();
      ctx.save();
      ctx.beginPath();
      ctx.rect(
        this.xi() + padding,
        this.config.scrollbar.yi() + padding,
        this.w() - padding * 2,
        this.config.scrollbar.h() - padding * 2
      );
      ctx.clip();
      for (let i = this.draw_start; i <= this.draw_end; i++) {
        const el = this.config.list[i];
        const p = this.list_coordinates[i];
        ctx.save();
        ctx.translate(p.x, p.y);
        el.draw(ctx, transform, dt);
        ctx.restore();
      }
      ctx.restore();
    });
    this.config.scrollbar.draw(ctx, transform, dt);
    if (this.disabled) {
      configDraw(
        ctx,
        transform,
        {
          ...this.config.draw_config,
          fill_style: 'rgba(255, 255, 255, 0.4)',
        },
        false,
        false,
        () => {
          drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
        }
      );
    }
  }

  scroll(dy: number, mode: number, _dx?: number): boolean {
    return this.config.scrollbar.scroll(dy, mode, _dx);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.config.scrollbar.mousemove(canvas, screen, transform);
    const is_hovering = this.isHovering();
    const point = this.config.draw_config.fixed_position ? screen : canvas;
    for (const [i, el] of this.config.list.entries()) {
      if (this.disabled || !is_hovering || i < this.draw_start || i > this.draw_end) {
        el.setHovering(false);
      } else {
        const local = subtractPoint2D(point, this.list_coordinates[i]);
        el.mousemove(local, local, transform);
      }
    }
    return is_hovering;
  }

  mousedown(e: MouseEvent): boolean {
    this.config.scrollbar.mousedown(e);
    for (const [i, el] of this.config.list.entries()) {
      if (this.disabled || i < this.draw_start || i > this.draw_end) {
        continue;
      } else {
        el.mousedown(e);
      }
    }
    return this.isClicking();
  }

  mouseup(e: MouseEvent): void {
    for (const [i, el] of this.config.list.entries()) {
      if (this.disabled || i < this.draw_start || i > this.draw_end) {
        el.setClicking(false);
      } else {
        el.mouseup(e);
      }
    }
    this.config.scrollbar.mouseup(e);
  }

  mouseOver(canvas: Point2D, screen: Point2D): boolean {
    return this.config.scrollbar.mouseOver(canvas, screen);
  }

  xi(): number {
    return this.config.scrollbar.xi();
  }

  xf(): number {
    return this.config.scrollbar.xf();
  }

  yi(): number {
    const yi = this.config.scrollbar.yi();
    if (this.config.title) {
      return yi - this.config.title.size;
    }
    return yi;
  }

  yf(): number {
    return this.config.scrollbar.yf();
  }

  w(): number {
    return this.config.scrollbar.w();
  }

  h(): number {
    const h = this.config.scrollbar.h();
    if (this.config.title) {
      return h + this.config.title.size;
    }
    return h;
  }
}
