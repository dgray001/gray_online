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
}

export class DwgListbox<
  T extends CanvasComponent = CanvasComponent,
  R extends DwgRectScrollbar = DwgRectScrollbar,
> implements CanvasComponent {
  private config: ListboxConfig<T, R>;
  // Index of the first list item shown
  private draw_start = 0;
  // Index of the last list item shown
  private draw_end = 0;

  constructor(config: ListboxConfig<T, R>) {
    this.config = config;
    console.log('!! ?', config.draw_config);
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

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(ctx, transform, this.config.draw_config, this.isHovering(), this.isClicking(), () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
    });
    this.config.scrollbar.draw(ctx, transform, dt);
  }

  scroll(dy: number, mode: number, _dx?: number): boolean {
    return this.config.scrollbar.scroll(dy, mode, _dx);
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    for (const [i, item] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        item.setHovering(false);
      } else {
        item.mousemove(m, transform);
      }
    }
    this.config.scrollbar.mousemove(m, transform);
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    for (const [i, item] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        continue;
      } else {
        item.mousedown(e);
      }
    }
    this.config.scrollbar.mousedown(e);
    return this.isClicking();
  }

  mouseup(e: MouseEvent): void {
    for (const [i, item] of this.config.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        item.setClicking(false);
      } else {
        item.mouseup(e);
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
