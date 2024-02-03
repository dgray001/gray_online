import {BoardTransformData} from '../../../util/canvas_board/canvas_board';
import {CanvasComponent, configDraw} from '../../../util/canvas_components/canvas_component';
import {drawLine, drawRect, drawText} from '../../../util/canvas_util';
import {Point2D} from '../../../util/objects2d';
import {DwgRisq} from '../risq';
import {buildingImage} from '../risq_buildings';
import {RisqBuilding, RisqResource} from '../risq_data';
import {resourceImage, resourceTypeImage} from '../risq_resources';
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
    let h = 4 * this.config.w;
    if (h > this.risq.canvasSize().height) {
      h = this.risq.canvasSize().height;
    }
    this.size = {x: h / 3, y: h};
    this.close_button.setPosition({x: this.size.x, y: this.yi() + 0.5 * this.close_button.h()});
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
    ctx.beginPath();
    this.close_button.draw(ctx, transform, dt);
  }

  private drawResource(ctx: CanvasRenderingContext2D, resource: RisqResource) {
    let yi = this.yi() + this.drawName(ctx, resource.display_name);
    yi += this.drawImage(ctx, yi, resourceImage(resource));
    this.drawSeparator(ctx, yi);
    yi += 8;
    ctx.beginPath();
    ctx.drawImage(this.risq.getIcon(resourceTypeImage(resource)), this.xi() + 0.1 * this.w(), yi, 40, 40);
    drawText(ctx, resource.resources_left.toString(), {
      p: {x: this.xi() + 0.1 * this.w() + 48, y: yi + 20},
      w: 0.9 * this.w() - 48,
      fill_style: 'black',
      baseline: 'middle',
      font: '36px serif',
    });
    yi += 50;
    drawText(ctx, `Base gather speed: ${resource.base_gather_speed}`, {
      p: {x: this.xi() + 0.1 * this.w(), y: yi + 12},
      w: 0.9 * this.w(),
      fill_style: 'black',
      baseline: 'middle',
      font: '18px serif',
    });
  }

  private drawBuilding(ctx: CanvasRenderingContext2D, building: RisqBuilding) {
    let yi = this.yi() + this.drawName(ctx, building?.display_name ?? 'Empty Plot');
    yi += this.drawImage(ctx, yi, buildingImage(building));
    this.drawSeparator(ctx, yi);
    if (!building) {
      yi += 12;
      drawText(ctx, 'An empty piece of land that we can build on', {
        p: {x: this.xi() + 0.1 * this.w(), y: yi},
        w: 0.9 * this.w(),
        fill_style: 'black',
        align: 'left',
        font: `14px serif`,
      });
      return;
    }
    // TODO: if visibility
    this.drawSeparator(ctx, this.yi() + this.size.y / 2);
    if (this.risq.getPlayer().player.player_id === building.player_id) {
      // TODO: 
      this.drawSeparator(ctx, this.yi() + 3 * this.size.y / 4);
    }
    // TODO: draw other building stuff
  }

  private drawName(ctx: CanvasRenderingContext2D, name: string): number {
    const text_size = Math.min(40, this.size.y / 4 / 3);
    drawText(ctx, name, {
      p: {x: this.xc(), y: this.yi()},
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: `bold ${0.85 * text_size}px serif`,
    });
    return text_size;
  }

  private drawImage(ctx: CanvasRenderingContext2D, yi: number, img_name: string): number {
    ctx.beginPath();
    const max_img_height = (this.size.y / 4) - yi + this.yi();
    const img_height = Math.min(max_img_height - 6, 0.8 * this.w());
    ctx.drawImage(this.risq.getIcon(img_name), 0.5 * (this.w() - img_height), yi, img_height, img_height);
    return max_img_height;
  }

  private drawSeparator(ctx: CanvasRenderingContext2D, yi: number) {
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
    ctx.lineWidth = 2;
    drawLine(ctx, {x: this.xi() + 0.1 * this.w(), y: yi}, {x: this.xf() - 0.1 * this.w(), y: yi});
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
