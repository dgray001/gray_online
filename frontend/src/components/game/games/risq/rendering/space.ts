import { ColorRGB } from '../../../../../scripts/color_rgb';
import { DEV } from '../../../../../scripts/util';
import { drawHexagon, drawText } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import type { RisqDrawHost } from './draw_host';
import type { RisqSpace } from '../model/types';
import { RisqResourceType, RisqUnitType, RisqVisibilityLevel } from '../model/types';
import { resourceTypeImage } from './assets/resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from './assets/unit';
import { drawSpaceOverviewDots } from './overview_dots';
import { FOG_OVERLAY_IMAGE, RisqViewMode, spaceOwnerColor } from './terrain';
import {
  INNER_ZONE_MULTIPLIER,
  OUTER_ZONE_INDICES,
  zoneBuildingLocalOffset,
  zoneUnitSlotLocalOffsets,
} from './zones/geometry';
import { drawForestTrees, drawRisqZone, getSpaceTerrainImage, getZoneFill, isEmptyPlot } from './zones/draw';
import type { RisqEntityPass } from './zones/draw';
import { getZoneUnitSlots } from './zones/slots';
import { drawZoneCorpses } from './zones/corpses';

export enum DrawRisqSpaceDetail {
  OWNERSHIP,
  SPACE_DETAILS,
  ZONE_DETAILS,
}

export declare interface DrawRisqSpaceConfig {
  hex_r: number;
  inset_w: number;
  inset_h: number;
  // height of a 'row' in the inset box (up to 4 rows)
  inset_row: number;
  inset_gap: number;
  draw_detail: DrawRisqSpaceDetail;
  view_mode: RisqViewMode;
  rotation: number;
  entity_pass?: RisqEntityPass;
}

export const SPACE_BORDER_REFERENCE_RADIUS = 60;
const space_line_width: Record<DrawRisqSpaceDetail, number> = {
  [DrawRisqSpaceDetail.OWNERSHIP]: 2,
  [DrawRisqSpaceDetail.SPACE_DETAILS]: 1.4,
  [DrawRisqSpaceDetail.ZONE_DETAILS]: 0.8,
};

/** Draws a hex-cut image (see scripts/cut_hex_texture.py) stretched to fill a hexagon of radius r */
export function drawHexImage(ctx: CanvasRenderingContext2D, img: CanvasImageSource, c: Point2D, r: number): void {
  const w = Math.sqrt(3) * r;
  const h = 2 * r;
  ctx.drawImage(img, c.x - 0.5 * w, c.y - 0.5 * h, w, h);
}

export function borderStrokeStyle(owner_color: ColorRGB | undefined, alpha: number): string {
  return owner_color
    ? `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, ${alpha})`
    : `rgba(255, 255, 255, ${alpha})`;
}

export function fillHexOverlay(ctx: CanvasRenderingContext2D, c: Point2D, r: number, fill_style: string): void {
  const prev_stroke = ctx.strokeStyle;
  ctx.strokeStyle = 'transparent';
  ctx.fillStyle = fill_style;
  drawHexagon(ctx, c, r);
  ctx.strokeStyle = prev_stroke;
}

