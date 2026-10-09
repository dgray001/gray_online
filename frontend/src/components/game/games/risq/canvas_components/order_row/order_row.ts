import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import { createTooltipState, drawTooltip, shouldShowTooltip } from '../../../../util/canvas_components/tooltip';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqFrontendOrder } from '../../model/types';
import { UNIT_CLUSTER_ICON_SIZE, drawUnitTypeCluster, unitClusterIconKey } from '../../rendering/zones/draw';
import { unitImage } from '../../rendering/assets/unit';
import type { DwgRisq } from '../../risq';
import { RisqOrderCancelButton } from './order_cancel_button';
import type { ResolvedRow } from './order_row_resolve';
import { resolveOrderRow } from './order_row_resolve';

export interface RisqOrderRowConfig {
  game: DwgRisq;
  w: number;
  /** Whether to draw the subject icon/count badge; true in the right panel, false in the left panel */
  show_subject: boolean;
  order: RisqFrontendOrder;
  /** Other same-producible BuildingCreate orders collapsed behind this one, not yet started */
  collapsed_orders?: RisqFrontendOrder[];
  /** Whether a cancel for this (already-submitted) order is pending, drawn as a strikethrough */
  cancelling?: boolean;
  explicit_cancel?: boolean;
  onCancel: (order: RisqFrontendOrder) => void;
  onCancelAll?: (orders: RisqFrontendOrder[]) => void;
  onSelect?: (order: RisqFrontendOrder) => void;
}

export const ROW_H = 30;
const COLLAPSED_STRIP_H = 14;
const ICON_S = 20;
const PADDING = 5;
const CANCEL_S = 14;
const CHIP_W = 30;

export class RisqOrderRow implements CanvasComponent {
  private config: RisqOrderRowConfig;
  private hovering = false;
  private clicking = false;
  private cancel_all_hover = false;
  private cancel_all_clicking = false;
  private resolved: ResolvedRow;
  private cancel_button: RisqOrderCancelButton;
  private progress_tooltip = createTooltipState();

  constructor(config: RisqOrderRowConfig) {
    this.config = config;
    this.resolved = resolveOrderRow(config);
    this.cancel_button = new RisqOrderCancelButton(
      CANCEL_S,
      () => {
        const collapsed = this.config.collapsed_orders;
        this.config.onCancel(collapsed?.length ? collapsed[collapsed.length - 1] : this.config.order);
      },
      config.explicit_cancel
    );
    this.positionButtons();
    if (config.cancelling && !config.explicit_cancel) {
      this.disableCancel();
    }
  }

  getOrder(): RisqFrontendOrder {
    return this.config.order;
  }

  /** All orders this row represents, including any collapsed behind it */
  getOrders(): RisqFrontendOrder[] {
    return [this.config.order, ...(this.config.collapsed_orders ?? [])];
  }

  disableCancel(): void {
    this.cancel_button.disable();
  }

  enableCancel(): void {
    if (!this.config.cancelling || this.config.explicit_cancel) {
      this.cancel_button.enable();
    }
  }

  private positionButtons(): void {
    this.cancel_button.setPosition({ x: this.xi() + this.w() - PADDING - CANCEL_S, y: this.yi() + PADDING });
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
    if (!hovering) {
      this.progress_tooltip.hover_ms = 0;
      this.cancel_all_hover = false;
      this.cancel_button.setHovering(false);
    }
  }
  isClicking(): boolean {
    return this.clicking;
  }
  setClicking(clicking: boolean): void {
    this.clicking = clicking;
    if (!clicking) {
      this.cancel_all_clicking = false;
      this.cancel_button.setClicking(false);
    }
  }

  setW(w: number): void {
    this.config.w = w;
    this.positionButtons();
  }

  private isCollapsed(): boolean {
    return !!this.config.collapsed_orders?.length;
  }

  private isSelected(): boolean {
    return this.config.show_subject && this.config.game.selection.isOrderSubjectSelected(this.config.order);
  }

