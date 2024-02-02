import {BoardTransformData} from '../../../util/canvas_board/canvas_board';
import {CanvasComponent} from '../../../util/canvas_components/canvas_component';
import {Point2D} from '../../../util/objects2d';
import {DwgRisq} from '../risq';
import {RisqLeftPanelButton} from './left_panel_close';

/** Config for the left panel */
export declare interface LeftPanelConfig {
  w: number;
}

export class RisqLeftPanel implements CanvasComponent {
  private close_button: RisqLeftPanelButton;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D;
  private showing = false;

  constructor(risq: DwgRisq, config: LeftPanelConfig) {
    this.risq = risq;
    if (config.w < 1) {
      config.w = 100;
    }
    this.config = config;
    this.resolveSize();
  }

  resolveSize() {
    let h = 3 * this.config.w;
    if (h > 0.8 * this.risq.canvasSize().height) {
      h = 0.8 * this.risq.canvasSize().height;
    }
    this.size = {x: h / 3, y: h};
  }

  isHovering(): boolean {
    return false;
  }

  isClicking(): boolean {
    return false;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    return false;
  }

  mousedown(e: MouseEvent): boolean {
    return false;
  }

  mouseup(_e: MouseEvent) {
  }

  xi(): number {
    return 0;
  }
  yi(): number {
    return 0.5 * (this.risq.canvasSize().height - this.size.y);
  }
  xf(): number {
    return this.showing ? this.xi() + this.w() : 0;
  }
  yf(): number {
    return this.showing ? this.yi() + this.h() : 0;
  }
  xc(): number {
    return this.xi() + 0.5 * this.w();
  }
  yc(): number {
    return this.yi() + 0.5 * this.h();
  }
  w(): number {
    return this.size.x;
  }
  h(): number {
    return this.size.y;
  }
}
