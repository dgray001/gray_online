import {BoardTransformData} from '../../../util/canvas_board/canvas_board';
import {CanvasComponent, configDraw} from '../../../util/canvas_components/canvas_component';
import {drawRect, drawText} from '../../../util/canvas_util';
import {Point2D} from '../../../util/objects2d';
import {DwgRisq} from '../risq';
import {RisqBuilding, RisqResource} from '../risq_data';
import {RisqLeftPanelButton} from './left_panel_close';

/** Config for the left panel */
export declare interface LeftPanelConfig {
  w: number;
  background: string;
}

/** All the data types that can be displayed in the left panel */
export enum LeftPanelDataType {
  RESOURCE,
  BUILDING,
}

export class RisqLeftPanel implements CanvasComponent {
  private close_button: RisqLeftPanelButton;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D;
  private showing = false;
  private hovering = false;
  private data_type: LeftPanelDataType;
  private data: any; // data being shown in panel

  constructor(risq: DwgRisq, config: LeftPanelConfig) {
    this.risq = risq;
    if (config.w < 1) {
      config.w = 120;
    }
    this.config = config;
    this.close_button = new RisqLeftPanelButton(risq);
    this.resolveSize();
  }

  resolveSize() {
    let h = 3 * this.config.w;
    if (h > 0.8 * this.risq.canvasSize().height) {
      h = 0.8 * this.risq.canvasSize().height;
    }
    this.size = {x: h / 3, y: h};
    this.close_button.setPosition({x: this.size.x, y: this.yi()});
  }

  isHovering(): boolean {
    return this.hovering;
  }

  isClicking(): boolean {
    return false;
  }

  isShowing(): boolean {
    return this.showing;
  }

  close() {
    this.showing = false;
  }

  openPanel(data_type: LeftPanelDataType, data: any) {
    this.data_type = data_type;
    this.data = data;
    this.showing = true;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    if (!this.isShowing()) {
      return;
    }
    configDraw(ctx, transform, {
      fill_style: this.config.background,
      stroke_style: 'transparent',
      stroke_width: 0,
      fixed_position: true,
    }, false, false, () => {
      drawRect(ctx, {x: this.xi(), y: this.yi()}, this.w(), this.h());
      switch(this.data_type) {
        case LeftPanelDataType.RESOURCE:
          this.drawResource(ctx, this.data);
          break;
        case LeftPanelDataType.BUILDING:
          this.drawBuilding(ctx, this.data);
          break;
        default:
          console.error('Unknown data type for left panel', this.data_type);
          break;
      }
    });
    this.close_button.draw(ctx, transform, dt);
  }

  private drawResource(ctx: CanvasRenderingContext2D, resource: RisqResource) {
    let yi = this.yi() + this.drawName(ctx, resource.display_name);
    // TODO: draw image
    // TODO: draw resource type and what's left and base gather speed
  }

  private drawBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding) {
    let yi = this.yi() + this.drawName(ctx, building?.display_name ?? 'Empty Plot');
    // TODO: draw image
    // TODO: draw other building stuff
  }

  private drawName(ctx: CanvasRenderingContext2D, name: string): number {
    const text_size = Math.min(40, this.size.y / 3 / 3);
    drawText(ctx, name, {
      p: {x: this.xc(), y: this.yi()},
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: `bold ${0.85 * text_size}px serif`,
    });
    return text_size;
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    if (this.close_button.mousemove(m, transform)) {
      return true;
    }
    m = {
      x: m.x * transform.scale - transform.view.x,
      y: m.y * transform.scale - transform.view.y,
    };
    if (m.x > this.xi() && m.y > this.yi() && m.x < this.xf() && m.y < this.yf()) {
      this.hovering = true;
    } else {
      this.hovering = false;
    }
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    if (this.close_button.mousedown(e)) {
      return true;
    }
    return false;
  }

  mouseup(e: MouseEvent) {
    this.close_button.mouseup(e);
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
