import { ColorRGB } from '../../../../scripts/color_rgb';
import { DEV } from '../../../../scripts/util';
import { drawHexagon, drawText } from '../../util/canvas_util';
import type { Point2D } from '../../util/objects2d';
import type { DwgRisq } from './risq';
import type { RisqSpace } from './risq_data';
import { RisqResourceType, RisqVisibilityLevel } from './risq_data';
import { resourceTypeImage } from './risq_resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from './risq_unit';
import { RisqViewMode, spaceOwnerColor } from './risq_view_mode';
import { INNER_ZONE_MULTIPLIER, drawRisqZone, getZoneFill } from './risq_zone';

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
  // height of a 'row' in the inset box (up to 4 rows)
  inset_row: number;
  draw_detail: DrawRisqSpaceDetail;
  view_mode: RisqViewMode;
}

const space_line_width: Record<DrawRisqSpaceDetail, number> = {
  [DrawRisqSpaceDetail.OWNERSHIP]: 3,
  [DrawRisqSpaceDetail.SPACE_DETAILS]: 2,
  [DrawRisqSpaceDetail.ZONE_DETAILS]: 1.2,
};

/** Draws the input risq space */
export function drawRisqSpace(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
) {
  const owner_color = spaceOwnerColor(space, game.getGame()?.players ?? []);
  ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
  const fill = getSpaceFill(space, config.view_mode, owner_color);
  ctx.fillStyle = fill.getString();
  ctx.lineWidth = space_line_width[config.draw_detail];
  drawHexagon(ctx, space.center, config.hex_r);
  if (DEV) {
    drawText(ctx, space.coordinate.x + ', ' + space.coordinate.y, {
      p: space.center,
      w: 1.5 * config.hex_r,
      fill_style: 'rgba(0, 0, 0, 0.8)',
      align: 'center',
      baseline: 'middle',
      font: `${0.6 * config.inset_row}px serif`,
    });
  }
  ctx.textAlign = 'left';
  const black_text = fill.getBrightness() > 0.5;
  drawSpaceContent(ctx, game, space, config, black_text, owner_color);
  if (space.visibility === RisqVisibilityLevel.FOG) {
    ctx.strokeStyle = 'transparent';
    ctx.fillStyle = 'rgba(40, 45, 55, 0.5)';
    drawHexagon(ctx, space.center, config.hex_r);
  }
}

function drawSpaceContent(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean,
  owner_color: ColorRGB | undefined
) {
  if (config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP) {
    return; // ownership and terrain indicated by space fill color
  } else if (config.draw_detail === DrawRisqSpaceDetail.SPACE_DETAILS) {
    if (space.visibility < RisqVisibilityLevel.FOG) {
      return;
    }
    let building_img = game.getIcon('icons/building64');
    let villager_img = game.getIcon('icons/villager64');
    let unit_img = game.getIcon('icons/unit64');
    const gold_img = game.getIcon(resourceTypeImage(RisqResourceType.GOLD));
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
    let y = space.center.y - 0.5 * config.inset_h;
    const draw_count_row = (img: CanvasImageSource, count: string) => {
      ctx.drawImage(img, xs, y, config.inset_row, config.inset_row);
      ctx.fillText(`: ${count}`, xs + config.inset_row + 2, y, config.inset_w - config.inset_row - 2);
      y += config.inset_row + 2;
    };
    if (config.view_mode === RisqViewMode.OWNERSHIP) {
      draw_count_row(gold_img, space.gold_income?.toString() ?? '??');
      return;
    }
    if (config.view_mode === RisqViewMode.ALL) {
      draw_count_row(building_img, space.buildings?.size.toString() ?? '0');
    }
    if (config.view_mode !== RisqViewMode.MILITARY) {
      const resources = [...(space.total_resources?.entries() ?? [])].filter(([, amount]) => amount > 0);
      for (const [i, [resource_type]] of resources.entries()) {
        ctx.drawImage(
          game.getIcon(resourceTypeImage(resource_type)),
          xs + i * (config.inset_row + 2),
          y,
          config.inset_row,
          config.inset_row
        );
      }
      if (resources.length > 0) {
        y += config.inset_row + 2;
      }
    }
    if (config.view_mode === RisqViewMode.RESOURCE) {
      if (space.visibility >= RisqVisibilityLevel.GOOD) {
        draw_count_row(villager_img, space.num_villager_units?.toString() ?? '0');
      }
      return;
    }
    if (space.visibility === RisqVisibilityLevel.POOR) {
      const combo_icon = game
        .getImageCache()
        .getImage(comboUnitIconKey(!black_text), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
          drawComboUnitIcon(combo_ctx, villager_img, unit_img)
        );
      if (combo_icon) {
        draw_count_row(combo_icon, space.unit_count?.toString() ?? '0');
      }
    } else if (space.visibility >= RisqVisibilityLevel.GOOD) {
      if (config.view_mode === RisqViewMode.ALL) {
        draw_count_row(villager_img, space.num_villager_units?.toString() ?? '0');
      }
      draw_count_row(unit_img, space.num_military_units?.toString() ?? '0');
    }
  } else if (config.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
    if (space.visibility < RisqVisibilityLevel.FOG || !space.zones) {
      return;
    }
    ctx.translate(space.center.x, space.center.y);
    let zone = space.zones[1][1];
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    ctx.lineWidth = 0.1;
    ctx.fillStyle = getZoneFill(zone, config.view_mode, owner_color).getString();
    const r = config.hex_r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    let zone_r = 0.45 * r;
    drawHexagon(ctx, { x: 0, y: 0 }, inner_r);
    drawRisqZone(
      ctx,
      game,
      zone,
      space.visibility,
      config.view_mode,
      black_text,
      zone_r,
      0,
      {
        x: 0.18 * r * Math.cos((3 * Math.PI) / 6),
        y: 0.18 * r * Math.sin((3 * Math.PI) / 6),
      },
      {
        x: 0.18 * r * Math.cos((7 * Math.PI) / 6),
        y: 0.18 * r * Math.sin((7 * Math.PI) / 6),
      },
      {
        x: 0.18 * r * Math.cos((11 * Math.PI) / 6),
        y: 0.18 * r * Math.sin((11 * Math.PI) / 6),
      }
    );
    zone_r = 0.43 * r;
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
      ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
      ctx.fillStyle = getZoneFill(zone, config.view_mode, owner_color).getString();
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
      drawRisqZone(
        ctx,
        game,
        zone,
        space.visibility,
        config.view_mode,
        black_text,
        zone_r,
        rotation,
        { x: 0.73 * r * Math.cos(0), y: 0.76 * r * Math.sin(0) },
        { x: 0.53 * r * Math.cos(-theta), y: 0.53 * r * Math.sin(-theta) },
        { x: 0.53 * r * Math.cos(theta), y: 0.53 * r * Math.sin(theta) }
      );
      ctx.rotate(-rotation);
    }
    ctx.translate(-space.center.x, -space.center.y);
  }
}

/** Returns the fill color for the input space */
export function getSpaceFill(
  space: RisqSpace,
  view_mode: RisqViewMode = RisqViewMode.ALL,
  owner_color: ColorRGB | undefined = undefined,
  check_hover = true
): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (!!space) {
    color.setColor(90, 90, 90, 0.8);
    if (space.visibility > 0) {
      if (view_mode === RisqViewMode.OWNERSHIP) {
        if (owner_color) {
          color.setColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), 0.85);
        } else {
          color.setColor(90, 90, 90, 0.85);
        }
      } else {
        color.setColor(10, 120, 10, 0.8);
        if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
          color.addColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), 0.25);
        }
      }
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
