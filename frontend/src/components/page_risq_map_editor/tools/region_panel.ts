import type { BoardTransformData } from '../../game/util/canvas_board/canvas_board';
import { configDraw } from '../../game/util/canvas_components/canvas_component';
import { drawRect, drawText } from '../../game/util/canvas_util';
import type { Point2D } from '../../game/util/objects2d';
import type { EditorConfigPanel, EditorToolContext, PanelBounds } from '../editor_tool';
import type { MapDocRegion } from '../map_doc';

const PADDING = 6;
const GAP = 4;
const ROW_H = 24;
const INK = 'rgb(59, 36, 19)';
const FONT = '12px "Fira Sans", sans-serif';
const BOLD_FONT = 'bold 12px "Fira Sans", sans-serif';

function contains(rect: PanelBounds, p: Point2D): boolean {
  return p.x >= rect.x && p.x <= rect.x + rect.w && p.y >= rect.y && p.y <= rect.y + rect.h;
}
export class RegionPanel implements EditorConfigPanel {
  private bounds: PanelBounds = { x: 0, y: 0, w: 0, h: 0 };
  private mouse: Point2D = { x: 0, y: 0 };
  private active_index = 0;
  private scroll_offset = 0;

  constructor(
    private context: EditorToolContext,
    private on_add_region: () => void,
    private on_delete_region: (index: number) => void,
    private on_change_bonus: (index: number, delta: number) => void
  ) {}

  activeIndex(): number {
    return this.active_index;
  }

  activeRegion(): MapDocRegion | undefined {
    const doc = this.context.doc();
    return doc?.regions?.[this.active_index];
  }
  setBounds(bounds: PanelBounds): void {
    this.bounds = bounds;
  }

  mousemove(_canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.mouse = screen;
    return contains(this.bounds, screen);
  }

  isHovering(): boolean {
    return contains(this.bounds, this.mouse);
  }

  setHovering(_hovering: boolean): void {}
  isClicking(): boolean {
    return false;
  }
  setClicking(_clicking: boolean): void {}

  scroll(dy: number): boolean {
    if (!contains(this.bounds, this.mouse)) {
      return false;
    }
    this.scroll_offset = Math.max(0, this.scroll_offset + dy);
    return true;
  }
  mousedown(e: MouseEvent): boolean {
    if (e.button !== 0 || !contains(this.bounds, this.mouse)) {
      return false;
    }
    const x = this.bounds.x + PADDING;
    let y = this.bounds.y + PADDING + ROW_H + GAP;
    if (contains({ x, y, w: 70, h: ROW_H }, this.mouse)) {
      this.on_add_region();
      return true;
    }
    if (contains({ x: x + 76, y, w: 70, h: ROW_H }, this.mouse)) {
      this.on_delete_region(this.active_index);
      return true;
    }
    y += ROW_H + GAP;
    if (contains({ x: x + 90, y, w: 24, h: ROW_H }, this.mouse)) {
      this.on_change_bonus(this.active_index, -1);
      return true;
    }
    if (contains({ x: x + 120, y, w: 24, h: ROW_H }, this.mouse)) {
      this.on_change_bonus(this.active_index, 1);
      return true;
    }
    return this.handleRegionListClick(y + ROW_H + GAP);
  }

  mouseup(_e: MouseEvent): void {}
  private handleRegionListClick(start_y: number): boolean {
    const doc = this.context.doc();
    const regions = doc?.regions ?? [];
    for (let i = 0; i < regions.length; i++) {
      const rect: PanelBounds = {
        x: this.bounds.x + PADDING,
        y: start_y + i * (ROW_H + GAP) - this.scroll_offset,
        w: this.bounds.w - 2 * PADDING,
        h: ROW_H,
      };
      if (contains(rect, this.mouse)) {
        this.active_index = i;
        return true;
      }
    }
    return false;
  }
  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    if (this.bounds.w <= 0 || this.bounds.h <= 0) {
      return;
    }
    const draw_config = { fill_style: 'transparent', stroke_width: 1, fixed_position: true };
    configDraw(ctx, transform, draw_config, false, false, () => {
      const doc = this.context.doc();
      const x = this.bounds.x + PADDING;
      let y = this.bounds.y + PADDING;
      drawText(ctx, 'Regions', { p: { x, y: y + 10 }, w: this.bounds.w, fill_style: INK, font: BOLD_FONT });
      y += ROW_H + GAP;
      this.drawButton(ctx, '+ New', { x, y, w: 70, h: ROW_H });
      this.drawButton(ctx, '- Del', { x: x + 76, y, w: 70, h: ROW_H });
      y += ROW_H + GAP;
      const current = this.activeRegion();
      const bonus = current?.gold_bonus ?? 0;
      drawText(ctx, `Gold bonus: ${bonus}`, { p: { x, y: y + 10 }, w: 90, fill_style: INK, font: FONT });
      this.drawButton(ctx, '-', { x: x + 90, y, w: 24, h: ROW_H });
      this.drawButton(ctx, '+', { x: x + 120, y, w: 24, h: ROW_H });
      y += ROW_H + GAP * 2;
      this.drawRegions(ctx, doc?.regions ?? [], y);
    });
  }
  private drawRegions(ctx: CanvasRenderingContext2D, regions: MapDocRegion[], start_y: number): void {
    regions.forEach((r, i) => {
      const rect: PanelBounds = {
        x: this.bounds.x + PADDING,
        y: start_y + i * (ROW_H + GAP) - this.scroll_offset,
        w: this.bounds.w - 2 * PADDING,
        h: ROW_H,
      };
      if (rect.y + rect.h < this.bounds.y || rect.y > this.bounds.y + this.bounds.h) {
        return;
      }
      const active = this.active_index === i;
      ctx.fillStyle = active
        ? 'rgb(200, 160, 120)'
        : contains(rect, this.mouse)
          ? 'rgb(220, 200, 180)'
          : 'rgb(240, 230, 215)';
      drawRect(ctx, rect, rect.w, rect.h, 3);
      drawText(ctx, `${r.name} (${r.spaces.length} spaces)`, {
        p: { x: rect.x + 8, y: rect.y + rect.h / 2 },
        w: rect.w - 16,
        fill_style: INK,
        font: FONT,
        baseline: 'middle',
      });
    });
  }

  private drawButton(ctx: CanvasRenderingContext2D, label: string, rect: PanelBounds): void {
    ctx.fillStyle = contains(rect, this.mouse) ? 'rgb(220, 200, 180)' : 'rgb(240, 230, 215)';
    drawRect(ctx, rect, rect.w, rect.h, 3);
    drawText(ctx, label, {
      p: { x: rect.x + rect.w / 2, y: rect.y + rect.h / 2 },
      w: rect.w,
      fill_style: INK,
      font: FONT,
      align: 'center',
      baseline: 'middle',
    });
  }

  xi(): number {
    return this.bounds.x;
  }
  yi(): number {
    return this.bounds.y;
  }
  xf(): number {
    return this.bounds.x + this.bounds.w;
  }
  yf(): number {
    return this.bounds.y + this.bounds.h;
  }
  w(): number {
    return this.bounds.w;
  }
  h(): number {
    return this.bounds.h;
  }
}
