import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawRect } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import { configDraw, type CanvasComponent, type DrawConfig } from '../canvas_component';
import type { DwgRectScrollbar } from './rect_scrollbar';

/** Data describing a listbox config */
export declare interface ListboxConfig<T extends CanvasComponent, R extends DwgRectScrollbar = DwgRectScrollbar> {
  list: T[];
  scrollbar: R;
  draw_config: DrawConfig;
  gap?: number;
  padding?: number;
  // TODO: implement title (need to adjust how we determine scrollable area)
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

  constructor(config: ListboxConfig<T, R>) {
    config.scrollbar.setValue({ value: 0, value_min: 0, value_max: 0 });
    this.config = config;
  }

  isHovering() {
    return this.config.scrollbar.isHovering();
  }

  setHovering(hovering: boolean) {
    this.config.scrollbar.setHovering(hovering);
  }

  isClicking() {
    return this.config.scrollbar.isClicking();
  }

  setClicking(clicking: boolean) {
    this.config.scrollbar.setClicking(clicking);
  }

  setAllSizes(size: number, p: Point2D, w: number, h: number) {
    this.config.scrollbar.setAllSizes(size, p, w, h);
  }

  getPadding(): number {
    return this.config.padding ?? 0;
  }

  getGap(): number {
    return this.config.gap ?? 0;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(ctx, transform, this.config.draw_config, this.isHovering(), this.isClicking(), () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
      let [xi, yi, value, padding] = [this.xi(), this.yi(), this.config.scrollbar.value(), this.getPadding()];
      ctx.save();
      ctx.beginPath();
      ctx.rect(
        this.xi() + padding, 
        this.yi() + padding, 
        this.w() - (padding * 2), 
        this.h() - (padding * 2)
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
          if (yi + el.h() >= this.yi() + padding) {
            this.draw_start = i;
            found_start = true;
          } else {
            draw_el = false;
          }
        } else if (!found_end) {
          if (yi + el.h() >= this.yf() - padding) {
            this.draw_end = i;
            found_end = true;
          }
        } else {
          draw_el = false;
        }
        if (draw_el) {
          el.draw(ctx, transform, dt);
        }
        const yi_adjust = el.h() + ((i + 1 < this.config.list.length) ? this.getGap() : 0);
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
          value_max: yi - initial_yi - this.h() + 2 * padding,
        });
      }
    });
    this.config.scrollbar.draw(ctx, transform, dt);
  }

  scroll(dy: number, mode: number, _dx?: number): boolean {
    return this.config.scrollbar.scroll(dy, mode, _dx);
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    for (const [i, el] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        el.setHovering(false);
      } else {
        el.mousemove(m, transform);
      }
    }
    this.config.scrollbar.mousemove(m, transform);
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    for (const [i, el] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        continue;
      } else {
        el.mousedown(e);
      }
    }
    this.config.scrollbar.mousedown(e);
    return this.isClicking();
  }

  mouseup(e: MouseEvent): void {
    for (const [i, el] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
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
    return this.config.scrollbar.yi();
  }

  yf(): number {
    return this.config.scrollbar.yf();
  }

  w(): number {
    return this.config.scrollbar.w();
  }

  h(): number {
    return this.config.scrollbar.h();
  }
}
