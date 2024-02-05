import {BoardTransformData} from '../../../util/canvas_board/canvas_board';
import {CanvasComponent, configDraw} from '../../../util/canvas_components/canvas_component';
import {drawHexagon, drawLine, drawRect, drawText} from '../../../util/canvas_util';
import {Point2D} from '../../../util/objects2d';
import {DwgRisq} from '../risq';
import {buildingImage} from '../risq_buildings';
import {RisqAttackType, RisqBuilding, RisqCombatStats, RisqResource, RisqSpace, RisqZone, ZONE_VISIBILITY, risqTerrainName} from '../risq_data';
import {resourceImage, resourceTypeImage} from '../risq_resources';
import {getSpaceFill} from '../risq_space';
import {INNER_ZONE_MULTIPLIER, setZoneFill} from '../risq_zone';
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
  SPACE,
  ZONE,
}

export class RisqLeftPanel implements CanvasComponent {
  private close_button: RisqLeftPanelButton;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D;
  private showing = false;
  private hovering = false;
  private data_type: LeftPanelDataType;
  private visibility: number;
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
    this.data_type = undefined;
    this.visibility = undefined;
    this.data = undefined;
  }

  openPanel(data_type: LeftPanelDataType, visibility: number, data: any) {
    if (visibility < 1) {
      return; // not explored
    }
    if (visibility < ZONE_VISIBILITY && [
      LeftPanelDataType.RESOURCE,
      LeftPanelDataType.BUILDING,
      LeftPanelDataType.ZONE,
    ].includes(data_type)) {
      return; // zones not visible
    }
    this.data_type = data_type;
    this.visibility = visibility;
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
        case LeftPanelDataType.SPACE:
          this.drawSpace(ctx, this.data);
          break;
        case LeftPanelDataType.ZONE:
          this.drawZone(ctx, this.data);
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
    const resources_left = this.visibility < 4 ? '??' : resource.resources_left.toString();
    drawText(ctx, resources_left, {
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
      drawText(ctx, 'An empty lot that can be built on', {
        p: {x: this.xi() + 0.1 * this.w(), y: yi},
        w: 0.9 * this.w(),
        fill_style: 'black',
        align: 'left',
        font: `14px serif`,
      });
      return;
    }
    yi = this.yi() + 0.25 * this.size.y + 6;
    this.drawCombatStats(ctx, yi, yi + 0.25 * this.size.y - 12, building.combat_stats);
    if (this.risq.getPlayer().player.player_id === building.player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      // TODO: draw action buttons
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
      // TODO: draw orders
    }
  }

  private drawSpace(ctx: CanvasRenderingContext2D, space: RisqSpace) {
    let yi = this.yi() + this.drawName(ctx, risqTerrainName(space.terrain));
    drawText(ctx, 'space', {
      p: {x: this.xc(), y: yi},
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    const hexagon_height = Math.min(this.w(), this.yi() + 0.4 * this.h() - yi - separator_distance);
    ctx.strokeStyle = 'rgba(250, 250, 250, 1)';
    ctx.lineWidth = 2;
    ctx.fillStyle = getSpaceFill(space, false).getString();
    const r = 0.5 * hexagon_height;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const c = {x: this.xc(), y: yi + r};
    drawHexagon(ctx, c, r);
    if (this.visibility >= ZONE_VISIBILITY) {
      ctx.strokeStyle = 'rgba(250, 250, 250, 0.7)';
      ctx.lineWidth = 0.5;
      let zone = space.zones[1][1];
      setZoneFill(ctx, zone, false);
      drawHexagon(ctx, c, inner_r);
      const a = Math.PI / 3;
      for (var i = 0; i < 6; i++) {
        let direction_vector: Point2D = {x: 0, y: 0};
        switch(i) {
          case 0:
            direction_vector = {x: 2, y: 1};
            break;
          case 1:
            direction_vector = {x: 2, y: 0};
            break;
          case 2:
            direction_vector = {x: 1, y: 0};
            break;
          case 3:
            direction_vector = {x: 0, y: 0};
            break;
          case 4:
            direction_vector = {x: 0, y: 1};
            break;
          case 5:
            direction_vector = {x: 1, y: 2};
            break;
        }
        zone = space.zones[direction_vector.x][direction_vector.y];
        setZoneFill(ctx, zone, false);
        ctx.beginPath();
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 6), c.y + inner_r * Math.sin(a * i + Math.PI / 6));
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 2), c.y + inner_r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 2), c.y + r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 6), c.y + r * Math.sin(a * i + Math.PI / 6));
        ctx.closePath();
        ctx.stroke();
        ctx.fill();
      }
    }
    yi += hexagon_height + separator_distance;
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    if (this.visibility >= 2) {
      const rows = 4;
      const image_size = Math.min(
        36,
        (1 / rows) * (0.6 * this.h() - separator_distance - (rows - 1) * separator_distance),
      );
      ctx.fillStyle = 'black';
      const draw_row = (img: HTMLImageElement, text: string) => {
        ctx.drawImage(img, this.xi() + 0.1 * this.w(), yi, image_size, image_size);
        drawText(ctx, `: ${text}`, {
          p: {x: this.xi() + 0.1 * this.w() + image_size + 2, y: yi},
          w: 0.9 * this.w() - image_size - 2,
          fill_style: 'black',
          align: 'left',
          font: `bold ${image_size}px serif`,
        });
        yi += image_size + separator_distance;
      };
      draw_row(this.risq.getIcon('icons/building64'), space.buildings?.size.toString());
      draw_row(this.risq.getIcon('icons/villager64'), space.num_villager_units?.toString());
      draw_row(this.risq.getIcon('icons/unit64'), space.num_military_units?.toString());
      const resources = [...space.total_resources.entries()].filter(r => r[1] > 0).map(r => r[0]).sort((a, b) => a - b);
      const num_resources = resources.length + 0.3 * (resources.length - 1); // account for slashes
      const resource_image_size = Math.min(image_size, (1.0 / num_resources) * 0.8 * this.w());
      for (const [i, r] of resources.entries()) {
        if (i > 0) {
          drawText(ctx, '/', {
            p: {x: this.xi() + 0.1 * this.w() + (i * 1.3 - 0.15) * resource_image_size, y: yi},
            w: 0.3 * resource_image_size,
            fill_style: 'black',
            align: 'center',
            font: `bold ${resource_image_size}px serif`,
          });
        }
        ctx.drawImage(
          this.risq.getIcon(resourceTypeImage(r)),
          this.xi() + 0.1 * this.w() + i * 1.3 * resource_image_size,
          yi, resource_image_size, resource_image_size
        );
      }
      yi += image_size + separator_distance;
    }
  }

  private drawZone(ctx: CanvasRenderingContext2D, data: {space: RisqSpace, zone: RisqZone}) {
    let yi = this.yi() + this.drawName(ctx, risqTerrainName(data.space.terrain));
    drawText(ctx, 'zone', {
      p: {x: this.xc(), y: yi},
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    const hexagon_height = Math.min(this.w(), this.yi() + 0.4 * this.h() - yi - separator_distance);
    ctx.strokeStyle = 'rgba(250, 250, 250, 1)';
    ctx.lineWidth = 2;
    ctx.fillStyle = getSpaceFill(data.space, false).getString();
    const r = 0.5 * hexagon_height;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const c = {x: this.xc(), y: yi + r};
    drawHexagon(ctx, c, r);
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.7)';
    ctx.lineWidth = 0.5;
    let zone = data.space.zones[1][1];
    setZoneFill(ctx, zone, false);
    drawHexagon(ctx, c, inner_r);
    if (this.visibility >= ZONE_VISIBILITY) {
      const a = Math.PI / 3;
      for (var i = 0; i < 6; i++) {
        let direction_vector: Point2D = {x: 0, y: 0};
        switch(i) {
          case 0:
            direction_vector = {x: 2, y: 1};
            break;
          case 1:
            direction_vector = {x: 2, y: 0};
            break;
          case 2:
            direction_vector = {x: 1, y: 0};
            break;
          case 3:
            direction_vector = {x: 0, y: 0};
            break;
          case 4:
            direction_vector = {x: 0, y: 1};
            break;
          case 5:
            direction_vector = {x: 1, y: 2};
            break;
        }
        zone = data.space.zones[direction_vector.x][direction_vector.y];
        setZoneFill(ctx, zone, false);
        ctx.beginPath();
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 6), c.y + inner_r * Math.sin(a * i + Math.PI / 6));
        ctx.lineTo(c.x + inner_r * Math.cos(a * i + Math.PI / 2), c.y + inner_r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 2), c.y + r * Math.sin(a * i + Math.PI / 2));
        ctx.lineTo(c.x + r * Math.cos(a * i + Math.PI / 6), c.y + r * Math.sin(a * i + Math.PI / 6));
        ctx.closePath();
        ctx.stroke();
        ctx.fill();
      }
    }
    yi += hexagon_height + separator_distance;
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    // TODO: draw resources/buildings/units based on visibility (max height = 0.6 * this.h() - separator_distance)
  }

  private drawName(ctx: CanvasRenderingContext2D, name: string): number {
    const text_size = Math.min(40, (1 / 12) * this.size.y);
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
    const max_img_height = (0.25 * this.size.y) - yi + this.yi();
    const img_height = Math.min(max_img_height - 6, 0.8 * this.w());
    ctx.drawImage(this.risq.getIcon(img_name), 0.5 * (this.w() - img_height), yi, img_height, img_height);
    return max_img_height;
  }

  private drawSeparator(ctx: CanvasRenderingContext2D, yi: number) {
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
    ctx.lineWidth = 2;
    drawLine(ctx, {x: this.xi() + 0.1 * this.w(), y: yi}, {x: this.xf() - 0.1 * this.w(), y: yi});
  }

  private drawCombatStats(ctx: CanvasRenderingContext2D, yi: number, yf: number, cs: RisqCombatStats) {
    const gap_size = 4;
    const num_rows = 4;
    if (!(yf - yi > (num_rows + 1) * gap_size)) {
      return;
    }
    const row_size = (1 / num_rows) * (yf - yi);
    yi += gap_size;
    const xi = this.xi() + 0.1 * this.w();
    const health_height = Math.min(14, 0.4 * row_size);
    ctx.strokeStyle = 'black';
    ctx.lineWidth = 0.4;
    ctx.fillStyle = 'black';
    drawRect(ctx, {x: xi, y: yi}, 0.8 * this.w(), health_height);
    if (cs.max_health > 0 && cs.health > 0) {
      ctx.fillStyle = 'rgb(100, 250, 100)';
      drawRect(ctx, {x: xi, y: yi}, (cs.health / cs.max_health) * 0.8 * this.w(), health_height);
    }
    drawText(ctx, `${cs.health} / ${cs.max_health}`, {
      p: {x: xi, y: yi + health_height + 0.1 * row_size},
      w: 0.8 * this.w(),
      fill_style: 'black',
      align: 'left',
      font: `${health_height}px serif`,
    });
    yi += row_size + gap_size;
    if (cs.attack_type !== RisqAttackType.NONE) {
      // TODO: draw attack
      // TODO: draw piercing
    }
    // TODO: draw defense
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
