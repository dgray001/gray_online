import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { BottomPanelItem } from './bottom_panel';
import { RisqMercenaryButton } from './mercenary_button';
import { clipBottomPanelItem } from './clip_item';

const ROWS = 1;
const GAP = 8;
const COST_HEIGHT = 20;

/** Single row of hire buttons with gold prices underneath */
export class RisqMercenaryGrid implements BottomPanelItem {
  private risq: DwgRisq;
  private s: number;
  private buttons: RisqMercenaryButton[] = [];
  private mercenary_ids_key = '';
  private p: Point2D = { x: 0, y: 0 };
  private available_width = Infinity;
  private scroll_x = 0;

  constructor(risq: DwgRisq, s: number) {
    this.risq = risq;
    this.s = s;
  }
  setAvailableWidth(width: number): void {
    if (this.available_width === Math.max(0, width)) {
      return;
    }
    this.available_width = Math.max(0, width);
    this.scroll_x = Math.min(this.scroll_x, Math.max(0, this.contentWidth() - this.w()));
    this.setPosition(this.p);
  }

  dataRefreshed(): void {
    const mercenaries = this.risq.getPlayer()?.available_mercenaries ?? [];
    const ids_key = mercenaries.map((m) => m.id).join(',');
    if (ids_key !== this.mercenary_ids_key) {
      this.mercenary_ids_key = ids_key;
      this.buttons = mercenaries.map((m) => new RisqMercenaryButton(this.risq, m, this.s));
      this.scroll_x = Math.min(this.scroll_x, Math.max(0, this.contentWidth() - this.w()));
      this.setPosition(this.p);
    }
    for (const button of this.buttons) {
      button.dataRefreshed();
    }
  }

  setPosition(p: Point2D): void {
    this.p = p;
    for (const [i, button] of this.buttons.entries()) {
      button.setPosition({
        x: p.x + Math.floor(i / ROWS) * (this.s + GAP) - this.scroll_x,
        y: p.y + (i % ROWS) * (this.s + GAP),
      });
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    ctx.save();
    clipBottomPanelItem(ctx, transform, this);
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
      button.drawCost(ctx, transform);
    }
    ctx.restore();
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    for (const button of this.buttons) {
      button.drawTooltip(ctx, transform, risq, dt);
    }
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    if (screen.x < this.xi() || screen.x > this.xf() || screen.y < this.yi() || screen.y > this.yf()) {
      this.setHovering(false);
      return false;
    }
    return this.buttons.map((button) => button.mousemove(canvas, screen, transform)).some(Boolean);
  }
  scroll(dy: number, mode: number): boolean {
    const next = Math.max(0, Math.min(this.contentWidth() - this.w(), this.scroll_x + dy * (mode ? 16 : 1)));
    if (next === this.scroll_x) {
      return false;
    }
    this.scroll_x = next;
    this.setPosition(this.p);
    this.setHovering(false);
    return true;
  }

  mousedown(e: MouseEvent): boolean {
    return this.buttons.map((button) => button.mousedown(e)).some(Boolean);
  }

  mouseup(e: MouseEvent): void {
    for (const button of this.buttons) {
      button.mouseup(e);
    }
  }

  isHovering(): boolean {
    return this.buttons.some((button) => button.isHovering());
  }
  setHovering(hovering: boolean): void {
    this.buttons.forEach((button) => button.setHovering(hovering));
  }
  isClicking(): boolean {
    return this.buttons.some((button) => button.isPressed());
  }
  setClicking(clicking: boolean): void {
    this.buttons.forEach((button) => button.setClicking(clicking));
  }

  private cols(): number {
    return Math.ceil(this.buttons.length / ROWS);
  }
  private contentWidth(): number {
    return this.cols() > 0 ? this.cols() * (this.s + GAP) - GAP : 0;
  }
  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.xi() + this.w();
  }
  yf(): number {
    return this.yi() + this.h();
  }
  w(): number {
    return Math.min(this.available_width, this.contentWidth());
  }
  h(): number {
    return this.buttons.length > 0 ? this.s + COST_HEIGHT : 0;
  }
}
