import type { ColorRGB } from '../../../../../scripts/color_rgb';
import { drawCircle, drawRect, drawText } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import { addPoint2D } from '../../../util/objects2d';
import { hasBuildings, hasEconomy } from '../application/session';
import type { RisqSpace, RisqZone } from '../model/types';
import { RisqResourceType, RisqUnitType, RisqVisibilityLevel } from '../model/types';
import type { RisqImageCache } from './assets/image_cache';
import { isForestResource, mapResourceTypeIcon, resourceType, resourceTypeImage } from './assets/resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon } from './assets/unit';
import type { RisqDrawHost } from './board/board_renderer';
import type { DrawRisqSpaceConfig } from './space';
import { RisqViewMode, buildingVisibleInViewMode, unitVisibleInViewMode } from './terrain';
import {
  centerUnitMaxZoom,
  centerUnitSlotCapacity,
  edgeUnitSlotCapacity,
  findOuterZoneIndex,
  zoneCenterOffset,
  zoneUnitSlotOffsets,
  zoneVertices,
} from './zones/geometry';
import { buildZoneUnitSlots } from './zones/slots';

export const RESOURCE_DOT_COLORS: Record<RisqResourceType | 'selected', string> = {
  [RisqResourceType.ERROR]: 'black',
  [RisqResourceType.FOOD]: '#a0c692',
  [RisqResourceType.WOOD]: 'rgb(25, 85, 35)',
  [RisqResourceType.STONE]: 'rgb(160, 160, 160)',
  [RisqResourceType.GOLD]: 'rgb(220, 180, 35)',
  selected: 'white',
};

export function resourceDot(cache: RisqImageCache, type: RisqResourceType | 'selected'): HTMLCanvasElement | undefined {
  const size = 16;
  return cache.getImage(`minimap_resource_${type}`, size, [], (ctx: CanvasRenderingContext2D): void => {
    ctx.fillStyle = RESOURCE_DOT_COLORS[type];
    ctx.strokeStyle = 'transparent';
    drawCircle(ctx, { x: size / 2, y: size / 2 }, size / 2);
  });
}

function drawForestZone(ctx: CanvasRenderingContext2D, zone: RisqZone, center: Point2D, hex_r: number): void {
  ctx.save();
  ctx.fillStyle = RESOURCE_DOT_COLORS[RisqResourceType.WOOD];
  ctx.beginPath();
  for (const point of zoneVertices(zone.coordinate, hex_r)) {
    ctx.lineTo(center.x + point.x, center.y + point.y);
  }
  ctx.closePath();
  ctx.fill();
  ctx.restore();
}

/** Draws a resource zone as a colored dot, or a filled zone for forests, around the space's center */
export function drawZoneResourceDot(
  ctx: CanvasRenderingContext2D,
  cache: RisqImageCache,
  zone: RisqZone,
  center: Point2D,
  hex_r: number,
  scale: number
): void {
  const resource = zone.resource;
  if (!resource) {
    return;
  }
  if (isForestResource(resource)) {
    drawForestZone(ctx, zone, center, hex_r);
    return;
  }
  const image = resourceDot(cache, resourceType(resource));
  const point = addPoint2D(center, zoneCenterOffset(zone.coordinate, hex_r));
  const radius = scale * Math.max(1, 0.175 * hex_r);
  if (image) {
    ctx.drawImage(image, point.x - radius, point.y - radius, radius * 2, radius * 2);
  }
}

const dot_offsets = new WeakMap<RisqZone, { view_mode: RisqViewMode; offsets: Map<number, Point2D> }>();

/** Per-unit offsets in hex-radius units for overview dots, cached per zone and view mode */
export function unitDotOffsets(
  zone: RisqZone,
  active_player_id: number,
  view_mode: RisqViewMode
): Map<number, Point2D> {
  const cached = dot_offsets.get(zone);
  if (cached?.view_mode === view_mode) {
    return cached.offsets;
  }
  const unit_r = 1 / (2 * centerUnitMaxZoom(1, 1));
  const is_center = findOuterZoneIndex(zone.coordinate) === -1;
  const capacity = is_center ? centerUnitSlotCapacity(1, unit_r) : edgeUnitSlotCapacity(1, unit_r);
  const slots = buildZoneUnitSlots(zone, active_player_id, capacity, view_mode);
  const slot_offsets = zoneUnitSlotOffsets(zone.coordinate, 1, slots.length, unit_r);
  const offsets = new Map<number, Point2D>();
  for (const [i, slot] of slots.entries()) {
    for (const group of slot) {
      for (const id of group.units) {
        offsets.set(id, slot_offsets[i]);
      }
    }
  }
  dot_offsets.set(zone, { view_mode, offsets });
  return offsets;
}

