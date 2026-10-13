import { drawCircle } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import { addPoint2D } from '../../../util/objects2d';
import type { RisqSpace, RisqZone } from '../model/types';
import { RisqResourceType, RisqVisibilityLevel } from '../model/types';
import type { RisqImageCache } from './assets/image_cache';
import { isForestResource, resourceType } from './assets/resources';
import type { RisqDrawHost } from './draw_host';
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

const OVERVIEW_DOT_VIEW_MODES = [RisqViewMode.ALL, RisqViewMode.RESOURCE, RisqViewMode.MILITARY];

/** Zoomed-out replacement for zone detail: resource dots with the terrain, entity dots in the entity passes */
export function drawSpaceOverviewDots(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): void {
  if (space.visibility < RisqVisibilityLevel.FOG || !OVERVIEW_DOT_VIEW_MODES.includes(config.view_mode)) {
    return;
  }
  if (config.entity_pass === 'terrain') {
    const scale = config.view_mode === RisqViewMode.RESOURCE ? 1.4 : 1;
    for (const zone of config.view_mode === RisqViewMode.MILITARY ? [] : (space.zones?.flat() ?? [])) {
      drawZoneResourceDot(ctx, game.getImageCache(), zone, space.center, config.hex_r, scale);
    }
  } else {
    drawEntityDots(ctx, game, space, config);
  }
}
