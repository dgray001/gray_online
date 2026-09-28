import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { BottomPanelItem } from './bottom_panel';
import { RisqMercenaryButton } from './mercenary_button';

const ROWS = 2;
const GAP = 8;

/** Column-major grid of hire buttons, two rows tall and as many columns as unlocked mercenaries need */
export class RisqMercenaryGrid implements BottomPanelItem {
  private risq: DwgRisq;
  private s: number;
  private buttons: RisqMercenaryButton[] = [];
  private mercenary_ids_key = '';
  private p: Point2D = { x: 0, y: 0 };

  constructor(risq: DwgRisq, s: number) {
    this.risq = risq;
    this.s = s;
  }

  dataRefreshed(): void {
    const mercenaries = this.risq.getPlayer()?.available_mercenaries ?? [];
    const ids_key = mercenaries.map((m) => m.id).join(',');
    if (ids_key !== this.mercenary_ids_key) {
      this.mercenary_ids_key = ids_key;
      this.buttons = mercenaries.map((m) => new RisqMercenaryButton(this.risq, m, this.s));
    }
    for (const button of this.buttons) {
      button.dataRefreshed();
    }
  }

  setPosition(p: Point2D): void {
    this.p = p;
    for (const [i, button] of this.buttons.entries()) {
      button.setPosition({ x: p.x + Math.floor(i / ROWS) * (this.s + GAP), y: p.y + (i % ROWS) * (this.s + GAP) });
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
    }
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    for (const button of this.buttons) {
      button.drawTooltip(ctx, transform, risq, dt);
    }
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    return this.buttons.map((button) => button.mousemove(canvas, screen, transform)).some(Boolean);
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
  setHovering(_hovering: boolean): void {}
  isClicking(): boolean {
    return this.buttons.some((button) => button.isPressed());
  }
  setClicking(_clicking: boolean): void {}

  private cols(): number {
    return Math.ceil(this.buttons.length / ROWS);
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
    return this.cols() > 0 ? this.cols() * (this.s + GAP) - GAP : 0;
  }
  h(): number {
    return this.buttons.length > 0 ? Math.min(this.buttons.length, ROWS) * (this.s + GAP) - GAP : 0;
  }
}