function fillUprightSquare(ctx: CanvasRenderingContext2D, p: Point2D, r: number, rotation: number, fill: string): void {
  ctx.save();
  ctx.fillStyle = fill;
  ctx.translate(p.x, p.y);
  ctx.rotate(-rotation);
  ctx.fillRect(-r, -r, 2 * r, 2 * r);
  ctx.restore();
}

/** Player-colored dots for the space's visible units and buildings; selected ones are drawn white and larger */
function drawEntityDots(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): void {
  const selected_pass = config.entity_pass === 'selected';
  const radius = (selected_pass ? 0.25 : 0.175) * config.hex_r;
  const players = game.getGame()?.players ?? [];
  const dot = (offset: Point2D, player_id: number): void => {
    const point = addPoint2D(space.center, { x: config.hex_r * offset.x, y: config.hex_r * offset.y });
    const color = selected_pass ? 'white' : (players[player_id]?.color.getString() ?? 'white');
    fillUprightSquare(ctx, point, radius, config.rotation, color);
  };
  for (const zone of space.zones?.flat() ?? []) {
    const offsets = unitDotOffsets(zone, game.getPlayerId(), config.view_mode);
    for (const unit of zone.units.values()) {
      const shown = unit.garrisoned_in === undefined && unitVisibleInViewMode(unit.unit_type, config.view_mode);
      if (shown && game.selection.isUnitSelected(unit.internal_id) === selected_pass) {
        dot(offsets.get(unit.internal_id) ?? zoneCenterOffset(zone.coordinate, 1), unit.player_id);
      }
    }
    const building = zone.building;
    const building_shown = !!building && buildingVisibleInViewMode(building, config.view_mode);
    if (building_shown && game.selection.isBuildingSelected(building.internal_id) === selected_pass) {
      dot(zoneCenterOffset(zone.coordinate, 1), building.player_id);
    }
  }
}

function drawResourceDots(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): void {
  const selected_resources: RisqZone[] = [];
  for (const zone of space.zones?.flat() ?? []) {
    if (!zone.resource) {
      continue;
    }
    drawZoneResourceDot(ctx, game.getImageCache(), zone, space.center, config.hex_r, 1);
    if (game.selection.isResourceSelected(zone.resource.internal_id)) {
      selected_resources.push(zone);
    }
  }
  for (const zone of selected_resources) {
    const image = resourceDot(game.getImageCache(), 'selected');
    const point = addPoint2D(space.center, zoneCenterOffset(zone.coordinate, config.hex_r));
    const radius = 0.25 * config.hex_r;
    if (image) {
      ctx.drawImage(image, point.x - radius, point.y - radius, radius * 2, radius * 2);
    }
  }
}

const INSET_ROWS = 4;
const OVERVIEW_DOT_VIEW_MODES = [RisqViewMode.ALL, RisqViewMode.RESOURCE, RisqViewMode.MILITARY];

/** Zoomed-out replacement for zone detail: entity and resource dots in the entity passes */
export function drawSpaceOverviewDots(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): void {
  if (space.visibility < RisqVisibilityLevel.FOG || !OVERVIEW_DOT_VIEW_MODES.includes(config.view_mode)) {
    return;
  }
  const show_resource_dots = config.view_mode !== RisqViewMode.MILITARY && config.view_mode !== RisqViewMode.OWNERSHIP;
  if (config.entity_pass === 'terrain') {
    if (show_resource_dots) {
      drawResourceDots(ctx, game, space, config);
    }
  } else {
    drawEntityDots(ctx, game, space, config);
  }
}

