import { ColorRGB } from '../../../../scripts/color_rgb';
import { DEV } from '../../../../scripts/util';
import { drawHexagon, drawText } from '../../util/canvas_util';
import type { Point2D } from '../../util/objects2d';
import type { DwgRisq } from './risq';
import type { RisqSpace } from './risq_data';
import { RisqResourceType, RisqVisibilityLevel } from './risq_data';
import { resourceTypeImage } from './risq_resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from './risq_unit';
import { FOG_OVERLAY_IMAGE, RisqViewMode, spaceOwnerColor, terrainImage } from './risq_terrain';
import {
  INNER_ZONE_MULTIPLIER,
  OUTER_ZONE_INDICES,
  drawRisqZone,
  getZoneFill,
  zoneBuildingLocalOffset,
  zoneUnitSlotLocalOffsets,
} from './risq_zone';

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
  rotation: number;
}

const space_line_width: Record<DrawRisqSpaceDetail, number> = {
  [DrawRisqSpaceDetail.OWNERSHIP]: 2,
  [DrawRisqSpaceDetail.SPACE_DETAILS]: 1.2,
  [DrawRisqSpaceDetail.ZONE_DETAILS]: 0.8,
};

/** Draws a hex-cut image (see scripts/cut_hex_texture.py) stretched to fill a hexagon of radius r */
export function drawHexImage(ctx: CanvasRenderingContext2D, img: CanvasImageSource, c: Point2D, r: number) {
  const w = Math.sqrt(3) * r;
  const h = 2 * r;
  ctx.drawImage(img, c.x - 0.5 * w, c.y - 0.5 * h, w, h);
}

export function borderStrokeStyle(owner_color: ColorRGB | undefined, alpha: number): string {
  return owner_color
    ? `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, ${alpha})`
    : `rgba(255, 255, 255, ${alpha})`;
}

export function fillHexOverlay(ctx: CanvasRenderingContext2D, c: Point2D, r: number, fill_style: string) {
  const prev_stroke = ctx.strokeStyle;
  ctx.strokeStyle = 'transparent';
  ctx.fillStyle = fill_style;
  drawHexagon(ctx, c, r);
  ctx.strokeStyle = prev_stroke;
}

/** Draws the input risq space */
export function drawRisqSpace(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
) {
  const owner_color = spaceOwnerColor(space.ownership, game.getGame()?.players ?? []);
  ctx.strokeStyle = 'transparent';
  ctx.lineWidth = space_line_width[config.draw_detail];
  let black_text = false;
  if (config.view_mode === RisqViewMode.OWNERSHIP || space.visibility === RisqVisibilityLevel.UNEXPLORED) {
    const fill = getSpaceFill(space, config.view_mode, owner_color);
    ctx.fillStyle = fill.getString();
    drawHexagon(ctx, space.center, config.hex_r);
    black_text = fill.getBrightness() > 0.5;
  } else {
    drawHexImage(ctx, game.getIcon(terrainImage(space.terrain_id)), space.center, config.hex_r);
    if (config.view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
      const tint = `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, 0.25)`;
      fillHexOverlay(ctx, space.center, config.hex_r, tint);
    }
    if (space.hovered) {
      fillHexOverlay(
        ctx,
        space.center,
        config.hex_r,
        space.clicked ? 'rgba(210, 210, 210, 0.4)' : 'rgba(190, 190, 190, 0.2)'
      );
    }
  }
  if (DEV) {
    ctx.translate(space.center.x, space.center.y);
    ctx.rotate(-config.rotation);
    ctx.translate(-space.center.x, -space.center.y);
    drawText(ctx, space.coordinate.x + ', ' + space.coordinate.y, {
      p: space.center,
      w: 1.5 * config.hex_r,
      fill_style: 'rgba(255, 255, 255, 0.8)',
      align: 'center',
      baseline: 'middle',
      font: `${0.6 * config.inset_row}px serif`,
    });
    ctx.translate(space.center.x, space.center.y);
    ctx.rotate(config.rotation);
    ctx.translate(-space.center.x, -space.center.y);
  }
  ctx.textAlign = 'left';
  drawSpaceContent(ctx, game, space, config, black_text, owner_color);
  if (config.draw_detail !== DrawRisqSpaceDetail.OWNERSHIP && space.visibility === RisqVisibilityLevel.SPY) {
    const badge_size = 0.35 * config.hex_r;
    ctx.drawImage(
      game.getIcon(`icons/eye${black_text ? '' : '_white'}64`),
      space.center.x - 0.5 * badge_size,
      space.center.y - config.hex_r + 2,
      badge_size,
      badge_size
    );
  }
  if (space.visibility === RisqVisibilityLevel.FOG) {
    ctx.globalAlpha = 0.55;
    drawHexImage(ctx, game.getIcon(FOG_OVERLAY_IMAGE), space.center, config.hex_r);
    ctx.globalAlpha = 1;
  }
}

