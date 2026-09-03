import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import type { RisqFrontendOrder } from '../../risq_data';
import { RisqOrderCancelButton } from './order_cancel_button';
import type { RisqOrderRowConfig } from './order_row_data';
import type { ResolvedRow } from './order_row_resolve';
import { resolveOrderRow } from './order_row_resolve';

const ROW_H = 30;
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

  constructor(config: RisqOrderRowConfig) {
    this.config = config;
    this.resolved = resolveOrderRow(config);
    this.cancel_button = new RisqOrderCancelButton(CANCEL_S, () => {
      const collapsed = this.config.collapsed_orders;
      this.config.onCancel(collapsed?.length ? collapsed[collapsed.length - 1] : this.config.order);
    });
    this.positionButtons();
  }

  getOrder(): RisqFrontendOrder {
    return this.config.order;
  }

  /** All orders this row represents, including any collapsed behind it */
  getOrders(): RisqFrontendOrder[] {
    return [this.config.order, ...(this.config.collapsed_orders ?? [])];
  }

  private positionButtons(): void {
    this.cancel_button.setPosition({ x: this.xi() + this.w() - PADDING - CANCEL_S, y: this.yi() + PADDING });
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }
  isClicking(): boolean {
    return this.clicking;
  }
  setClicking(clicking: boolean): void {
    this.clicking = clicking;
  }

  setW(w: number): void {
    this.config.w = w;
    this.positionButtons();
  }

  private isCollapsed(): boolean {
    return !!this.config.collapsed_orders?.length;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'rgb(241, 226, 196)',
        stroke_style: 'rgb(59, 36, 19)',
        stroke_width: 0.6,
        hover_fill_style: 'rgb(247, 236, 212)',
        click_fill_style: 'rgb(252, 244, 224)',
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), ROW_H, 3);
        let x = this.xi() + PADDING;
        const yc = this.yi() + ROW_H / 2;
        if (this.config.show_subject && this.resolved.subject_icon) {
          ctx.drawImage(this.config.game.getIcon(this.resolved.subject_icon), x, yc - ICON_S / 2, ICON_S, ICON_S);
          const count = this.config.order.subjects.length;
          if (count > 1) {
            drawText(ctx, `×${count}`, {
              p: { x, y: yc + ICON_S / 2 - 7 },
              w: ICON_S,
              fill_style: 'rgb(59, 36, 19)',
              align: 'right',
              font: 'bold 8px sans-serif',
            });
          }
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
        this.cancel_button.draw(ctx, transform, dt);
        if (this.isCollapsed()) {
          this.drawCancelAllStrip(ctx);
        }
      }
    );
  }

  private chipsWidth(): number {
    return this.resolved.cost.length * (CHIP_W + 3);
  }

  private drawChips(ctx: CanvasRenderingContext2D, x: number, yc: number) {
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

  private drawProgress(ctx: CanvasRenderingContext2D, progress: number) {
    const y = this.yi() + ROW_H - 2;
    const clamped = Math.max(0, Math.min(1, progress));
    ctx.fillStyle = 'rgba(59, 36, 19, 0.12)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, { x: this.xi(), y }, this.w(), 2);
    ctx.fillStyle = 'rgb(46, 125, 58)';
    drawRect(ctx, { x: this.xi(), y }, this.w() * clamped, 2);
  }

  private drawCancelAllStrip(ctx: CanvasRenderingContext2D) {
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

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    this.hovering = !(m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf());
    this.cancel_button.mousemove(m, transform);
    if (this.isCollapsed()) {
      const sy = this.yi() + ROW_H;
      this.cancel_all_hover = this.hovering && m.y >= sy && m.y <= sy + COLLAPSED_STRIP_H;
    } else {
      this.cancel_all_hover = false;
    }
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.cancel_button.mousedown(e)) {
      return true;
    }
    if (this.cancel_all_hover) {
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
    if (this.cancel_all_clicking && this.cancel_all_hover && this.config.collapsed_orders) {
      this.config.onCancelAll?.([this.config.order, ...this.config.collapsed_orders]);
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
    return this.isCollapsed() ? ROW_H + COLLAPSED_STRIP_H : ROW_H;
  }
}
