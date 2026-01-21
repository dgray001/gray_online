import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';
import type { DwgScrollbar } from '../scrollbar/scrollbar';

/** Data describing a listbox config */
export declare interface ListboxConfig<T extends CanvasComponent, R extends DwgScrollbar = DwgScrollbar> {
  list: T[];
  scrollbar: R;
  // function to draw items; returns the size that item took to draw
  draw: () => number;
  // means the listbox can draw partial items
  pixel_listbox?: boolean;
}

export abstract class DwgListbox<T extends CanvasComponent, R extends DwgScrollbar = DwgScrollbar> implements CanvasComponent {
  private list: T[];
  private scrollbar: R;
  // Index of the first list item shown
  private draw_start = 0;
  // Index of the last list item shown
  private draw_end = 0;

  constructor(config: ListboxConfig<T, R>) {
    this.scrollbar = config.scrollbar;
    this.list = config.list;
  }

  isHovering() {
    return this.scrollbar.isHovering();
  }

  setHovering(hovering: boolean) {
    this.scrollbar.setHovering(hovering);
  }

  isClicking() {
    return this.scrollbar.isClicking();
  }

  setClicking(clicking: boolean) {
    this.scrollbar.setClicking(clicking);
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    this.drawBox(ctx, transform, dt);
    //
    this.scrollbar.draw(ctx, transform, dt);
  }

  abstract drawBox(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    for (const [i, item] of this.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        item.setHovering(false);
      } else {
        item.mousemove(m, transform);
      }
    }
    this.scrollbar.mousemove(m, transform);
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    for (const [i, item] of this.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        continue;
      } else {
        item.mousedown(e);
      }
    }
    this.scrollbar.mousedown(e);
    return this.isClicking();
  }

  mouseup(e: MouseEvent): void {
    for (const [i, item] of this.list.entries()) {
      if (i < this.draw_start || i > this.draw_end) {
        item.setClicking(false);
      } else {
        item.mouseup(e);
      }
    }
    this.scrollbar.mouseup(e);
  }

  abstract mouseOver(m: Point2D, transform: BoardTransformData): boolean;
  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
