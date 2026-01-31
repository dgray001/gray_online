import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { DwgButton } from '../../../../util/canvas_components/button/button';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawHexagon, drawLine, drawRect, drawText } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { equalsPoint2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import { buildingImage } from '../../risq_buildings';
import type {
  RectHoverData,
  RisqBuilding,
  RisqCombatStats,
  RisqResource,
  RisqSpace,
  RisqUnit,
  RisqZone,
  UnitByTypeData,
} from '../../risq_data';
import { RisqAttackType, ZONE_VISIBILITY, coordinateToIndex, risqTerrainName } from '../../risq_data';
import { resourceImage, resourceTypeImage } from '../../risq_resources';
import { getSpaceFill } from '../../risq_space';
import { UNIT_HEALTHBAR_COLOR_BACKGROUND, UNIT_HEALTHBAR_COLOR_HEALTH, unitImage } from '../../risq_unit';
import { INNER_ZONE_MULTIPLIER, getZoneFill, resolveHoveredZones } from '../../risq_zone';
import { RisqLeftPanelButton } from './left_panel_close';
import type { LeftPanelConfig, LeftPanelData, PlayerUnitsDrawData, UnitsDrawData } from './left_panel_data';
import { HoverableObjectType, LeftPanelDataType } from './left_panel_data';

export class RisqLeftPanel implements CanvasComponent {
  private close_button: RisqLeftPanelButton;

  private risq: DwgRisq;
  private config: LeftPanelConfig;
  private size: Point2D = { x: 0, y: 0 };
  private showing = false;
  private hovering = false;
  private visibility?: number;
  private data?: LeftPanelData;
  private buttons: DwgButton[] = [];
  private hovered_zone?: RisqZone; // relevant when drawing space and zone
  private hovered_object?: RisqUnit | RisqBuilding | RisqResource;
  private hovered_object_type: HoverableObjectType = HoverableObjectType.NONE;

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
    this.size = { x: h / 3, y: h };
    this.close_button.setPosition({
      x: this.size.x,
      y: this.yi() + 0.5 * this.close_button.h(),
    });
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return false;
  }

  setClicking(_clicking: boolean): void {}

  isShowing(): boolean {
    return this.showing;
  }

  close() {
    this.showing = false;
    this.visibility = undefined;
    this.data = undefined;
    this.hovered_zone = undefined;
  }

  openPanel(open_data: LeftPanelData, visibility: number) {
    if (visibility < 1) {
      return; // not explored
    }
    if (
      visibility < ZONE_VISIBILITY &&
      [
        LeftPanelDataType.RESOURCE,
        LeftPanelDataType.BUILDING,
        LeftPanelDataType.ZONE,
        LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
        LeftPanelDataType.UNITS,
        LeftPanelDataType.UNITS_BY_TYPE,
        LeftPanelDataType.ECONOMIC_UNITS,
        LeftPanelDataType.MILITARY_UNITS,
        LeftPanelDataType.UNIT,
      ].includes(open_data.data_type)
    ) {
      return; // zones not visible
    }
    this.visibility = visibility;
    this.data = open_data;
    this.showing = true;
    switch (open_data.data_type) {
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        this.checkUnitsByTypeData(open_data.data);
        break;
      case LeftPanelDataType.UNITS:
        this.checkUnitsData(open_data.data);
        break;
      default:
        // TODO: implement other validations
        break;
    }
  }

  private checkUnitsData(data: UnitsDrawData) {
    if (!data || !data.space || !data.space.units || !data.units_by_player || data.units_by_player.size < 1) {
      this.close();
      return;
    }
    const new_data: [number, UnitByTypeData[]][] = [];
    for (const [player_id, units_by_type] of data.units_by_player.entries()) {
      if (player_id < 0 || units_by_type.length < 1) {
        continue;
      }
      new_data.push([player_id, units_by_type.filter((u) => u.units.size > 0)]);
    }
    if (new_data.length < 1) {
      this.close();
      return;
    }
    if (new_data.length > 1) {
      this.data = {
        data_type: LeftPanelDataType.MULTIPLE_PLAYERS_UNITS,
        data: { space: data.space, units_by_player: new_data },
      };
      return;
    }
    this.checkUnitsByTypeData({ space: data.space, units: new_data[0][1] });
  }

  /** Check units of a single player and change datatype accordingly */
  private checkUnitsByTypeData(data: PlayerUnitsDrawData) {
    if (data.units.length === 1 && data.units[0].units.size === 1) {
      const unit = data.space.units?.get([...data.units[0].units.values()][0]);
      if (!!unit) {
        this.data = { data_type: LeftPanelDataType.UNIT, data: unit }; // set data as internal id
      } else {
        this.close();
      }
      return;
    }
    const new_data = { space: data.space, units: data.units };
    const data_type = (() => {
      const has_economic_units = data.units.some((u) => u.unit_id < 11);
      const has_military_units = data.units.some((u) => u.unit_id > 10);
      if (has_economic_units && has_military_units) {
        return LeftPanelDataType.UNITS_BY_TYPE;
      } else if (has_economic_units) {
        return LeftPanelDataType.ECONOMIC_UNITS;
      } else if (has_military_units) {
        return LeftPanelDataType.MILITARY_UNITS;
      }
      return undefined;
    })();
    if (!data_type) {
      this.close();
      return;
    }
    this.data = { data_type, data: new_data };
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    if (!this.isShowing()) {
      return;
    }
    configDraw(
      ctx,
      transform,
      {
        fill_style: this.config.background,
        stroke_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
      },
      false,
      false,
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
        switch (this.data?.data_type) {
          case LeftPanelDataType.RESOURCE:
            this.drawResource(ctx, this.data.data);
            break;
          case LeftPanelDataType.BUILDING:
            this.drawBuilding(ctx, this.data.data);
            break;
          case LeftPanelDataType.SPACE:
            this.drawSpace(ctx, this.data.data);
            break;
          case LeftPanelDataType.ZONE:
            this.drawZone(ctx, this.data.data);
            break;
          case LeftPanelDataType.MULTIPLE_PLAYERS_UNITS:
            // TODO: implement
            break;
          case LeftPanelDataType.UNITS_BY_TYPE:
          case LeftPanelDataType.ECONOMIC_UNITS:
          case LeftPanelDataType.MILITARY_UNITS:
            this.drawUnits(ctx, this.data.data);
            break;
          case LeftPanelDataType.UNITS:
            // TODO: implement error (since this should be changed in checkUnitsByTypeData)
            break;
          case LeftPanelDataType.UNIT:
            this.drawUnit(ctx, this.data.data);
            break;
          default:
            console.error('Unknown data type for left panel', this.data);
            break;
        }
        for (const button of this.buttons) {
          button.draw(ctx, transform, dt);
        }
        // TODO: logic in case yi has gone off the rectangle
      }
    );
    ctx.beginPath();
    this.close_button.draw(ctx, transform, dt);
  }

  private drawUnitImage(ctx: CanvasRenderingContext2D, unit: RisqUnit, p: Point2D, s: number) {
    ctx.drawImage(this.risq.getIcon(unitImage(unit)), p.x, p.y, s, s);
    ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    ctx.lineWidth = 0.4;
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    drawRect(ctx, { x: p.x, y: p.y + 0.8 * s }, s, 0.18 * s);
    if (unit.combat_stats.max_health > 0 && unit.combat_stats.health > 0) {
      ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
      drawRect(
        ctx,
        { x: p.x, y: p.y + 0.8 * s },
        (unit.combat_stats.health / unit.combat_stats.max_health) * s,
        0.2 * s
      );
    }
    if (unit.hover_data.hovered) {
      if (unit.hover_data.clicked) {
        ctx.fillStyle = 'rgba(250, 250, 250, 0.4)';
      } else {
        ctx.fillStyle = 'rgba(220, 220, 220, 0.2)';
      }
      ctx.strokeStyle = 'transparent';
      drawRect(ctx, p, s, s);
    }
    unit.hover_data.ps = p;
    unit.hover_data.pe = { x: p.x + s, y: p.y + s };
  }

  private drawUnits(ctx: CanvasRenderingContext2D, data: PlayerUnitsDrawData) {
    const total_units = data.units.map((u) => u.units.size).reduce((a, b) => a + b);
    let yi = this.yi() + this.drawName(ctx, `${total_units} Units`);
    const separator_distance = 8;
    const image_size = total_units < 40 ? 64 : 36;
    const u_img_mult = 1.3;
    const units_per_row = Math.floor((0.8 * this.w()) / (u_img_mult * image_size));
    if (units_per_row < 1) {
      return;
    }
    const rows = data.units.map((u) => Math.ceil(u.units.size / units_per_row)).reduce((a, b) => a + b);
    const max_height = this.yi() + 0.5 * this.size.y - separator_distance - yi;
    const max_rows = Math.floor(max_height / (image_size + separator_distance));
    if (rows > max_rows) {
      // TODO: implement just number of each unit
    } else {
      yi = this.yi() + 0.5 * this.size.y - rows * (image_size + separator_distance);
      for (const unit_data of data.units) {
        if (unit_data.units.size < 1) {
          continue;
        }
        let i = 0;
        for (const unit_id of unit_data.units.values()) {
          const unit = data.space.units?.get(unit_id);
          if (!unit) {
            continue;
          }
          if (i >= units_per_row) {
            i = 0;
            yi += image_size + separator_distance;
          }
          const x = this.xi() + 0.1 * this.w() + i * (image_size + separator_distance);
          this.drawUnitImage(ctx, unit, { x, y: yi }, image_size);
          i++;
        }
        yi += image_size + separator_distance;
      }
    }
    if (this.risq.getPlayer()?.player.player_id === data.units[0].player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      // TODO: draw action buttons => based on this.data_type
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
      // TODO: draw orders => only if they share orders
    }
  }

  private drawUnit(ctx: CanvasRenderingContext2D, unit: RisqUnit) {
    let yi = this.yi() + this.drawName(ctx, unit.display_name);
    yi += this.drawImage(ctx, yi, unitImage(unit));
    this.drawSeparator(ctx, yi);
    yi = this.yi() + 0.25 * this.size.y + 6;
    this.drawCombatStats(ctx, yi, yi + 0.25 * this.size.y - 12, unit.combat_stats);
    if (this.risq.getPlayer()?.player.player_id === unit.player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      // TODO: draw action buttons
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
      // TODO: draw orders
    }
  }

  private drawResource(ctx: CanvasRenderingContext2D, resource: RisqResource) {
    let yi = this.yi() + this.drawName(ctx, resource.display_name);
    yi += this.drawImage(ctx, yi, resourceImage(resource));
    this.drawSeparator(ctx, yi);
    yi += 8;
    ctx.beginPath();
    ctx.drawImage(this.risq.getIcon(resourceTypeImage(resource)), this.xi() + 0.1 * this.w(), yi, 40, 40);
    const resources_left = (this.visibility ?? 0) < 4 ? '??' : resource.resources_left.toString();
    drawText(ctx, resources_left, {
      p: { x: this.xi() + 0.1 * this.w() + 48, y: yi + 20 },
      w: 0.9 * this.w() - 48,
      fill_style: 'black',
      baseline: 'middle',
      font: '36px serif',
    });
    yi += 50;
    drawText(ctx, `Base gather speed: ${resource.base_gather_speed}`, {
      p: { x: this.xi() + 0.1 * this.w(), y: yi + 12 },
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
        p: { x: this.xi() + 0.1 * this.w(), y: yi },
        w: 0.9 * this.w(),
        fill_style: 'black',
        align: 'left',
        font: `14px serif`,
      });
      return;
    }
    yi = this.yi() + 0.25 * this.size.y + 6;
    this.drawCombatStats(ctx, yi, yi + 0.25 * this.size.y - 12, building.combat_stats);
    if (this.risq.getPlayer()?.player.player_id === building.player_id) {
      this.drawSeparator(ctx, this.yi() + 0.5 * this.size.y);
      // TODO: draw action buttons
      this.drawSeparator(ctx, this.yi() + 0.75 * this.size.y);
      // TODO: draw orders
    }
  }

  private drawSpace(ctx: CanvasRenderingContext2D, space: RisqSpace) {
    let yi = this.yi() + this.drawName(ctx, risqTerrainName(space.terrain));
    drawText(ctx, 'space', {
      p: { x: this.xc(), y: yi },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    yi += this.drawSpaceHexagon(ctx, space, separator_distance, yi);
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    if ((this.visibility ?? 0) >= 2) {
      const rows = 4;
      const image_size = Math.min(
        36,
        (1 / rows) * (0.6 * this.h() - separator_distance - (rows - 1) * separator_distance)
      );
      ctx.fillStyle = 'black';
      const draw_row = (img: HTMLImageElement, text: string) => {
        ctx.drawImage(img, this.xi() + 0.1 * this.w(), yi, image_size, image_size);
        drawText(ctx, `: ${text}`, {
          p: { x: this.xi() + 0.1 * this.w() + image_size + 2, y: yi },
          w: 0.9 * this.w() - image_size - 2,
          fill_style: 'black',
          align: 'left',
          font: `bold ${image_size}px serif`,
        });
        yi += image_size + separator_distance;
      };
      draw_row(this.risq.getIcon('icons/building64'), space.buildings?.size.toString() ?? '??');
      draw_row(this.risq.getIcon('icons/villager64'), space.num_villager_units?.toString() ?? '??');
      draw_row(this.risq.getIcon('icons/unit64'), space.num_military_units?.toString() ?? '??');
      const resources = [...(space.total_resources?.entries() ?? [])]
        .filter((r) => r[1] > 0)
        .map((r) => r[0])
        .sort((a, b) => a - b);
      const num_resources = resources.length + 0.3 * (resources.length - 1); // account for slashes
      const resource_image_size = Math.min(image_size, (1.0 / num_resources) * 0.8 * this.w());
      for (const [i, r] of resources.entries()) {
        if (i > 0) {
          drawText(ctx, '/', {
            p: {
              x: this.xi() + 0.1 * this.w() + (i * 1.3 - 0.15) * resource_image_size,
              y: yi,
            },
            w: 0.3 * resource_image_size,
            fill_style: 'black',
            align: 'center',
            font: `bold ${resource_image_size}px serif`,
          });
        }
        ctx.drawImage(
          this.risq.getIcon(resourceTypeImage(r)),
          this.xi() + 0.1 * this.w() + i * 1.3 * resource_image_size,
          yi,
          resource_image_size,
          resource_image_size
        );
      }
      yi += image_size + separator_distance;
    }
  }

  private hexagon_r: number = 0;
  private hexagon_c: Point2D = { x: 0, y: 0 };
  private drawSpaceHexagon(
    ctx: CanvasRenderingContext2D,
    space: RisqSpace,
    separator_distance: number,
    yi: number,
    curr_zone: Point2D = { x: -1, y: -1 }
  ): number {
    const hexagon_height = Math.min(this.w(), this.yi() + 0.4 * this.h() - yi - separator_distance);
    ctx.strokeStyle = 'rgba(250, 250, 250, 1)';
    ctx.lineWidth = 2;
    ctx.fillStyle = getSpaceFill(space, false).getString();
    const r = 0.5 * hexagon_height;
    this.hexagon_r = r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const c = { x: this.xc(), y: yi + r };
    this.hexagon_c = c;
    drawHexagon(ctx, c, r);
    if ((this.visibility ?? 0) >= ZONE_VISIBILITY && space.zones) {
      ctx.strokeStyle = 'rgba(250, 250, 250, 0.7)';
      ctx.lineWidth = 0.5;
      let zone = space.zones[1][1];
      const zone_fill = getZoneFill(zone, true, 4);
      if (curr_zone.x === 1 && curr_zone.y === 1) {
        zone_fill.addColor(255, 255, 255, 0.2);
      } else {
        zone_fill.dAlpha(-0.2);
      }
      ctx.fillStyle = zone_fill.getString();
      drawHexagon(ctx, c, inner_r);
      const a = Math.PI / 3;
      for (let i = 0; i < 6; i++) {
        let direction_vector: Point2D = { x: 0, y: 0 };
        switch (i) {
          case 0:
            direction_vector = { x: 2, y: 1 };
            break;
          case 1:
            direction_vector = { x: 2, y: 0 };
            break;
          case 2:
            direction_vector = { x: 1, y: 0 };
            break;
          case 3:
            direction_vector = { x: 0, y: 0 };
            break;
          case 4:
            direction_vector = { x: 0, y: 1 };
            break;
          case 5:
            direction_vector = { x: 1, y: 2 };
            break;
        }
        zone = space.zones[direction_vector.x][direction_vector.y];
        const zone_fill = getZoneFill(zone, true, 4);
        if (curr_zone.x === direction_vector.x && curr_zone.y === direction_vector.y) {
          zone_fill.addColor(255, 255, 255, 0.2);
        } else {
          zone_fill.dAlpha(-0.2);
        }
        ctx.fillStyle = zone_fill.getString();
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
    return hexagon_height + separator_distance;
  }

  private drawZone(ctx: CanvasRenderingContext2D, data: { space: RisqSpace; zone: RisqZone }) {
    let yi = this.yi() + this.drawName(ctx, risqTerrainName(data.space.terrain));
    drawText(ctx, 'zone', {
      p: { x: this.xc(), y: yi },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: '18px serif',
    });
    yi += 26;
    const separator_distance = 8;
    yi += this.drawSpaceHexagon(ctx, data.space, separator_distance, yi, coordinateToIndex(1, data.zone.coordinate));
    this.drawSeparator(ctx, yi);
    yi += separator_distance;
    const max_image_size = 36; // it should be this size
    const u_img_mult = 1.3;
    const units_per_row = Math.floor((0.8 * this.w() - 1.6 * max_image_size) / (u_img_mult * max_image_size));
    const economic_rows = Math.ceil(data.zone.economic_units.length / units_per_row);
    const military_rows = Math.ceil(data.zone.military_units.length / units_per_row);
    const rows = 1 + economic_rows + military_rows;
    const image_size = Math.min(
      max_image_size,
      (1 / rows) * (0.6 * this.h() - separator_distance - (rows - 1) * separator_distance)
    );
    ctx.fillStyle = 'black';
    const draw_row = (img: HTMLImageElement, text: string, hover_data?: RectHoverData) => {
      const ps = { x: this.xi() + 0.1 * this.w(), y: yi };
      const pe = { x: ps.x + 0.8 * this.w(), y: ps.y + image_size };
      if (hover_data?.hovered) {
        ctx.strokeStyle = 'transparent';
        if (hover_data?.clicked) {
          ctx.fillStyle = 'rgba(250, 250, 250, 0.4)';
        } else {
          ctx.fillStyle = 'rgba(210, 210, 210, 0.25)';
        }
        drawRect(ctx, ps, pe.x - ps.x, pe.y - ps.y);
      }
      ctx.drawImage(img, ps.x, ps.y, image_size, image_size);
      drawText(ctx, `: ${text}`, {
        p: { x: ps.x + image_size + 2, y: yi + 0.5 * image_size },
        w: 0.8 * this.w() - image_size - 2,
        fill_style: 'black',
        align: 'left',
        baseline: 'middle',
        font: `bold ${image_size}px serif`,
      });
      if (!!hover_data) {
        hover_data.ps = ps;
        hover_data.pe = pe;
      }
    };
    if (!!data.zone.resource) {
      draw_row(
        this.risq.getIcon(resourceImage(data.zone.resource)),
        data.zone.resource.display_name,
        data.zone.resource.hover_data
      );
    } else {
      draw_row(
        this.risq.getIcon(buildingImage(data.zone.building)),
        data.zone.building?.display_name ?? 'Empty Plot',
        data.zone.building?.hover_data
      );
    }
    yi += image_size + separator_distance;
    const xi = this.xi() + 0.1 * this.w() + 0.6 * image_size;
    if (data.zone.economic_units.length > 0) {
      draw_row(this.risq.getIcon('icons/villager64'), '');
      let i = 1;
      let j = 0;
      for (const u of data.zone.economic_units) {
        const unit = data.zone.units.get(u);
        if (!unit) {
          continue;
        }
        const p = {
          x: xi + i * u_img_mult * image_size,
          y: yi + j * u_img_mult * image_size,
        };
        this.drawUnitImage(ctx, unit, p, image_size);
        i++;
        if (i > units_per_row) {
          i = 1;
          j++;
        }
      }
      yi += economic_rows * u_img_mult * image_size + separator_distance;
    }
    if (data.zone.military_units.length > 0) {
      draw_row(this.risq.getIcon('icons/unit64'), '');
      let i = 1;
      let j = 0;
      for (const u of data.zone.military_units) {
        const unit = data.zone.units.get(u);
        if (!unit) {
          continue;
        }
        const p = {
          x: xi + i * u_img_mult * image_size,
          y: yi + j * u_img_mult * image_size,
        };
        this.drawUnitImage(ctx, unit, p, image_size);
        i++;
        if (i > units_per_row) {
          i = 1;
          j++;
        }
      }
      yi += military_rows * u_img_mult * image_size + separator_distance;
    }
  }

  private drawName(ctx: CanvasRenderingContext2D, name: string): number {
    const text_size = Math.min(40, (1 / 12) * this.size.y);
    drawText(ctx, name, {
      p: { x: this.xc(), y: this.yi() + 3 },
      w: this.w(),
      fill_style: 'black',
      align: 'center',
      font: `bold ${0.85 * text_size}px serif`,
    });
    return text_size + 3;
  }

  private drawImage(ctx: CanvasRenderingContext2D, yi: number, img_name: string): number {
    ctx.beginPath();
    const max_img_height = 0.25 * this.size.y - yi + this.yi();
    const img_height = Math.min(max_img_height - 6, 0.8 * this.w());
    ctx.drawImage(this.risq.getIcon(img_name), 0.5 * (this.w() - img_height), yi, img_height, img_height);
    return max_img_height;
  }

  private drawSeparator(ctx: CanvasRenderingContext2D, yi: number) {
    ctx.strokeStyle = 'rgba(60, 60, 60, 0.7)';
    ctx.lineWidth = 2;
    drawLine(ctx, { x: this.xi() + 0.1 * this.w(), y: yi }, { x: this.xf() - 0.1 * this.w(), y: yi });
  }

  private drawCombatStats(ctx: CanvasRenderingContext2D, yi: number, yf: number, cs: RisqCombatStats) {
    const gap_size = 4;
    const num_rows = 4; // health, attack, piercing, defense
    if (!(yf - yi > (num_rows + 1) * gap_size)) {
      return;
    }
    const row_size = (1 / num_rows) * (yf - yi);
    yi += gap_size;
    const xi = this.xi() + 0.1 * this.w();
    const health_height = Math.min(14, 0.4 * row_size);
    ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    ctx.lineWidth = 0.4;
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
    drawRect(ctx, { x: xi, y: yi }, 0.8 * this.w(), health_height);
    if (cs.max_health > 0 && cs.health > 0) {
      ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
      drawRect(ctx, { x: xi, y: yi }, (cs.health / cs.max_health) * 0.8 * this.w(), health_height);
    }
    drawText(ctx, `${cs.health} / ${cs.max_health}`, {
      p: { x: xi, y: yi + health_height + 0.1 * row_size },
      w: 0.8 * this.w(),
      fill_style: 'black',
      align: 'left',
      font: `${health_height}px serif`,
    });
    yi += row_size + gap_size;
    const draw_stat_row = (stats: [string, number][]) => {
      let xi = this.xi() + 0.1 * this.w();
      const dx = (0.8 * this.w()) / stats.length;
      const image_size = Math.min(0.7 * row_size, 0.3 * dx);
      let drew_one = false;
      for (const s of stats.filter((s) => !!s[1])) {
        drew_one = true;
        ctx.drawImage(this.risq.getIcon(s[0]), xi, yi, image_size, image_size);
        drawText(ctx, s[1].toString(), {
          p: { x: xi + image_size + 0.1 * dx, y: yi + 0.5 * image_size },
          w: dx - image_size,
          fill_style: 'black',
          align: 'left',
          baseline: 'middle',
          font: `${0.9 * image_size}px serif`,
        });
        xi += dx;
      }
      if (drew_one) {
        yi += row_size + gap_size;
      }
    };
    if (cs.attack_type !== RisqAttackType.NONE) {
      draw_stat_row([
        ['risq/icons/attack_blunt', cs.attack_blunt],
        ['risq/icons/attack_piercing', cs.attack_piercing],
        ['risq/icons/attack_magic', cs.attack_magic],
      ]);
      draw_stat_row([
        ['risq/icons/penetration_blunt', cs.penetration_blunt],
        ['risq/icons/penetration_piercing', cs.penetration_piercing],
        ['risq/icons/penetration_magic', cs.penetration_magic],
      ]);
    }
    draw_stat_row([
      ['risq/icons/defense_blunt', cs.defense_blunt],
      ['risq/icons/defense_piercing', cs.defense_piercing],
      ['risq/icons/defense_magic', cs.defense_magic],
    ]);
  }

  private objectHoverLogic(
    m: Point2D,
    object: RisqUnit | RisqBuilding | RisqResource | undefined,
    object_type: HoverableObjectType
  ): boolean {
    if (!object) {
      return false;
    }
    if (
      m.x < object.hover_data.ps.x ||
      m.y < object.hover_data.ps.y ||
      m.x > object.hover_data.pe.x ||
      m.y > object.hover_data.pe.y
    ) {
      object.hover_data.hovered = false;
      return false;
    } else {
      if (
        !!this.hovered_object &&
        (this.hovered_object.internal_id !== object.internal_id || this.hovered_object_type !== object_type)
      ) {
        this.hovered_object.hover_data.hovered = false;
        this.hovered_object.hover_data.clicked = false;
      }
      object.hover_data.hovered = true;
      this.hovered_object = object;
      this.hovered_object_type = !!object ? object_type : HoverableObjectType.NONE;
      return true;
    }
  }

  scroll(_dy: number, _mode: number): boolean {
    return false;
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
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        const space: RisqSpace =
          this.data?.data_type === LeftPanelDataType.SPACE ? this.data.data : this.data.data.space;
        const new_hovered_zone = resolveHoveredZones(m, space, this.hexagon_r, this.hexagon_c, true);
        if (!!this.hovered_zone && !equalsPoint2D(this.hovered_zone.coordinate, new_hovered_zone?.coordinate)) {
          this.hovered_zone.hovered = false;
          this.hovered_zone.clicked = false;
        }
        this.hovered_zone = new_hovered_zone;
        if (this.data.data_type === LeftPanelDataType.ZONE) {
          const zone = this.data.data.zone;
          const unit_ids: number[] = [...zone.economic_units, ...zone.military_units];
          for (const unit_id of unit_ids) {
            const unit = zone.units.get(unit_id);
            if (!unit) {
              continue;
            }
            this.objectHoverLogic(m, unit, HoverableObjectType.UNIT);
          }
          this.objectHoverLogic(m, zone.resource, HoverableObjectType.RESOURCE);
          this.objectHoverLogic(m, zone.building, HoverableObjectType.BUILDING);
        }
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        for (const unit_data of this.data.data.units) {
          if (unit_data.units.size < 1) {
            continue;
          }
          for (const unit_id of unit_data.units.values()) {
            const unit = this.data.data.space.units?.get(unit_id);
            if (!unit) {
              continue;
            }
            this.objectHoverLogic(m, unit, HoverableObjectType.UNIT);
          }
        }
        break;
      default:
        break;
    }
    for (const button of this.buttons) {
      button.mousemove(m, transform);
    }
    return this.isHovering();
  }

  mousedown(e: MouseEvent): boolean {
    if (this.close_button.mousedown(e)) {
      return true;
    }
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        if (!!this.hovered_zone) {
          this.hovered_zone.clicked = true;
        } else if (!!this.hovered_object) {
          this.hovered_object.hover_data.clicked = true;
        }
        break;
      case LeftPanelDataType.UNITS:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (!!this.hovered_object) {
          this.hovered_object.hover_data.clicked = true;
        }
        break;
      default:
        break;
    }
    for (const button of this.buttons) {
      button.mousedown(e);
    }
    return this.isHovering();
  }

  mouseup(e: MouseEvent) {
    this.close_button.mouseup(e);
    switch (this.data?.data_type) {
      case LeftPanelDataType.SPACE:
      case LeftPanelDataType.ZONE:
        const space: RisqSpace =
          this.data?.data_type === LeftPanelDataType.SPACE ? this.data.data : this.data.data.space;
        if (!!this.hovered_zone && this.hovered_zone.clicked) {
          this.hovered_zone.clicked = false;
          if (this.hovered_zone.hovered) {
            this.openPanel(
              {
                data_type: LeftPanelDataType.ZONE,
                data: {
                  space,
                  zone: this.hovered_zone,
                },
              },
              this.visibility ?? 0
            );
          }
        } else if (!!this.hovered_object && this.hovered_object.hover_data.clicked) {
          this.hovered_object.hover_data.clicked = false;
          if (this.hovered_object.hover_data.hovered) {
            switch (this.hovered_object_type) {
              case HoverableObjectType.UNIT:
                if (e.shiftKey && this.data.data_type === LeftPanelDataType.ZONE) {
                  const units_by_type = this.data.data.zone.units_by_type.get(
                    (this.hovered_object as RisqUnit).player_id
                  );
                  if (!!units_by_type) {
                    this.openPanel(
                      {
                        data_type: LeftPanelDataType.UNITS_BY_TYPE,
                        data: {
                          space,
                          units: [...units_by_type.values()].filter(
                            (u: UnitByTypeData) => u.unit_id === (this.hovered_object as RisqUnit).unit_id
                          ),
                        },
                      },
                      this.visibility ?? 0
                    );
                  }
                } else {
                  // @ts-ignore
                  this.openPanel({ data_type: LeftPanelDataType.UNIT, data: this.hovered_object }, this.visibility);
                }
                break;
              case HoverableObjectType.BUILDING:
                // @ts-ignore
                this.openPanel({ data_type: LeftPanelDataType.BUILDING, data: this.hovered_object }, this.visibility);
                break;
              case HoverableObjectType.RESOURCE:
                // @ts-ignore
                this.openPanel({ data_type: LeftPanelDataType.RESOURCE, data: this.hovered_object }, this.visibility);
                break;
              default:
                break;
            }
          }
        }
        break;
      case LeftPanelDataType.UNITS_BY_TYPE:
      case LeftPanelDataType.ECONOMIC_UNITS:
      case LeftPanelDataType.MILITARY_UNITS:
        if (!!this.hovered_object && this.hovered_object.hover_data.clicked) {
          this.hovered_object.hover_data.clicked = false;
          if (this.hovered_object.hover_data.hovered) {
            if (e.shiftKey) {
              this.openPanel(
                {
                  data_type: LeftPanelDataType.UNITS_BY_TYPE,
                  data: {
                    space: this.data.data.space,
                    units: this.data.data.units.filter(
                      (u: UnitByTypeData) => u.unit_id === (this.hovered_object as RisqUnit).unit_id
                    ),
                  },
                },
                this.visibility ?? 0
              );
            } else {
              // @ts-ignore
              this.openPanel({ data_type: LeftPanelDataType.UNIT, data: this.hovered_object }, this.visibility);
            }
          }
        }
        break;
      default:
        break;
    }
    for (const button of this.buttons) {
      button.mouseup(e);
    }
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
