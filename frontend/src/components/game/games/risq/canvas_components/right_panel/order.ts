import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw, type CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';

/** Data describing a risq order config */
export declare interface RisqOrderConfig {
  w: number;
}

export class RisqOrder implements CanvasComponent {
  private config: RisqOrderConfig;
  private hovering = false;
  private clicking = false;

  constructor(config: RisqOrderConfig) {
    this.config = config;
  }

  isHovering() {
    return this.hovering;
  }

  setHovering(hovering: boolean) {
    this.hovering = hovering;
  }

  isClicking() {
    return this.clicking;
  }

  setClicking(clicking: boolean) {
    this.clicking = clicking;
  }

  setW(w: number) {
    this.config.w = w;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number) {
    configDraw(
      ctx,
      transform,
      {
        fill_style: 'rgb(0, 200, 0)',
        stroke_width: 0,
        hover_fill_style: 'rgb(80, 255, 80)',
        click_fill_style: 'rgb(160, 255, 160)',
      },
      this.isHovering(),
      this.isClicking(),
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h(), 8);
      }
    );
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    if (m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf()) {
      this.hovering = false;
    } else {
      this.hovering = true;
    }
    return this.isHovering();
  }

  mousedown(_e: MouseEvent): boolean {
    if (this.isHovering()) {
      this.clicking = true;
    }
    return this.isClicking();
  }

  mouseup(_e: MouseEvent): void {
    this.clicking = false;
  }

  xi(): number {
    return 0;
  }

  xf(): number {
    return this.xi() + this.w();
  }

  yi(): number {
    return 0;
  }

  yf(): number {
    return this.yi() + this.h();
  }

  w(): number {
    return this.config.w;
  }

  h(): number {
    return 120;
  }
}