export function drawRisqSpaceBorder(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
) {
  const owner_color = spaceOwnerColor(space.ownership, game.getGame()?.players ?? []);
  const line_width = space_line_width[config.draw_detail];
  ctx.fillStyle = 'transparent';
  ctx.strokeStyle = borderStrokeStyle(owner_color, 1);
  ctx.lineWidth = line_width;
  drawHexagon(ctx, space.center, config.hex_r - 0.5 * line_width);
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
    ctx.save();
    ctx.translate(space.center.x, space.center.y);
    ctx.rotate(-config.rotation);
    ctx.translate(-space.center.x, -space.center.y);
    try {
      drawSpaceDetailsContent(ctx, game, space, config, black_text);
    } finally {
      ctx.restore();
    }
  } else if (config.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS) {
    if (space.visibility < RisqVisibilityLevel.FOG || !space.zones) {
      return;
    }
    ctx.translate(space.center.x, space.center.y);
    let zone = space.zones[1][1];
    ctx.strokeStyle = borderStrokeStyle(owner_color, 0.9);
    ctx.lineWidth = 0.1;
    ctx.fillStyle = getZoneFill(zone, config.view_mode, owner_color).getString();
    const r = config.hex_r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const active_player_id = game.getPlayerId();
    drawHexagon(ctx, { x: 0, y: 0 }, inner_r);
    drawRisqZone(
      ctx,
      game,
      zone,
      space.visibility,
      config.view_mode,
      black_text,
      r,
      config.rotation,
      zoneBuildingLocalOffset(zone.coordinate, r),
      zoneUnitSlotLocalOffsets(zone.coordinate, r),
      active_player_id
    );
    const a = Math.PI / 3;
    for (let i = 0; i < 6; i++) {
      const direction_vector = OUTER_ZONE_INDICES[i];
      zone = space.zones[direction_vector.x][direction_vector.y];
      ctx.strokeStyle = borderStrokeStyle(owner_color, 0.9);
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
      drawRisqZone(
        ctx,
        game,
        zone,
        space.visibility,
        config.view_mode,
        black_text,
        r,
        rotation + config.rotation,
        zoneBuildingLocalOffset(zone.coordinate, r),
        zoneUnitSlotLocalOffsets(zone.coordinate, r),
        active_player_id
      );
      ctx.rotate(-rotation);
    }
    ctx.translate(-space.center.x, -space.center.y);
  }
}

function drawSpaceDetailsContent(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean
) {
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
  if (space.visibility === RisqVisibilityLevel.POOR) {
    ctx.drawImage(
      game.getIcon(`icons/no_eye${black_text ? '' : '_white'}64`),
      space.center.x - 0.5 * config.inset_row,
      y,
      config.inset_row,
      config.inset_row
    );
    y += config.inset_row + 2;
  }
  if (config.view_mode === RisqViewMode.OWNERSHIP) {
    draw_count_row(gold_img, space.gold_income?.toString() ?? '??');
    return;
  }
  if (config.view_mode === RisqViewMode.ALL) {
    draw_count_row(building_img, space.buildings?.size.toString() ?? '0');
  }
  if (config.view_mode === RisqViewMode.RESOURCE) {
    if (space.visibility >= RisqVisibilityLevel.GOOD) {
      draw_count_row(villager_img, space.num_villager_units?.toString() ?? '0');
    }
  } else if (space.visibility === RisqVisibilityLevel.POOR) {
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
    color.setColor(50, 50, 50, 0.8);
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
    } else if (check_hover && space.hovered) {
      color.addColor(150, 150, 150, 0.1);
    }
  }
  return color;
}
