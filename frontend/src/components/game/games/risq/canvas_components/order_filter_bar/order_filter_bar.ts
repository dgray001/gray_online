import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import type {
  DropdownBounds,
  MultiSelectOption,
} from '../../../../util/canvas_components/dropdown/multi_select_dropdown';
import { DwgMultiSelectDropdown } from '../../../../util/canvas_components/dropdown/multi_select_dropdown';
import type { Point2D } from '../../../../util/objects2d';
import {
  ORDER_HIDE_OPTIONS,
  ORDER_SUBJECT_OPTIONS,
  ORDER_TYPE_OPTIONS,
  RisqOrderFilter,
} from '../../application/orders/order_filter';
import type { DwgRisq } from '../../risq';

const GAP = 4;
export const ORDER_FILTER_BAR_H = 24;

/** Multi-select dropdowns that edit a RisqOrderFilter */
export class RisqOrderFilterBar implements CanvasComponent {
  readonly filter = new RisqOrderFilter();
  private dropdowns: DwgMultiSelectDropdown[];
  private p: Point2D = { x: 0, y: 0 };
  private width = 0;

  constructor(
    private risq: DwgRisq,
    private bounds: () => DropdownBounds,
    private on_change: () => void,
    show_subject_filter: boolean = true
  ) {
    this.dropdowns = [
      this.createDropdown('Order type', ORDER_TYPE_OPTIONS, this.filter.type_groups),
      this.createDropdown('Hide selected', ORDER_HIDE_OPTIONS, this.filter.hidden),
    ];
    if (show_subject_filter) {
      this.dropdowns.unshift(this.createDropdown('Subject type', ORDER_SUBJECT_OPTIONS, this.filter.subject_classes));
    }
  }

  private createDropdown(title: string, options: MultiSelectOption[], selected: Set<number>): DwgMultiSelectDropdown {
    return new DwgMultiSelectDropdown({
      title,
      options,
      selected,
      onChange: () => this.on_change(),
      getIcon: (path) => this.risq.getIcon(path),
      bounds: () => this.bounds(),
    });
  }

  setPosition(p: Point2D): void {
    this.p = p;
    this.layout();
  }

  setW(w: number): void {
    this.width = w;
    this.layout();
  }

  private layout(): void {
    const button_w = (this.width - (this.dropdowns.length - 1) * GAP) / this.dropdowns.length;
    for (const [i, dropdown] of this.dropdowns.entries()) {
      dropdown.setPosition({ x: this.p.x + i * (button_w + GAP), y: this.p.y });
      dropdown.setSize(button_w, ORDER_FILTER_BAR_H);
    }
  }

  isAnyOpen(): boolean {
    return this.dropdowns.some((dropdown) => dropdown.isOpen());
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    for (const dropdown of this.dropdowns) {
      dropdown.draw(ctx, transform, dt);
    }
  }

  scroll(dy: number): boolean {
    return this.dropdowns.map((dropdown) => dropdown.scroll(dy)).some(Boolean);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    return this.dropdowns.map((dropdown) => dropdown.mousemove(canvas, screen, transform)).some(Boolean);
  }

  mousedown(e: MouseEvent): boolean {
    return this.dropdowns.map((dropdown) => dropdown.mousedown(e)).some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    for (const dropdown of this.dropdowns) {
      dropdown.mouseup(e);
    }
  }

  isHovering(): boolean {
    return this.dropdowns.some((dropdown) => dropdown.isHovering());
  }

  setHovering(hovering: boolean): void {
    this.dropdowns.forEach((dropdown) => dropdown.setHovering(hovering));
  }

  isClicking(): boolean {
    return this.dropdowns.some((dropdown) => dropdown.isClicking());
  }

  setClicking(clicking: boolean): void {
    this.dropdowns.forEach((dropdown) => dropdown.setClicking(clicking));
  }

  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.p.x + this.width;
  }
  yf(): number {
    return this.p.y + ORDER_FILTER_BAR_H;
  }
  w(): number {
    return this.width;
  }
  h(): number {
    return ORDER_FILTER_BAR_H;
  }
}
