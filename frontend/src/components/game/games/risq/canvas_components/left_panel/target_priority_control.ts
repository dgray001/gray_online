import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { RisqTargetCategory } from '../../risq_data';

const POOL_CATEGORIES = [RisqTargetCategory.ECONOMIC, RisqTargetCategory.MILITARY, RisqTargetCategory.BUILDING];
const PADDING = 4;
const CONTAINER_GAP = 8;
const DRAG_THRESHOLD = 4;

function categoryIcon(category: RisqTargetCategory): string {
  switch (category) {
    case RisqTargetCategory.ECONOMIC:
      return 'icons/villager64';
    case RisqTargetCategory.MILITARY:
      return 'icons/unit64';
    case RisqTargetCategory.BUILDING:
      return 'icons/building64';
    default:
      return '';
  }
}

interface CategoryRect {
  category: RisqTargetCategory;
  x: number;
  y: number;
  w: number;
  h: number;
}

interface ControlLayout {
  pool: CategoryRect[];
  container: { x: number; y: number; w: number; h: number };
  chips: CategoryRect[];
}

/** Lets the player rank target categories: click a pool square to prioritize it, click a chip to unprioritize it, drag a chip to reorder */
export class RisqTargetPriorityControl {
  private risq: DwgRisq;
  private unit_internal_ids: number[];
  private p: Point2D = { x: 0, y: 0 };
  private control_w = 0;
  private square_size = 0;
  private priority: RisqTargetCategory[] = [];
  private last_mouse: Point2D = { x: 0, y: 0 };
  private hovering_category?: RisqTargetCategory;
  private pressed_pool_category?: RisqTargetCategory;
  private dragging_category?: RisqTargetCategory;
  private drag_order?: RisqTargetCategory[];
  private drag_start_x = 0;
  private drag_offset_x = 0;
  private dragged = false;

  constructor(risq: DwgRisq, unit_internal_ids: number[]) {
    this.risq = risq;
    this.unit_internal_ids = unit_internal_ids;
  }

  setUnitIds(unit_internal_ids: number[]) {
    this.unit_internal_ids = unit_internal_ids;
  }

  setPosition(p: Point2D) {
    this.p = p;
  }

  setSize(w: number, h: number) {
    this.control_w = w;
    this.square_size = h - 2 * PADDING;
  }

  dataRefreshed() {
    if (this.dragging_category !== undefined) {
      return;
    }
    const player = this.risq.getPlayer();
    this.priority = player?.units.get(this.unit_internal_ids[0])?.target_priority ?? [];
  }

  private poolCategories(): RisqTargetCategory[] {
    return POOL_CATEGORIES.filter((c) => !this.priority.includes(c));
  }

  private computeLayout(): ControlLayout {
    const s = this.square_size;
    const chip_y = this.p.y + PADDING;
    const pool_categories = this.poolCategories();
    const pool: CategoryRect[] = pool_categories.map((category, i) => ({
      category,
      x: this.p.x + i * (s + PADDING),
      y: chip_y,
      w: s,
      h: s,
    }));
    const pool_w = pool_categories.length > 0 ? pool_categories.length * (s + PADDING) - PADDING : 0;
    const container_x = this.p.x + pool_w + (pool_categories.length > 0 ? CONTAINER_GAP : 0);
    const chip_categories = this.dragging_category !== undefined && this.drag_order ? this.drag_order : this.priority;
    const container_w = 2 * PADDING + (chip_categories.length > 0 ? chip_categories.length * (s + PADDING) - PADDING : 0);
    const chips: CategoryRect[] = chip_categories.map((category, i) => ({
      category,
      x: container_x + PADDING + i * (s + PADDING),
      y: chip_y,
      w: s,
      h: s,
    }));
    return { pool, container: { x: container_x, y: this.p.y, w: container_w, h: s + 2 * PADDING }, chips };
  }

  private pointIn(m: Point2D, r: CategoryRect): boolean {
    return m.x >= r.x && m.x <= r.x + r.w && m.y >= r.y && m.y <= r.y + r.h;
  }

  private updateDragOrder() {
    if (this.dragging_category === undefined || !this.drag_order) {
      return;
    }
    const layout = this.computeLayout();
    const dragged_center = this.clampedDragX(layout.container) + this.square_size / 2;
    const step = this.square_size + PADDING;
    const slot_center = (i: number) => layout.container.x + PADDING + i * step + this.square_size / 2;
    const current_index = this.drag_order.indexOf(this.dragging_category);
    const own_center = slot_center(current_index);
    let new_index = current_index;
    if (current_index < this.drag_order.length - 1 && dragged_center > (own_center + slot_center(current_index + 1)) / 2) {
      new_index = current_index + 1;
    } else if (current_index > 0 && dragged_center < (own_center + slot_center(current_index - 1)) / 2) {
      new_index = current_index - 1;
    }
    if (new_index !== current_index) {
      const others = this.drag_order.filter((c) => c !== this.dragging_category);
      others.splice(new_index, 0, this.dragging_category);
      this.drag_order = others;
    }
  }

