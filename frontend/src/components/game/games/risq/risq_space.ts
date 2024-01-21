import {ColorRGB} from "../../../../scripts/color_rgb";
import {drawHexagon} from "../../util/canvas_util";
import {DwgRisq} from "./risq";
import {RisqSpace} from "./risq_data";

/** Config data for drawing a risq space */
export declare interface DrawRisqSpaceConfig {
  hex_r: number;
  inset_w: number;
  inset_h: number;
  inset_row: number;
}

/** Draws the input risq space */
export function drawRisqSpace(ctx: CanvasRenderingContext2D, game: DwgRisq,
  space: RisqSpace, config: DrawRisqSpaceConfig
) {
  const fill = getSpaceFill(space);
  ctx.fillStyle = fill.getString();
  drawHexagon(ctx, space.center, config.hex_r);
  if (space.visibility > 0) {
    let building_img = game.getIcon('icons/building64');
    let unit_img = game.getIcon('icons/unit64');
    if (fill.getBrightness() > 0.5) {
      ctx.fillStyle = 'black';
    } else {
      ctx.fillStyle = 'white';
      building_img = game.getIcon('icons/building_white64');
      unit_img = game.getIcon('icons/unit_white64');
    }
    ctx.textBaseline = 'top';
    const xs = space.center.x - 0.5 * config.inset_w;
    const y1 = space.center.y - 0.5 * config.inset_h;
    ctx.drawImage(building_img, xs, y1, config.inset_row, config.inset_row);
    ctx.fillText(`: ${space.buildings.size.toString()}`, xs + config.inset_row + 2, y1, config.inset_w - config.inset_row - 2);
    const y2 = space.center.y - 0.5 * config.inset_h + config.inset_row + 2;
    ctx.drawImage(unit_img, xs, y2, config.inset_row, config.inset_row);
    ctx.fillText(`: ${space.units.size.toString()}`, xs + config.inset_row + 2, y2, config.inset_w - config.inset_row - 2);
  }
}

/** Returns the fill color for the input space */
export function getSpaceFill(space: RisqSpace): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (!!space) {
    color.setColor(90, 90, 90, 0.8);
    if (space.visibility > 0) {
      color.setColor(10, 120, 10, 0.8);
      if (space.hovered) {
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