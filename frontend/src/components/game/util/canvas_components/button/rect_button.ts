import {BoardTransformData} from '../../canvas_board/canvas_board';
import {Point2D} from '../../objects2d';
import {DrawConfig, configDraw} from '../canvas_component';
import {ButtonConfig, DwgButton} from './button';

/** Config data for a rect button */
export declare interface RectButtonConfig {
  button_config: ButtonConfig;
  p: Point2D;
  w: number;
  h: number;
  draw_config: DrawConfig;
}

export abstract class DwgRectButton extends DwgButton {
  private rect_config: RectButtonConfig;

  constructor(config: RectButtonConfig) {
    super(config.button_config);
    this.rect_config = config;
  }

  override _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
    configDraw(ctx, transform, this.rect_config.draw_config, this.isHovering(), this.isClicking(), () => {
      ctx.rect(this.rect_config.p.x, this.rect_config.p.y, this.rect_config.w, this.rect_config.h);
    });
  }

  override mouseOver(m: Point2D, transform: BoardTransformData): boolean {
    if (this.rect_config.draw_config.fixed_position) {
      m = {
        x: m.x * transform.scale - transform.view.x,
        y: m.y * transform.scale - transform.view.y,
      };
    }
    if (m.x < this.rect_config.p.x || m.y < this.rect_config.p.y) {
      return false;
    }
    if (
      m.x > this.rect_config.p.x + this.rect_config.w ||
      m.y > this.rect_config.p.y + this.rect_config.h
    ) {
      return false;
    }
    return true;
  }
}