export function drawRisqSpaceBase(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): boolean {
  const owner_color = spaceOwnerColor(space.ownership, game.getGame()?.players ?? []);
  const region = game.session.getRegionForSpace(space.coordinate_key);
  const region_owned = (region?.owner ?? -1) >= 0;
  ctx.strokeStyle = 'transparent';
  ctx.lineWidth = space_line_width[config.draw_detail] * (config.hex_r / SPACE_BORDER_REFERENCE_RADIUS);
  let black_text = false;
  if (config.view_mode === RisqViewMode.OWNERSHIP || space.visibility === RisqVisibilityLevel.UNEXPLORED) {
    const region_hovered = !!region && region === game.hover.hoveredRegion();
    const region_clicked =
      !!region && (game.selection.isRegionSelected(region) || (region_hovered && !!game.hover.space()?.clicked));
    const fill = getSpaceFill(space, config.view_mode, owner_color, true, region_hovered, region_clicked);
    if (region_owned) {
      fill.dBrightness(-0.22);
    }
    ctx.fillStyle = fill.getString();
    drawHexagon(ctx, space.center, config.hex_r);
    black_text = fill.getBrightness() > 0.5;
  } else {
    const override_zones = config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP ? [] : (space.zones?.flat() ?? []);

    if (game.getGame()?.background_image) {
      ctx.globalAlpha = 0.5;
    }
    drawHexImage(ctx, getSpaceTerrainImage(game, space.terrain_id, override_zones), space.center, config.hex_r);
    ctx.globalAlpha = 1.0;

    if (config.view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
      const tint = `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, ${region_owned ? 0.45 : 0.25})`;
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
  if (
    config.draw_detail !== DrawRisqSpaceDetail.OWNERSHIP &&
    !isOverviewViewMode(config.view_mode) &&
    space.visibility >= RisqVisibilityLevel.FOG
  ) {
    ctx.translate(space.center.x, space.center.y);
    for (const zone of space.zones?.flat() ?? []) {
      drawZoneCorpses(ctx, game, zone, space.visibility, config.hex_r, config.rotation);
    }
    for (const zone of space.zones?.flat() ?? []) {
      drawForestTrees(ctx, game, zone, config.view_mode, config.hex_r, config.rotation);
    }
    ctx.translate(-space.center.x, -space.center.y);
  }
  return black_text;
}

export function drawRisqSpaceContent(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean
): void {
  const owner_color = spaceOwnerColor(space.ownership, game.getGame()?.players ?? []);
  if (DEV && (!config.entity_pass || config.entity_pass === 'unselected')) {
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
  if (config.entity_pass === 'terrain' || config.entity_pass === 'selected') {
    return;
  }
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

export function isOverviewViewMode(view_mode?: RisqViewMode): boolean {
  return view_mode === RisqViewMode.OWNERSHIP;
}

const overview_space_line_width: Record<DrawRisqSpaceDetail, number> = {
  [DrawRisqSpaceDetail.OWNERSHIP]: 1.6,
  [DrawRisqSpaceDetail.SPACE_DETAILS]: 1.1,
  [DrawRisqSpaceDetail.ZONE_DETAILS]: 0.6,
};

export function spaceBorderWidth(
  config: Pick<DrawRisqSpaceConfig, 'hex_r' | 'draw_detail'> & { view_mode?: RisqViewMode }
): number {
  const widths = isOverviewViewMode(config.view_mode) ? overview_space_line_width : space_line_width;
  return widths[config.draw_detail] * (config.hex_r / SPACE_BORDER_REFERENCE_RADIUS);
}

function drawSpaceContent(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean,
  owner_color: ColorRGB | undefined
): void {
  if (config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP) {
    drawSpaceOverviewDots(ctx, game, space, config);
    return;
  } else if (config.draw_detail === DrawRisqSpaceDetail.SPACE_DETAILS && config.entity_pass !== 'selected') {
    if (space.visibility < RisqVisibilityLevel.FOG || config.entity_pass === 'terrain') {
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
  } else if (
    config.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS ||
    (config.draw_detail === DrawRisqSpaceDetail.SPACE_DETAILS && config.entity_pass === 'selected')
  ) {
    if (space.visibility < RisqVisibilityLevel.FOG || !space.zones) {
      return;
    }
    ctx.translate(space.center.x, space.center.y);
    let zone = space.zones[1][1];
    ctx.strokeStyle = borderStrokeStyle(owner_color, 0.9);
    ctx.lineWidth = 0.1;
    ctx.fillStyle = getZoneFill(
      zone,
      config.view_mode,
      owner_color,
      true,
      1,
      isEmptyPlot(game.planning, zone)
    ).getString();
    const r = config.hex_r;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    const active_player_id = game.getPlayerId();
    const center_slots = getZoneUnitSlots(
      zone,
      active_player_id,
      game.viewport.centerMaxSlots(),
      game.viewport.edgeMaxSlots(),
      config.view_mode
    );
    const center_count = space.visibility === RisqVisibilityLevel.POOR ? 1 : center_slots.length;
    if (!config.entity_pass || config.entity_pass === 'terrain') {
      drawHexagon(ctx, { x: 0, y: 0 }, inner_r);
    }
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
      zoneUnitSlotLocalOffsets(zone.coordinate, r, center_count, game.viewport.unitRadius()),
      active_player_id,
      config.entity_pass
    );
    const a = Math.PI / 3;
    for (let i = 0; i < 6; i++) {
      const direction_vector = OUTER_ZONE_INDICES[i];
      zone = space.zones[direction_vector.x][direction_vector.y];
      ctx.strokeStyle = borderStrokeStyle(owner_color, 0.9);
      ctx.fillStyle = getZoneFill(
        zone,
        config.view_mode,
        owner_color,
        true,
        1,
        isEmptyPlot(game.planning, zone)
      ).getString();
      ctx.beginPath();
      ctx.lineTo(inner_r * Math.cos(a * i + Math.PI / 6), inner_r * Math.sin(a * i + Math.PI / 6));
      ctx.lineTo(inner_r * Math.cos(a * i + Math.PI / 2), inner_r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(r * Math.cos(a * i + Math.PI / 2), r * Math.sin(a * i + Math.PI / 2));
      ctx.lineTo(r * Math.cos(a * i + Math.PI / 6), r * Math.sin(a * i + Math.PI / 6));
      ctx.closePath();
      if (!config.entity_pass || config.entity_pass === 'terrain') {
        ctx.stroke();
        ctx.fill();
      }
      const rotation = a * (1 + i);
      ctx.rotate(rotation);
      const edge_slots = getZoneUnitSlots(
        zone,
        active_player_id,
        game.viewport.centerMaxSlots(),
        game.viewport.edgeMaxSlots(),
        config.view_mode
      );
      const edge_count = space.visibility === RisqVisibilityLevel.POOR ? 1 : edge_slots.length;
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
        zoneUnitSlotLocalOffsets(zone.coordinate, r, edge_count, game.viewport.unitRadius()),
        active_player_id,
        config.entity_pass
      );
      ctx.rotate(-rotation);
    }
    ctx.translate(-space.center.x, -space.center.y);
  }
}

function formatResourceAmount(amount: number): string {
  return amount >= 1000 ? `${(amount / 1000).toFixed(1)}k` : Math.round(amount).toString();
}

/** Inset row text, shrinking the font to fit the width rather than squishing it */
function drawInsetText(
  ctx: CanvasRenderingContext2D,
  text: string,
  p: Point2D,
  w: number,
  row_h: number,
  fill_style: string
): void {
  const center = { x: p.x, y: p.y + 0.5 * row_h };
  drawText(ctx, text, { p: center, w, fill_style, baseline: 'middle', font: `bold ${row_h}px serif` });
}

/** Draws the space's resource totals in at most two rows and returns the y below them */
function drawResourceAmountRows(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  xs: number,
  y: number,
  fill_style: string
): number {
  const resources = [...(space.total_resources?.entries() ?? [])].filter(([, amount]) => amount > 0);
  const rows = Math.min(resources.length, 2);
  const row_step = config.inset_row + config.inset_gap;
  const half_w = (config.inset_w - config.inset_gap) / 2;
  for (const [i, [resource_type, amount]] of resources.sort(([a], [b]) => a - b).entries()) {
    const row = i % rows;
    const cell_w = Math.ceil((resources.length - row) / rows) > 1 ? half_w : config.inset_w;
    const x = xs + Math.floor(i / rows) * (half_w + config.inset_gap);
    const row_y = y + row * row_step;
    ctx.drawImage(game.getIcon(resourceTypeImage(resource_type)), x, row_y, config.inset_row, config.inset_row);
    const text_p = { x: x + config.inset_row + config.inset_gap, y: row_y };
    const text_w = cell_w - config.inset_row - config.inset_gap;
    drawInsetText(ctx, `: ${formatResourceAmount(amount)}`, text_p, text_w, config.inset_row, fill_style);
  }
  return y + rows * row_step;
}

declare interface MilitaryCountCell {
  color?: ColorRGB;
  count: number;
}

function militaryCountsByPlayer(space: RisqSpace): Map<number, number> {
  const counts = new Map<number, number>();
  for (const unit of space.units?.values() ?? []) {
    if (unit.unit_type !== RisqUnitType.ECONOMIC) {
      counts.set(unit.player_id, (counts.get(unit.player_id) ?? 0) + 1);
    }
  }
  return counts;
}

/** One cell per player, yours first; past 8 players it collapses to your cell plus one for everyone else */
function militaryCountCells(space: RisqSpace, game: RisqDrawHost): MilitaryCountCell[] {
  const counts = militaryCountsByPlayer(space);
  const my_id = game.getPlayerId();
  const players = game.getGame()?.players ?? [];
  const ids = [...counts.keys()].sort((a, b) => (a === my_id ? -1 : b === my_id ? 1 : a - b));
  if (ids.length <= 8) {
    return ids.map((id) => ({ color: players[id]?.color, count: counts.get(id)! }));
  }
  const others = ids.filter((id) => id !== my_id).reduce((sum, id) => sum + counts.get(id)!, 0);
  const mine = counts.has(my_id) ? [{ color: players[my_id]?.color, count: counts.get(my_id)! }] : [];
  return [...mine, { count: others }];
}

/** Draws per-player military counts and returns the y below them */
function drawMilitaryCountRows(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  xs: number,
  y: number,
  theme: { icon: CanvasImageSource; fill_style: string }
): number {
  const cells = militaryCountCells(space, game);
  const columns = cells.length > 4 ? 2 : 1;
  const row_step = config.inset_row + config.inset_gap;
  const cell_w = columns === 2 ? (config.inset_w - config.inset_gap) / 2 : config.inset_w;
  const base_icon = game.getIcon('icons/unit64');
  for (const [i, cell] of cells.entries()) {
    const x = xs + (i % columns) * (cell_w + config.inset_gap);
    const row_y = y + Math.floor(i / columns) * row_step;
    const tinted = cell.color && game.getImageCache().getTintedIcon('icons/unit64', base_icon, 64, cell.color);
    ctx.drawImage(tinted || theme.icon, x, row_y, config.inset_row, config.inset_row);
    const text_p = { x: x + config.inset_row + config.inset_gap, y: row_y };
    const text_w = cell_w - config.inset_row - config.inset_gap;
    drawInsetText(ctx, `: ${cell.count}`, text_p, text_w, config.inset_row, theme.fill_style);
  }
  return y + Math.ceil(cells.length / columns) * row_step;
}

function drawSpaceDetailsContent(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean
): void {
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
  const draw_count_row = (img: CanvasImageSource, count: string): void => {
    ctx.drawImage(img, xs, y, config.inset_row, config.inset_row);
    const text_p = { x: xs + config.inset_row + config.inset_gap, y };
    const text_w = config.inset_w - config.inset_row - config.inset_gap;
    drawInsetText(ctx, `: ${count}`, text_p, text_w, config.inset_row, black_text ? 'black' : 'white');
    y += config.inset_row + config.inset_gap;
  };
  if (space.visibility === RisqVisibilityLevel.POOR) {
    ctx.drawImage(
      game.getIcon(`icons/no_eye${black_text ? '' : '_white'}64`),
      space.center.x - 0.5 * config.inset_row,
      y,
      config.inset_row,
      config.inset_row
    );
    y += config.inset_row + config.inset_gap;
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
    if (config.view_mode === RisqViewMode.MILITARY && space.num_military_units) {
      const theme = { icon: unit_img, fill_style: black_text ? 'black' : 'white' };
      y = drawMilitaryCountRows(ctx, game, space, config, xs, y, theme);
    } else {
      draw_count_row(unit_img, space.num_military_units?.toString() ?? '0');
    }
  }
  if (config.view_mode === RisqViewMode.RESOURCE) {
    y = drawResourceAmountRows(ctx, game, space, config, xs, y, black_text ? 'black' : 'white');
    draw_count_row(gold_img, `${space.gold_income ?? '??'}/turn`);
    return;
  }
  if (config.view_mode !== RisqViewMode.MILITARY) {
    const resources = [...(space.total_resources?.entries() ?? [])].filter(([, amount]) => amount > 0);
    for (const [i, [resource_type]] of resources.entries()) {
      ctx.drawImage(
        game.getIcon(resourceTypeImage(resource_type)),
        xs + i * (config.inset_row + config.inset_gap),
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
  check_hover = true,
  force_hover = false,
  force_clicked: boolean = false
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
      if (check_hover && (space.hovered || force_hover || force_clicked)) {
        if (space.clicked || force_clicked) {
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
