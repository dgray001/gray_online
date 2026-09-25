import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { screenToCanvas } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import type { RectScrollbarConfig } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import { DwgRectScrollbar } from '../../../../util/canvas_components/scrollbar/rect_scrollbar';
import { drawLine, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';

/** A bottom-panel item must be independently positionable so the panel can lay it out in a column */
export declare interface BottomPanelItem extends CanvasComponent {
  setPosition(p: Point2D): void;
  dataRefreshed?(): void;
  drawTooltip?(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void;
}

/** A group of items stacked in a column; groups are laid out left-to-right with a separator between them */
export type BottomPanelGroup = BottomPanelItem[];

export declare interface BottomPanelConfig {
  background: string;
  left_panel_w: number;
  right_panel_w: number;
}

class RisqBottomPanelScrollbar extends DwgRectScrollbar {
  scrollCallback(_value: number): void {}
}

/** Centered, content-sized bottom bar that stays clear of the left/right panels, scrolling horizontally on overflow */
export class RisqBottomPanel implements CanvasComponent {
  private static PADDING = 8;
  private static GAP = 8;
  private static BOTTOM_MARGIN = 12;
  private static SCROLLBAR_SIZE = 8;

  private risq: DwgRisq;
  private config: BottomPanelConfig;
  private groups: BottomPanelGroup[];
  private items: BottomPanelItem[];
  private group_widths: number[] = [];
  private separator_xs: number[] = [];
  private scrollbar: RisqBottomPanelScrollbar;
  private hovering = false;
  private p: Point2D = { x: 0, y: 0 };
  private panel_w = 0;
  private panel_h = 0;
  private content_w = 0;
  private content_h = 0;
  private overflowing = false;

  constructor(risq: DwgRisq, config: BottomPanelConfig, groups: BottomPanelGroup[]) {
    this.risq = risq;
    this.config = config;
    this.groups = groups;
    this.items = groups.flat();
    this.scrollbar = new RisqBottomPanelScrollbar({
      scrollbar_config: { value: { value: 0, value_min: 0, value_max: 0 }, step_size: 30, scroll_pixel_constant: 1 },
      p: { x: 0, y: 0 },
      w: 0,
      h: 0,
      scrollbar_size: RisqBottomPanel.SCROLLBAR_SIZE,
      min_bar_size: 1.5 * RisqBottomPanel.SCROLLBAR_SIZE,
      bar_dif_size: 1,
      vertical: false,
      draw_config: {
        fill_style: 'rgba(255, 255, 255, 0.4)',
        stroke_width: 0,
        hover_fill_style: 'rgba(255, 255, 255, 0.6)',
        click_fill_style: 'rgba(255, 255, 255, 0.8)',
        fixed_position: true,
      },
      space_draw_config: {
        fill_style: 'rgba(0, 0, 0, 0.15)',
        stroke_width: 0,
        hover_fill_style: 'rgba(0, 0, 0, 0.25)',
        click_fill_style: 'rgba(0, 0, 0, 0.35)',
        fixed_position: true,
        stroke_matches_fill_style: true,
      },
      background_color: 'rgba(0, 0, 0, 0.1)',
      arrow_scroll_amount: 30,
      space_scroll_amount: 100,
    } satisfies RectScrollbarConfig);
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }
  isClicking(): boolean {
    return this.scrollbar.isClicking();
  }
  setClicking(clicking: boolean): void {
    this.scrollbar.setClicking(clicking);
  }

  private groupHeight(group: BottomPanelGroup): number {
    return group.reduce((sum, item) => sum + item.h(), 0) + RisqBottomPanel.GAP * Math.max(group.length - 1, 0);
  }

  private recomputeLayout() {
    this.group_widths = this.groups.map((group) => Math.max(0, ...group.map((item) => item.w())));
    this.content_h = Math.max(0, ...this.groups.map((group) => this.groupHeight(group)));
    this.content_w =
      this.group_widths.reduce((sum, w) => sum + w, 0) + RisqBottomPanel.GAP * Math.max(this.groups.length - 1, 0);
    const canvas_w = this.risq.canvasSize().width;
    const max_side_panel_w = Math.max(this.config.left_panel_w, this.config.right_panel_w);
    const available_w = Math.max(canvas_w - 2 * (max_side_panel_w + RisqBottomPanel.PADDING), 0);
    this.overflowing = this.content_w + 2 * RisqBottomPanel.PADDING > available_w;
    this.panel_w = Math.min(this.content_w + 2 * RisqBottomPanel.PADDING, available_w);
    this.panel_h =
      this.content_h +
      2 * RisqBottomPanel.PADDING +
      (this.overflowing ? RisqBottomPanel.SCROLLBAR_SIZE + RisqBottomPanel.PADDING : 0);
    this.p = {
      x: 0.5 * (canvas_w - this.panel_w),
      y: this.risq.canvasSize().height - this.panel_h - RisqBottomPanel.BOTTOM_MARGIN,
    };
    const max_scroll = this.overflowing ? this.content_w - (this.panel_w - 2 * RisqBottomPanel.PADDING) : 0;
    this.scrollbar.setValue({
      value: Math.min(this.scrollbar.value(), max_scroll),
      value_min: 0,
      value_max: max_scroll,
    });
    if (this.overflowing) {
      this.scrollbar.setAllSizes(
        RisqBottomPanel.SCROLLBAR_SIZE,
        {
          x: this.xi() + RisqBottomPanel.PADDING,
          y: this.yi(),
        },
        this.panel_w - 2 * RisqBottomPanel.PADDING,
        this.panel_h - RisqBottomPanel.PADDING
      );
    }
    this.separator_xs = [];
    let x = this.xi() + RisqBottomPanel.PADDING - this.scrollbar.value();
    for (const [i, group] of this.groups.entries()) {
      const group_w = this.group_widths[i];
      const group_h = this.groupHeight(group);
      let y = this.yi() + RisqBottomPanel.PADDING + 0.5 * (this.content_h - group_h);
      for (const item of group) {
        item.setPosition({ x: x + 0.5 * (group_w - item.w()), y });
        y += item.h() + RisqBottomPanel.GAP;
      }
      x += group_w + RisqBottomPanel.GAP;
      if (i < this.groups.length - 1) {
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
    // clip in screen space (via screenToCanvas) since ctx is currently in world space, not inside a fixed_position block
    const clip_rect = {
      x: this.xi() + RisqBottomPanel.PADDING,
      y: this.yi() + RisqBottomPanel.PADDING,
      w: this.panel_w - 2 * RisqBottomPanel.PADDING,
      h: this.content_h,
    };
    ctx.save();
    const corners = [
      { x: clip_rect.x, y: clip_rect.y },
      { x: clip_rect.x + clip_rect.w, y: clip_rect.y },
      { x: clip_rect.x + clip_rect.w, y: clip_rect.y + clip_rect.h },
      { x: clip_rect.x, y: clip_rect.y + clip_rect.h },
    ].map((corner) => screenToCanvas(corner, transform));
    ctx.beginPath();
    corners.forEach((corner, i) => (i === 0 ? ctx.moveTo(corner.x, corner.y) : ctx.lineTo(corner.x, corner.y)));
    ctx.closePath();
    ctx.clip();
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
    ctx.lineWidth = 1;
    for (const separator_x of this.separator_xs) {
      drawLine(ctx, { x: separator_x, y: clip_rect.y }, { x: separator_x, y: clip_rect.y + clip_rect.h });
    }
    for (const item of this.items) {
      item.draw(ctx, transform, dt);
    }
    ctx.restore();
    if (this.overflowing) {
      this.scrollbar.draw(ctx, transform, dt);
    }
    for (const item of this.items) {
      item.drawTooltip?.(ctx, transform, this.risq, dt);
    }
  }

  scroll(dy: number, mode: number): boolean {
    if (!this.overflowing) {
      return false;
    }
    return this.scrollbar.scroll(dy, mode);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    const scrollbar_hovering = this.overflowing && this.scrollbar.mousemove(canvas, screen, transform);
    const item_hovering = this.items.map((item) => item.mousemove(canvas, screen, transform)).some(Boolean);
    this.hovering =
      scrollbar_hovering ||
      item_hovering ||
      (screen.x >= this.xi() && screen.y >= this.yi() && screen.x <= this.xf() && screen.y <= this.yf());
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.overflowing && this.scrollbar.mousedown(e)) {
      return true;
    }
    return this.items.map((item) => item.mousedown(e)).some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    this.scrollbar.mouseup(e);
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
