import {ColorRGB} from "../../../../scripts/color_rgb";
import {drawHexagon} from "../../util/canvas_util";
import {Point2D} from "../../util/objects2d";
import {DwgRisq} from "./risq";
import {RisqSpace} from "./risq_data";
import {INNER_ZONE_MULTIPLIER, drawRisqZone, setZoneFill} from "./risq_zone";

/** How much detail to draw in a space */
export enum DrawRisqSpaceDetail {
  OWNERSHIP,
  SPACE_DETAILS,
  ZONE_DETAILS,
}

/** Config data for drawing a risq space */
export declare interface DrawRisqSpaceConfig {
  hex_r: number;
  inset_w: number;
  inset_h: number;
  inset_row: number;
  draw_detail: DrawRisqSpaceDetail;
}

const space_line_width = new Map<DrawRisqSpaceDetail, number>([
  [DrawRisqSpaceDetail.OWNERSHIP, 3],
  [DrawRisqSpaceDetail.SPACE_DETAILS, 2],
  [DrawRisqSpaceDetail.ZONE_DETAILS, 1.2],
]);

/** Draws the input risq space */
export function drawRisqSpace(ctx: CanvasRenderingContext2D, game: DwgRisq,
  space: RisqSpace, config: DrawRisqSpaceConfig
) {
  ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
  const fill = getSpaceFill(space);
  ctx.fillStyle = fill.getString();
  ctx.lineWidth = space_line_width.get(config.draw_detail);
  drawHexagon(ctx, space.center, config.hex_r);
  ctx.textAlign = 'left';
  const black_text = fill.getBrightness() > 0.5;
  if (config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP) {
    return; // ownership and terrain indicated by space fill color
  } else if (config.draw_detail === DrawRisqSpaceDetail.SPACE_DETAILS) {
    if (space.visibility < 2) {
      return;
    }
    // TODO: draw resources as fourth row (resource icons of resources that are there)
    let building_img = game.getIcon('icons/building64');
    let villager_img = game.getIcon('icons/villager64');
    let unit_img = game.getIcon('icons/unit64');
    if (black_text) {
      ctx.fillStyle = 'black';
    } else {
      ctx.fillStyle = 'white';
      building_img = game.getIcon('icons/building_white64');
      villager_img = game.getIcon('icons/villager_white64');
      unit_img = game.getIcon('icons/unit_white64');
    }
    ctx.textBaseline = 'top';
    ctx.font = `bold ${config.inset_row}px serif`;
    const xs = space.center.x - 0.5 * config.inset_w;
    const y1 = space.center.y - 0.5 * config.inset_h;
    ctx.drawImage(building_img, xs, y1, config.inset_row, config.inset_row);
    ctx.fillText(`: ${space.buildings?.size.toString()}`, xs + config.inset_row + 2, y1, config.inset_w - config.inset_row - 2);
    const y2 = space.center.y - 0.5 * config.inset_h + config.inset_row + 2;
    ctx.drawImage(villager_img, xs, y2, config.inset_row, config.inset_row);
    ctx.fillText(`: ${space.num_villager_units?.toString()}`, xs + config.inset_row + 2,
      y2, config.inset_w - config.inset_row - 2);
    const y3 = space.center.y - 0.5 * config.inset_h + 2 * (config.inset_row + 2);
    ctx.drawImage(unit_img, xs, y3, config.inset_row, config.inset_row);
    ctx.fillText(`: ${space.num_military_units?.toString()}`, xs + config.inset_row + 2,
      y3, config.inset_w - config.inset_row - 2);
  } else if (config.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
    if (space.visibility < 3) {
      return;
    }
    ctx.translate(space.center.x, space.center.y);
    ctx.fillStyle = 'rgb(10, 120, 10, 0.8)';
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.2)';
    ctx.lineWidth = 0.1;
    let zone = space.zones[1][1];
    setZoneFill(ctx, zone);
    const r = config.hex_r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    let zone_r = 0.45 * r;
    drawHexagon(ctx, {x: 0, y: 0}, inner_r);
    drawRisqZone(ctx, game, zone, black_text, zone_r, 0,
      {x: 0.18 * r * Math.cos(3 * Math.PI / 6), y: 0.18 * r * Math.sin(3 * Math.PI / 6)},
      {x: 0.18 * r * Math.cos(7 * Math.PI / 6), y: 0.18 * r * Math.sin(7 * Math.PI / 6)},
      {x: 0.18 * r * Math.cos(11 * Math.PI / 6), y: 0.18 * r * Math.sin(11 * Math.PI / 6)},
    );
    zone_r = 0.43 * r;
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
      setZoneFill(ctx, zone);
      ctx.beginPath();
      ctx.lineTo(inner_r * Math.cos(a * i + Math.PI / 6), inner_r * Math.sin(a * i + Math.PI / 6));
      ctx.lineTo(inner_r * Math.cos(a * i + Math.PI / 2), inner_r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(r * Math.cos(a * i + Math.PI / 2), r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(r * Math.cos(a * i + Math.PI / 6), r * Math.sin(a * i + Math.PI / 6));
      ctx.closePath();
      ctx.stroke();
      ctx.fill();
      const rotation = a * (1 + i);
      ctx.rotate(rotation);
      const theta = Math.PI / 12;
      drawRisqZone(ctx, game, zone, black_text, zone_r, rotation,
        {x: 0.73 * r * Math.cos(0), y: 0.76 * r * Math.sin(0)},
        {x: 0.53 * r * Math.cos(-theta), y: 0.53 * r * Math.sin(-theta)},
        {x: 0.53 * r * Math.cos(theta), y: 0.53 * r * Math.sin(theta)},
      );
      ctx.rotate(-rotation);
    }
    ctx.translate(-space.center.x, -space.center.y);
  }
}

/** Returns the fill color for the input space */
export function getSpaceFill(space: RisqSpace, check_hover = true): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (!!space) {
    color.setColor(90, 90, 90, 0.8);
    if (space.visibility > 0) {
      // TODO: background color of space based on ownership and terrain
      color.setColor(10, 120, 10, 0.8);
      if (check_hover && space.hovered) {
        if (space.clicked) {
          color.addColor(210, 210, 210, 0.4);
        } else {
          color.addColor(190, 190, 190, 0.2);
        }
      }
    } else if (space.hovered) {
      color.addColor(150, 150, 150, 0.1);
    }
  }
  return color;
}