  private sendPriority(priority: RisqTargetCategory[]) {
    this.risq.setUnitTargetPriority(this.unit_internal_ids, priority);
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    configDraw(ctx, transform, { fill_style: 'transparent', stroke_width: 0, fixed_position: true }, false, false, () => {
      const layout = this.computeLayout();
      for (const rect of layout.pool) {
        this.drawSquare(ctx, rect);
      }
      ctx.fillStyle = 'transparent';
      ctx.strokeStyle = 'rgba(60, 40, 20, 0.8)';
      ctx.lineWidth = 1;
      drawRect(
        ctx,
        { x: layout.container.x, y: layout.container.y },
        layout.container.w,
        layout.container.h,
        0.3 * layout.container.h
      );
      for (const rect of layout.chips) {
        const dragging_this = this.dragged && rect.category === this.dragging_category;
        this.drawSquare(ctx, dragging_this ? { ...rect, x: this.clampedDragX(layout.container) } : rect);
      }
    });
  }

  private clampedDragX(container: { x: number; w: number }): number {
    const min_x = container.x + PADDING;
    const max_x = container.x + container.w - PADDING - this.square_size;
    return Math.max(min_x, Math.min(max_x, this.last_mouse.x - this.drag_offset_x));
  }

  private drawSquare(ctx: CanvasRenderingContext2D, rect: CategoryRect) {
    const dragging = rect.category === this.dragging_category && this.dragged;
    const hovered = rect.category === this.hovering_category;
    ctx.fillStyle = dragging
      ? 'rgba(210, 180, 130, 0.9)'
      : hovered
        ? 'rgba(210, 180, 130, 0.5)'
        : 'rgba(60, 40, 20, 0.15)';
    ctx.strokeStyle = 'rgba(60, 40, 20, 0.6)';
    ctx.lineWidth = 1;
    drawRect(ctx, { x: rect.x, y: rect.y }, rect.w, rect.h, 3);
    ctx.drawImage(this.risq.getIcon(categoryIcon(rect.category)), rect.x + 2, rect.y + 2, rect.w - 4, rect.h - 4);
  }

  mousemove(m: Point2D): boolean {
    this.last_mouse = m;
    if (this.dragging_category !== undefined) {
      if (!this.dragged && Math.abs(m.x - this.drag_start_x) > DRAG_THRESHOLD) {
        this.dragged = true;
      }
      if (this.dragged) {
        this.updateDragOrder();
      }
      return true;
    }
    const layout = this.computeLayout();
    const hit = [...layout.pool, ...layout.chips].find((r) => this.pointIn(m, r));
    this.hovering_category = hit?.category;
    return !!hit;
  }

  mousedown(_e: MouseEvent): boolean {
    const layout = this.computeLayout();
    const chip = layout.chips.find((r) => this.pointIn(this.last_mouse, r));
    if (chip) {
      this.dragging_category = chip.category;
      this.drag_order = [...this.priority];
      this.drag_start_x = this.last_mouse.x;
      this.drag_offset_x = this.last_mouse.x - chip.x;
      this.dragged = false;
      return true;
    }
    const pool = layout.pool.find((r) => this.pointIn(this.last_mouse, r));
    if (pool) {
      this.pressed_pool_category = pool.category;
      return true;
    }
    return false;
  }

  mouseup(_e: MouseEvent): void {
    if (this.dragging_category !== undefined) {
      if (this.dragged && this.drag_order) {
        this.priority = this.drag_order;
        this.sendPriority(this.drag_order);
      } else {
        this.sendPriority(this.priority.filter((c) => c !== this.dragging_category));
      }
      this.dragging_category = undefined;
      this.drag_order = undefined;
      this.dragged = false;
      return;
    }
    if (this.pressed_pool_category !== undefined) {
      const layout = this.computeLayout();
      const still_hovering = layout.pool.some(
        (r) => r.category === this.pressed_pool_category && this.pointIn(this.last_mouse, r)
      );
      if (still_hovering) {
        this.sendPriority([...this.priority, this.pressed_pool_category]);
      }
      this.pressed_pool_category = undefined;
    }
  }
}
