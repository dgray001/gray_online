import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';

/** A bottom-panel item must be independently positionable so the panel can lay it out in a column */
export declare interface BottomPanelItem extends CanvasComponent {
  readonly overflow?: number;
  setPosition(p: Point2D): void;
  dataRefreshed?(): void;
  setAvailableWidth?(width: number): void;
  setAvailableHeight?(height: number): void;
  drawTooltip?(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void;
}

/** A group of items stacked in a column; groups are laid out left-to-right with a separator between them */
export type BottomPanelGroup = BottomPanelItem[];

export declare interface BottomPanelConfig {
  background: string;
  left_panel_w: number;
  right_panel_w: number;
  center_item: BottomPanelItem;
}

/** Content-sized bottom bar anchored to the minimap's screen center */
export class RisqBottomPanel implements CanvasComponent {
  private static PADDING = 8;
  private static GAP = 8;
  private static BOTTOM_MARGIN = 12;

  private risq: DwgRisq;
  private config: BottomPanelConfig;
  private groups: BottomPanelGroup[];
  private items: BottomPanelItem[];
  private group_widths: number[] = [];
  private separator_xs: number[] = [];
  private hovering = false;
  private mouse_screen: Point2D = { x: -1, y: -1 };
  private p: Point2D = { x: 0, y: 0 };
  private panel_w = 0;
  private panel_h = 0;
  private content_w = 0;
  private content_h = 0;
  private layout_key = '';

  constructor(risq: DwgRisq, config: BottomPanelConfig, groups: BottomPanelGroup[]) {
    this.risq = risq;
    this.config = config;
    this.groups = groups;
    this.items = groups.flat();
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }
  isClicking(): boolean {
    return this.items.some((item) => item.isClicking());
  }
  setClicking(clicking: boolean): void {
    if (!clicking) {
      for (const item of this.items) {
        item.setClicking(false);
        item.setHovering(false);
      }
    }
  }

  private groupHeight(group: BottomPanelGroup): number {
    return group.reduce((sum, item) => sum + item.h(), 0) + RisqBottomPanel.GAP * Math.max(group.length - 1, 0);
  }