  private drawSubject(ctx: CanvasRenderingContext2D, x: number, yc: number): void {
    const game = this.config.game;
    const units = this.resolved.subject_units;
    if (units) {
      const total = this.config.order.subjects.length;
      const s = UNIT_CLUSTER_ICON_SIZE;
      const cluster = game.getImageCache().getImage(
        unitClusterIconKey(units, total),
        s,
        units.map((t) => game.getIcon(unitImage(t.unit_id))),
        (cctx) => {
          cctx.translate(s / 2, s / 2);
          drawUnitTypeCluster(
            cctx,
            game,
            units,
            { x: s / 2, y: s / 2 },
            total,
            'rgb(59, 36, 19)',
            'rgba(59, 36, 19, 0.4)'
          );
        }
      );
      if (cluster) {
        ctx.drawImage(cluster, x, yc - ICON_S / 2, ICON_S, ICON_S);
      }
      return;
    }
    ctx.drawImage(game.getIcon(this.resolved.subject_icon!), x, yc - ICON_S / 2, ICON_S, ICON_S);
    const count = this.config.order.subjects.length;
    if (count > 1) {
      this.drawCountBadge(ctx, `×${count}`, x + ICON_S, yc + ICON_S / 2);
    }
  }

  private drawCountBadge(ctx: CanvasRenderingContext2D, label: string, right: number, bottom: number): void {
    ctx.font = 'bold 8px sans-serif';
    const bw = ctx.measureText(label).width + 5;
    const bh = 10;
    const bx = right - bw + 3;
    const by = bottom - bh + 4;
    ctx.fillStyle = 'rgba(59, 36, 19, 0.7)';
    ctx.strokeStyle = 'transparent';
    ctx.lineWidth = 0;
    drawRect(ctx, { x: bx, y: by }, bw, bh, 3);
    drawText(ctx, label, {
      p: { x: bx + bw / 2, y: by + bh / 2 + 1 },
      w: bw,
      fill_style: 'rgb(241, 226, 196)',
      align: 'center',
      baseline: 'middle',
      font: 'bold 8px sans-serif',
    });
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    const selected = this.isSelected();
    configDraw(
      ctx,
      transform,
      {
        fill_style: selected ? 'rgb(218, 198, 160)' : 'rgb(241, 226, 196)',
        stroke_style: 'rgb(59, 36, 19)',
        stroke_width: selected ? 1.8 : 0.6,
        hover_fill_style: selected ? 'rgb(226, 207, 172)' : 'rgb(247, 236, 212)',
        click_fill_style: selected ? 'rgb(234, 217, 184)' : 'rgb(252, 244, 224)',
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), ROW_H, 3);
        let x = this.xi() + PADDING;
        const yc = this.yi() + ROW_H / 2;
        if (this.config.show_subject && (this.resolved.subject_units || this.resolved.subject_icon)) {
          this.drawSubject(ctx, x, yc);
          x += ICON_S + 4;
          ctx.strokeStyle = 'rgba(59, 36, 19, 0.4)';
          ctx.lineWidth = 1;
          drawLine(ctx, { x, y: yc - ICON_S / 2 }, { x, y: yc + ICON_S / 2 });
          x += 5;
        }
        ctx.drawImage(this.config.game.getIcon(this.resolved.icon), x, yc - ICON_S / 2, ICON_S, ICON_S);
        x += ICON_S + 5;
        const chips_w = this.chipsWidth();
        const text_w = this.w() - (x - this.xi()) - chips_w - PADDING - CANCEL_S;
        drawText(ctx, this.resolved.name, {
          p: { x, y: yc - 8 },
          w: text_w,
          fill_style: 'rgb(59, 36, 19)',
          align: 'left',
          font: 'bold 10.5px serif',
        });
        drawText(ctx, this.resolved.target, {
          p: { x, y: yc + 3 },
          w: text_w,
          fill_style: 'rgb(122, 92, 62)',
          align: 'left',
          font: '9.5px sans-serif',
        });
        this.drawChips(ctx, this.xi() + this.w() - PADDING - CANCEL_S - chips_w, yc);
        if (this.resolved.progress !== undefined) {
          this.drawProgress(ctx, this.resolved.progress);
        }
        if (!this.cancel_button.isDisabled()) {
          this.cancel_button.draw(ctx, transform, dt);
          if (this.isCollapsed()) {
            this.drawCancelAllStrip(ctx);
          }
        }
        if (this.config.cancelling) {
          ctx.strokeStyle = 'rgb(122, 46, 27)';
          ctx.lineWidth = 3;
          drawLine(ctx, { x: this.xi(), y: yc }, { x: this.xi() + this.w(), y: yc });
        }
      }
    );
    const hovering =
      !!this.resolved.progress_text && this.hovering && !this.cancel_button.isHovering() && !this.cancel_all_hover;
    const show_tooltip = shouldShowTooltip(this.progress_tooltip, hovering, this.clicking, dt);
    if (show_tooltip && this.resolved.progress_text) {
      drawTooltip(
        this.progress_tooltip,
        ctx,
        transform,
        this.config.game.viewport.canvasSize(),
        this.resolved.progress_text
      );
    }
  }

  private chipsWidth(): number {
    return this.resolved.cost.length * (CHIP_W + 3);
  }

  private drawChips(ctx: CanvasRenderingContext2D, x: number, yc: number): void {
    for (const chip of this.resolved.cost) {
      ctx.drawImage(this.config.game.getIcon(chip.icon), x, yc - 7, 14, 14);
      drawText(ctx, chip.amount.toString(), {
        p: { x: x + 16, y: yc },
        w: CHIP_W - 16,
        fill_style: 'rgb(59, 36, 19)',
        align: 'left',
        baseline: 'middle',
        font: '10px sans-serif',
      });
      x += CHIP_W + 3;
    }
  }

  private drawProgress(ctx: CanvasRenderingContext2D, progress: number): void {
    const y = this.yi() + ROW_H - 2;
    const clamped = Math.max(0, Math.min(1, progress));
    ctx.fillStyle = 'rgba(59, 36, 19, 0.12)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, { x: this.xi(), y }, this.w(), 2);
    ctx.fillStyle = 'rgb(46, 125, 58)';
    drawRect(ctx, { x: this.xi(), y }, this.w() * clamped, 2);
  }

  private drawCancelAllStrip(ctx: CanvasRenderingContext2D): void {
    const y = this.yi() + ROW_H;
    ctx.fillStyle = this.cancel_all_clicking
      ? 'rgba(80, 40, 25, 0.85)'
      : this.cancel_all_hover
        ? 'rgba(80, 40, 25, 0.7)'
        : 'rgba(80, 40, 25, 0.5)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, { x: this.xi(), y }, this.w(), COLLAPSED_STRIP_H, 0);
    const count = 1 + (this.config.collapsed_orders?.length ?? 0);
    drawText(ctx, `Cancel all (×${count})`, {
      p: { x: this.xf() - PADDING, y: y + COLLAPSED_STRIP_H / 2 },
      w: this.w() - 2 * PADDING,
      fill_style: 'rgb(255, 255, 255)',
      align: 'right',
      baseline: 'middle',
      font: '9px sans-serif',
    });
  }

  scroll(_dy: number, _mode: number): boolean {
    return false;
  }

  mousemove(_canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.hovering = !(screen.x < this.xi() || screen.y < this.yi() || screen.x > this.xf() || screen.y > this.yf());
    this.cancel_button.mousemove(screen, screen, transform);
    if (this.cancel_button.isDisabled()) {
      this.cancel_all_hover = false;
      return this.hovering;
    }
    if (this.isCollapsed()) {
      const sy = this.yi() + ROW_H;
      this.cancel_all_hover = this.hovering && screen.y >= sy && screen.y <= sy + COLLAPSED_STRIP_H;
    } else {
      this.cancel_all_hover = false;
    }
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.cancel_button.mousedown(e)) {
      return true;
    }
    if (!this.cancel_button.isDisabled() && this.cancel_all_hover) {
      this.cancel_all_clicking = true;
      return true;
    }
    if (this.hovering) {
      this.clicking = true;
      return true;
    }
    return false;
  }

  mouseup(e: MouseEvent): void {
    this.cancel_button.mouseup(e);
    if (
      !this.cancel_button.isDisabled() &&
      this.cancel_all_clicking &&
      this.cancel_all_hover &&
      this.config.collapsed_orders
    ) {
      this.config.onCancelAll?.([this.config.order, ...this.config.collapsed_orders]);
    } else if (this.clicking && this.hovering) {
      this.config.onSelect?.(this.config.order);
    }
    this.cancel_all_clicking = false;
    this.clicking = false;
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.w();
  }
  yf(): number {
    return this.h();
  }
  w(): number {
    return this.config.w;
  }
  h(): number {
    return this.isCollapsed() && !this.cancel_button.isDisabled() ? ROW_H + COLLAPSED_STRIP_H : ROW_H;
  }
}
