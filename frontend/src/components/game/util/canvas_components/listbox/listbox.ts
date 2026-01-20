import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';

export abstract class DwgListbox implements CanvasComponent {
  private hovering = false;
  private clicking = false;

  isHovering() {
    return this.hovering;
  }

  isClicking() {
    return this.clicking;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {}

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    return false;
  }

  mousedown(e: MouseEvent): boolean {
    return false;
  }

  mouseup(e: MouseEvent): void {}

  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