  private recomputeLayout(): void {
    const canvas_size = this.risq.viewport.canvasSize();
    const center = this.config.center_item;
    const center_index = this.groups.findIndex((group) => group.includes(center));
    const fixed_width = this.groups
      .slice(0, center_index)
      .filter((group) => !group.some((item) => item.setAvailableWidth))
      .reduce(
        (sum, group) =>
          sum + Math.max(0, ...group.map((item) => item.w() + 2 * (item.overflow ?? 0))) + RisqBottomPanel.GAP,
        0
      );
    const group_width =
      canvas_size.width / 2 -
      center.w() / 2 -
      (center.overflow ?? 0) -
      this.config.left_panel_w -
      fixed_width -
      2 * RisqBottomPanel.PADDING;
    const right_width =
      canvas_size.width / 2 -
      center.w() / 2 -
      (center.overflow ?? 0) -
      this.config.right_panel_w -
      2 * RisqBottomPanel.PADDING;
    for (const [i, group] of this.groups.entries()) {
      for (const item of group) {
        item.setAvailableWidth?.(i < center_index ? group_width : right_width);
      }
    }
    const available_height = Math.max(
      0,
      ...this.groups.map((group) => this.groupHeight(group.filter((item) => !item.setAvailableHeight)))
    );
    for (const item of this.items) {
      item.setAvailableHeight?.(available_height);
    }
    const key = JSON.stringify([canvas_size, this.items.map((item) => [item.w(), item.h()])]);
    if (key === this.layout_key) {
      return;
    }
    this.layout_key = key;
    const groups = this.groups.filter((group) => group.some((item) => item.w() > 0));
    this.group_widths = groups.map((group) => Math.max(0, ...group.map((item) => item.w() + 2 * (item.overflow ?? 0))));
    this.content_h = Math.max(0, ...groups.map((group) => this.groupHeight(group)));
    this.content_w =
      this.group_widths.reduce((sum, w) => sum + w, 0) + RisqBottomPanel.GAP * Math.max(groups.length - 1, 0);
    const canvas_w = this.risq.viewport.canvasSize().width;
    const center_group_index = groups.findIndex((group) => group.includes(center));
    const center_group = center_group_index < 0 ? groups.length : center_group_index;
    const center_offset =
      this.group_widths.slice(0, center_group).reduce((sum, w) => sum + w + RisqBottomPanel.GAP, 0) +
      0.5 * (this.group_widths[center_group] ?? 0);
    this.panel_w = this.content_w + 2 * RisqBottomPanel.PADDING;
    this.panel_h = this.content_h + 2 * RisqBottomPanel.PADDING;
    this.p = {
      x: canvas_w / 2 - center_offset - RisqBottomPanel.PADDING,
      y: this.risq.viewport.canvasSize().height - this.panel_h - RisqBottomPanel.BOTTOM_MARGIN,
    };
    this.separator_xs = [];
    let x = this.xi() + RisqBottomPanel.PADDING;
    for (const [i, group] of groups.entries()) {
      const group_w = this.group_widths[i];
      const group_h = this.groupHeight(group);
      let y = this.yi() + RisqBottomPanel.PADDING + 0.5 * (this.content_h - group_h);
      for (const item of group) {
        item.setPosition({ x: x + 0.5 * (group_w - item.w()), y });
        y += item.h() + RisqBottomPanel.GAP;
      }
      x += group_w + RisqBottomPanel.GAP;
      if (i < groups.length - 1) {
        this.separator_xs.push(x - 0.5 * RisqBottomPanel.GAP);
      }
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    for (const item of this.items) {
      item.dataRefreshed?.();
    }
    this.recomputeLayout();
    if (this.panel_w <= 0 || this.panel_h <= 0) {
      return;
    }
    configDraw(
      ctx,
      transform,
      {
        fill_style: this.config.background,
        stroke_style: 'rgba(255, 255, 255, 0.2)',
        stroke_width: 1,
        fixed_position: true,
      },
      false,
      false,
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.panel_w, this.panel_h, 0.5 * Math.min(this.panel_h, 32));
      }
    );
    const clip_rect = {
      x: this.xi() + RisqBottomPanel.PADDING,
      y: this.yi() + RisqBottomPanel.PADDING,
      w: this.panel_w - 2 * RisqBottomPanel.PADDING,
      h: this.content_h,
    };
    ctx.save();
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_width: 0, fixed_position: true },
      false,
      false,
      () => {
        ctx.roundRect(this.xi(), this.yi(), this.panel_w, this.panel_h, 0.5 * Math.min(this.panel_h, 32));
        ctx.clip();
      }
    );
    configDraw(
      ctx,
      transform,
      { fill_style: 'transparent', stroke_style: 'rgba(60, 60, 60, 0.7)', stroke_width: 1, fixed_position: true },
      false,
      false,
      () => {
        for (const separator_x of this.separator_xs) {
          drawLine(ctx, { x: separator_x, y: clip_rect.y }, { x: separator_x, y: clip_rect.y + clip_rect.h });
        }
      }
    );
    for (const item of this.items) {
      if (!this.itemContains(item, this.mouse_screen)) {
        item.setHovering(false);
        item.setClicking(false);
      }
      item.draw(ctx, transform, dt);
    }
    ctx.restore();
    for (const item of this.items) {
      if (this.itemContains(item, this.mouse_screen) && item.isHovering()) {
        item.drawTooltip?.(ctx, transform, this.risq, dt);
      }
    }
  }

  scroll(dy: number, mode: number): boolean {
    for (const item of this.items) {
      if (this.itemContains(item, this.mouse_screen) && item.scroll?.(dy, mode)) {
        return true;
      }
    }
    return false;
  }

  private contentContains(screen: Point2D): boolean {
    const radius = 0.5 * Math.min(this.panel_h, 32);
    const x = Math.max(this.xi() + radius, Math.min(screen.x, this.xf() - radius));
    const y = Math.max(this.yi() + radius, Math.min(screen.y, this.yf() - radius));
    return (
      screen.x >= this.xi() &&
      screen.x <= this.xf() &&
      screen.y >= this.yi() &&
      screen.y <= this.yf() &&
      Math.hypot(screen.x - x, screen.y - y) <= radius
    );
  }

  private itemContains(item: BottomPanelItem, screen: Point2D): boolean {
    return (
      this.contentContains(screen) &&
      screen.x >= item.xi() - (item.overflow ?? 0) &&
      screen.x <= item.xf() + (item.overflow ?? 0) &&
      screen.y >= item.yi() - (item.overflow ?? 0) &&
      screen.y <= item.yf() + (item.overflow ?? 0)
    );
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.mouse_screen = screen;
    const item_hovering = this.items
      .map((item) => {
        if (this.itemContains(item, screen)) {
          return item.mousemove(canvas, screen, transform);
        }
        item.setHovering(false);
        item.setClicking(false);
        return false;
      })
      .some(Boolean);
    this.hovering =
      item_hovering ||
      (screen.x >= this.xi() && screen.y >= this.yi() && screen.x <= this.xf() && screen.y <= this.yf());
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    return this.items.map((item) => this.itemContains(item, this.mouse_screen) && item.mousedown(e)).some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    for (const item of this.items) {
      item.mouseup(e);
    }
  }

  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.xi() + this.panel_w;
  }
  yf(): number {
    return this.yi() + this.panel_h;
  }
  w(): number {
    return this.panel_w;
  }
  h(): number {
    return this.panel_h;
  }
}
