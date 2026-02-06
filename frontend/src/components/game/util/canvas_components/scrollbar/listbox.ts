import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawRect, drawText } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
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
  // Time between scrollbar adjustments
  static SCROLLBAR_ADJUST_TIME = 100;
  private scrollbar_adjust_time = DwgListbox.SCROLLBAR_ADJUST_TIME;
  // Maintain a list of coordinates for mouse move events
  private list_coordinates: Point2D[] = [];
  private check_mousemove_on_next_draw = false;
  private disabled = false;

  constructor(config: ListboxConfig<T, R>) {
    config.scrollbar.setValue({ value: 0, value_min: 0, value_max: 0 });
    const scroll_callback = config.scrollbar.scrollCallback;
    config.scrollbar.scrollCallback = (value: number) => {
      if (scroll_callback) {
        scroll_callback(value);
      }
      this.check_mousemove_on_next_draw = true;
    };
    this.config = config;
    this.setList(config.list);
  }

  setList(list: T[]) {
    this.config.list = list;
    this.list_coordinates = [];
    for (const _ of this.config.list) {
      this.list_coordinates.push({
        x: 0,
        y: 0,
      });
    }
  }

  protected getList(): T[] {
    return this.config.list;
  }

  isHovering() {
    if (this.disabled) {
      return false;
    }
    return this.config.scrollbar.isHovering();
  }

  setHovering(hovering: boolean) {
    this.config.scrollbar.setHovering(hovering);
  }

  isClicking() {
    if (this.disabled) {
      return false;
    }
    return this.config.scrollbar.isClicking();
  }

  setClicking(clicking: boolean) {
    this.config.scrollbar.setClicking(clicking);
  }

  setAllSizes(size: number, p: Point2D, w: number, h: number) {
    const title_size = this.config.title?.size ?? 0;
    this.config.scrollbar.setAllSizes(size, { x: p.x, y: p.y + title_size }, w, h - title_size);
  }

  getPadding(): number {
    return this.config.padding ?? 0;
  }

  getGap(): number {
    return this.config.gap ?? 0;
  }

  disable() {
    this.disabled = true;
    this.config.scrollbar.disable();
  }

  enable() {
    this.disabled = false;
    this.config.scrollbar.enable();
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
      let [xi, yi, value, padding] = [
        this.xi(),
        this.config.scrollbar.yi(),
        this.config.scrollbar.value(),
        this.getPadding(),
      ];
      ctx.save();
      ctx.beginPath();
      ctx.rect(
        this.xi() + padding,
        this.config.scrollbar.yi() + padding,
        this.w() - padding * 2,
        this.config.scrollbar.h() - padding * 2
      );
      ctx.clip();
      yi += padding - value;
      ctx.translate(xi + padding, yi);
      let found_start = false;
      let found_end = false;
      const initial_yi = yi;
      for (const [i, el] of this.config.list.entries()) {
        let draw_el = true;
        if (!found_start) {
          if (yi + el.h() >= this.config.scrollbar.yi() + padding) {
            this.draw_start = i;
            found_start = true;
          } else {
            draw_el = false;
          }
        } else if (!found_end) {
          if (yi + el.h() >= this.config.scrollbar.yf() - padding) {
            this.draw_end = i;
            found_end = true;
          }
        } else {
          draw_el = false;
        }
        if (draw_el) {
          el.draw(ctx, transform, dt);
          if (this.config.draw_config.fixed_position) {
            this.list_coordinates[i] = {
              x: transform.view.x + xi + padding,
              y: transform.view.y + yi,
            };
          } else {
            this.list_coordinates[i] = {
              x: xi + padding,
              y: yi,
            };
          }
        }
        const yi_adjust = el.h() + (i + 1 < this.config.list.length ? this.getGap() : 0);
        yi += yi_adjust;
        ctx.translate(0, yi_adjust);
      }
      ctx.translate(-xi - padding, -yi);
      ctx.restore();
      this.scrollbar_adjust_time -= dt;
      if (this.scrollbar_adjust_time < 0) {
        this.scrollbar_adjust_time = DwgListbox.SCROLLBAR_ADJUST_TIME;
        this.config.scrollbar.setValue({
          value,
          value_min: 0,
          value_max: yi >= this.yf() - padding ? yi - initial_yi - this.h() + 2 * padding : 0,
        });
      }
    });
    this.config.scrollbar.draw(ctx, transform, dt);
    if (this.check_mousemove_on_next_draw) {
      const last_m = this.config.scrollbar.getLastMousemoveM();
      const last_transform = this.config.scrollbar.getLastMousemoveTransform();
      if (!!last_m && !!last_transform) {
        this.mousemove(last_m, last_transform);
      }
    }
    if (this.disabled) {
      configDraw(ctx, transform, {
        ...this.config.draw_config,
        fill_style: 'rgba(255, 255, 255, 0.4)',
      }, false, false, ()=> {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
      });
    }
  }

  scroll(dy: number, mode: number, _dx?: number): boolean {
    return this.config.scrollbar.scroll(dy, mode, _dx);
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    this.config.scrollbar.mousemove(m, transform);
    const is_hovering = this.isHovering();
    for (const [i, el] of this.config.list.entries()) {
      if (this.disabled || !is_hovering || i < this.draw_start || i > this.draw_end) {
        el.setHovering(false);
      } else {
        el.mousemove(m, {
          view: { ...this.list_coordinates[i] },
          scale: this.config.draw_config.fixed_position ? transform.scale : 1,
        });
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

  mouseOver(m: Point2D, transform: BoardTransformData): boolean {
    return this.config.scrollbar.mouseOver(m, transform);
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
