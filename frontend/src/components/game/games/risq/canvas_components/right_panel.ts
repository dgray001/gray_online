import {BoardTransformData} from '../../../util/canvas_board/canvas_board';
import {CanvasComponent, Rotation, configDraw} from '../../../util/canvas_components/canvas_component';
import {drawRect, drawText} from '../../../util/canvas_util';
import {Point2D} from '../../../util/objects2d';
import {DwgRisq} from '../risq';
import {RisqRightPanelButton} from './right_panel_button';

/** Config for the right panel */
export declare interface RightPanelConfig {
  w: number;
  is_open: boolean;
  background: string;
}

export class RisqRightPanel implements CanvasComponent {
  private open_button: RisqRightPanelButton;

  private risq: DwgRisq;
  private config: RightPanelConfig;
  private opening = false;

  constructor(risq: DwgRisq, config: RightPanelConfig) {
    this.risq = risq;
    this.config = config;
    this.open_button = new RisqRightPanelButton(risq);
    this.toggle(config.is_open);
  }

  isHovering(): boolean {
    return false;
  }

  isClicking(): boolean {
    return false;
  }

  isOpen(): boolean {
    return this.config.is_open;
  }

  toggle(open?: boolean) {
    this.config.is_open = open ?? !this.config.is_open;
    const position: Point2D = {
      x: this.risq.canvasSize().width - this.open_button.w(),
      y: 0.5 * this.open_button.h(),
    };
    if (this.config.is_open) {
      position.x -= this.config.w;
    }
    this.opening = true;
    this.open_button.setPosition(position, () => {
      this.opening = false;
      const rotation: Rotation = {
        direction: this.config.is_open,
        angle: this.config.is_open ? 0.5 * Math.PI : -0.5 * Math.PI,
      };
      this.open_button.setRotation(rotation);
    });
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    this.open_button.draw(ctx, transform, dt);
    if (this.opening || this.config.is_open) {
      configDraw(ctx, transform, {
        fill_style: this.config.background,
        stroke_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
      }, false, false, () => {
        drawRect(ctx, {x: this.xi(), y: this.yi()}, this.w(), this.h());
        if (!this.opening && this.config.is_open) {
          drawText(ctx, `Turn ${0}`, {
            p: {x: this.xc(), y: this.yi()},
            w: this.w(),
            fill_style: 'black',
            align: 'center',
            font: '12px',
          });
        }
      });
    }
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    if (this.open_button.mousemove(m, transform)) {
      return true;
    }
    return false;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.open_button.mousedown(e)) {
      return true;
    }
    return false;
  }

  mouseup(_e: MouseEvent) {
    this.open_button.mouseup(_e);
  }

  xi(): number {
    return this.open_button.xf();
  }
  yi(): number {
    return 0;
  }
  xf(): number {
    return this.risq.canvasSize().width;
  }
  yf(): number {
    return this.risq.canvasSize().height;
  }
  xc(): number {
    return this.xi() + 0.5 * this.w();
  }
  yc(): number {
    return this.yi() + 0.5 * this.h();
  }
  w(): number {
    return this.xf() - this.xi();
  }
  h(): number {
    return this.yf() - this.yi();
  }
}