export function drawUnitCountBadge(ctx: CanvasRenderingContext2D, count: number, p: Point2D, s: number): void {
  const text = `×${count}`;
  const font_size = Math.max(10, 0.22 * s);
  ctx.font = `bold ${font_size}px serif`;
  const badge_w = ctx.measureText(text).width + 6;
  const badge_h = font_size + 4;
  const badge_p = { x: p.x + s - badge_w, y: p.y + s - badge_h };
  ctx.fillStyle = 'rgba(0, 0, 0, 0.75)';
  ctx.strokeStyle = 'transparent';
  drawRect(ctx, badge_p, badge_w, badge_h, 3);
  drawText(ctx, text, {
    p: { x: badge_p.x + 0.5 * badge_w, y: badge_p.y + 0.5 * badge_h },
    w: badge_w,
    fill_style: 'white',
    align: 'center',
    baseline: 'middle',
    font: `bold ${font_size}px serif`,
  });
}

function formatResourceAmount(amount: number): string {
  return amount >= 1000 ? `${(amount / 1000).toFixed(1)}k` : Math.round(amount).toString();
}

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

function drawResourceAmountRows(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  xs: number,
  y: number,
  max_rows: number,
  fill_style: string
): number {
  const resources = [...(space.total_resources?.entries() ?? [])].filter(([, amount]) => amount > 0);
  const row_step = config.inset_row + config.inset_gap;
  const rows = Math.min(resources.length, max_rows);
  const half_w = (config.inset_w - config.inset_gap) / 2;
  for (const [i, [resource_type, amount]] of resources.sort(([a], [b]) => a - b).entries()) {
    const row = i % rows;
    const cell_w = Math.ceil((resources.length - row) / rows) > 1 ? half_w : config.inset_w;
    const x = xs + Math.floor(i / rows) * (half_w + config.inset_gap);
    const row_y = y + row * row_step;
    ctx.drawImage(mapResourceTypeIcon(game, resource_type), x, row_y, config.inset_row, config.inset_row);
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

export function drawSpaceDetailsContent(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean
): void {
  const economy = hasEconomy(game.getGame());
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
  let rows_used = 0;
  const draw_count_row = (img: CanvasImageSource, count: string): void => {
    ctx.drawImage(img, xs, y, config.inset_row, config.inset_row);
    const text_p = { x: xs + config.inset_row + config.inset_gap, y };
    const text_w = config.inset_w - config.inset_row - config.inset_gap;
    drawInsetText(ctx, `: ${count}`, text_p, text_w, config.inset_row, black_text ? 'black' : 'white');
    y += config.inset_row + config.inset_gap;
    rows_used++;
  };
  const draw_nonzero_row = (img: CanvasImageSource, count?: number): void => {
    if (count) {
      draw_count_row(img, count.toString());
    }
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
    rows_used++;
  }
  if (config.view_mode === RisqViewMode.OWNERSHIP) {
    draw_count_row(gold_img, space.gold_income?.toString() ?? '??');
    return;
  }
  if (config.view_mode === RisqViewMode.ALL && hasBuildings(game.getGame())) {
    draw_nonzero_row(building_img, space.buildings?.size);
  }
  if (config.view_mode === RisqViewMode.RESOURCE) {
    if (space.visibility >= RisqVisibilityLevel.GOOD) {
      draw_nonzero_row(villager_img, space.num_villager_units);
    }
  } else if (space.visibility === RisqVisibilityLevel.POOR) {
    const combo_icon = economy
      ? game
          .getImageCache()
          .getImage(comboUnitIconKey(!black_text), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
            drawComboUnitIcon(combo_ctx, villager_img, unit_img)
          )
      : unit_img;
    if (combo_icon) {
      draw_nonzero_row(combo_icon, space.unit_count);
    }
  } else if (space.visibility >= RisqVisibilityLevel.GOOD) {
    if (config.view_mode === RisqViewMode.ALL && economy) {
      draw_nonzero_row(villager_img, space.num_villager_units);
    }
    if (config.view_mode === RisqViewMode.MILITARY && space.num_military_units) {
      const theme = { icon: unit_img, fill_style: black_text ? 'black' : 'white' };
      y = drawMilitaryCountRows(ctx, game, space, config, xs, y, theme);
    } else {
      draw_nonzero_row(unit_img, space.num_military_units);
    }
  }
  if (config.view_mode === RisqViewMode.RESOURCE) {
    const rows_left = INSET_ROWS - rows_used - 1;
    y = drawResourceAmountRows(ctx, game, space, config, xs, y, rows_left, black_text ? 'black' : 'white');
    draw_count_row(mapResourceTypeIcon(game, RisqResourceType.GOLD), `${space.gold_income ?? '??'}/turn`);
    return;
  }
}
