import { getSettings } from '../../../../../scripts/settings_store';
import { ColorRGB } from '../../../../../scripts/color_rgb';
import { DEV } from '../../../../../scripts/util';
import { drawHexagon, drawText } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import type { RisqDrawHost } from './board/board_renderer';
import type { RisqSpace, RisqZone } from '../model/types';
import { RisqVisibilityLevel } from '../model/types';
import { drawSpaceDetailsContent, drawSpaceOverviewDots } from './overview_dots';
import { FOG_OVERLAY_IMAGE, RisqViewMode, buildingVisibleInViewMode, spaceOwnerColor } from './terrain';
import {
  BUILDING_CIRCLE_RADIUS_MULTIPLIER,
  INNER_ZONE_MULTIPLIER,
  OUTER_ZONE_INDICES,
  zoneBuildingLocalOffset,
  zoneUnitSlotLocalOffsets,
} from './zones/geometry';
import { buildingImage } from './assets/buildings';
import { scaledIcon, transformScale } from './assets/image_cache';
import { drawForestTrees, isForestResource, mapResourceIcon } from './assets/resources';
import { drawRisqZone, getSpaceTerrainImage, getZoneFill, isEmptyPlot } from './zones/draw';
import type { RisqEntityPass } from './zones/draw';
import { getZoneUnitSlots } from './zones/slots';
import { drawZoneCorpses } from './zones/corpses';
import { isCompletedWall, isWalledZone } from './walls/wall_state';

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
  zone_borders_only?: boolean;
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
    const region_hover =
      config.view_mode === RisqViewMode.RESOURCE && !!region && region === game.hover.hoveredRegion();
    const region_click =
      config.view_mode === RisqViewMode.RESOURCE &&
      !!region &&
      (game.selection.isRegionSelected(region) || (region_hover && !!game.hover.space()?.clicked));
    if (space.hovered || region_hover || region_click) {
      fillHexOverlay(
        ctx,
        space.center,
        config.hex_r,
        space.clicked || region_click ? 'rgba(210, 210, 210, 0.4)' : 'rgba(190, 190, 190, 0.2)'
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
  if (config.zone_borders_only || config.entity_pass === 'terrain' || config.entity_pass === 'selected') {
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

/** The territory border multiplier: the display setting unless the caller pins its own */
export function borderScale(config: { border_scale?: number }): number {
  return config.border_scale ?? getSettings().risq_display.territory_border_width;
}

export function spaceBorderWidth(
  config: Pick<DrawRisqSpaceConfig, 'hex_r' | 'draw_detail'> & { view_mode?: RisqViewMode; border_scale?: number }
): number {
  const widths = isOverviewViewMode(config.view_mode) ? overview_space_line_width : space_line_width;
  return widths[config.draw_detail] * (config.hex_r / SPACE_BORDER_REFERENCE_RADIUS) * borderScale(config);
}

function drawZoneBuildingSprite(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  hex_r: number,
  rot: number,
  space_rot: number
): void {
  const b = zone.building;
  if (!b) {
    return;
  }
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const scale = transformScale(ctx);
  const local = zoneBuildingLocalOffset(zone.coordinate, hex_r);
  ctx.save();
  if (rot !== 0) {
    ctx.rotate(rot);
  }
  ctx.translate(local.x, local.y);
  ctx.rotate(-(rot + space_rot));
  const icon = scaledIcon(
    game,
    buildingImage(b.building_id, b.under_construction, false, b.combat_stats),
    game.getGame()?.players[b.player_id]?.color,
    2 * building_r * scale
  );
  ctx.drawImage(icon, -building_r, -building_r, 2 * building_r, 2 * building_r);
  ctx.restore();
}

function drawZoneResourceSprite(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  hex_r: number,
  rot: number,
  space_rot: number
): void {
  const resource = zone.resource;
  if (!resource || isForestResource(resource)) {
    return;
  }
  const radius = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const scale = transformScale(ctx);
  const local = zoneBuildingLocalOffset(zone.coordinate, hex_r);
  ctx.save();
  ctx.rotate(rot);
  ctx.translate(local.x, local.y);
  ctx.rotate(-(rot + space_rot));
  const icon = game.getImageCache().getScaledSprite(mapResourceIcon(game, resource), 2 * radius * scale);
  ctx.drawImage(icon, -radius, -radius, 2 * radius, 2 * radius);
  ctx.restore();
}

function drawSpaceBuildings(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig
): void {
  if (config.view_mode === RisqViewMode.OWNERSHIP || !space.zones) {
    return;
  }
  const a = Math.PI / 3;
  ctx.save();
  ctx.translate(space.center.x, space.center.y);
  for (let i = -1; i < 6; i++) {
    const zone = i === -1 ? space.zones[1][1] : space.zones[OUTER_ZONE_INDICES[i].x][OUTER_ZONE_INDICES[i].y];
    const b = zone?.building;
    const rot = i === -1 ? 0 : a * (1 + i);
    if (isCompletedWall(b)) {
      continue;
    }
    if (b && buildingVisibleInViewMode(b, config.view_mode) && !game.selection.isBuildingSelected(b.internal_id)) {
      drawZoneBuildingSprite(ctx, game, zone, config.hex_r, rot, config.rotation);
    } else if (
      zone?.resource &&
      (config.view_mode === RisqViewMode.ALL || config.view_mode === RisqViewMode.RESOURCE)
    ) {
      drawZoneResourceSprite(ctx, game, zone, config.hex_r, rot, config.rotation);
    }
  }
  ctx.restore();
}

let wedge_cache_r = -1;
let wedge_cache: Point2D[][] = [];

function outerZoneWedges(r: number): Point2D[][] {
  if (r !== wedge_cache_r) {
    const a = Math.PI / 3;
    const inner_r = INNER_ZONE_MULTIPLIER * r;
    wedge_cache = Array.from({ length: 6 }, (_, i): Point2D[] => [
      { x: inner_r * Math.cos(a * i + Math.PI / 6), y: inner_r * Math.sin(a * i + Math.PI / 6) },
      { x: inner_r * Math.cos(a * i + Math.PI / 2), y: inner_r * Math.sin(a * i + Math.PI / 2) },
      { x: r * Math.cos(a * i + Math.PI / 2), y: r * Math.sin(a * i + Math.PI / 2) },
      { x: r * Math.cos(a * i + Math.PI / 6), y: r * Math.sin(a * i + Math.PI / 6) },
    ]);
    wedge_cache_r = r;
  }
  return wedge_cache;
}

function drawSpaceContent(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  space: RisqSpace,
  config: DrawRisqSpaceConfig,
  black_text: boolean,
  owner_color: ColorRGB | undefined
): void {
  if (config.zone_borders_only && config.draw_detail !== DrawRisqSpaceDetail.ZONE_DETAILS) {
    return;
  }
  if (config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP) {
    drawSpaceOverviewDots(ctx, game, space, config);
    return;
  } else if (config.draw_detail === DrawRisqSpaceDetail.SPACE_DETAILS && config.entity_pass !== 'selected') {
    if (space.visibility < RisqVisibilityLevel.FOG || config.entity_pass === 'terrain') {
      return;
    }
    drawSpaceBuildings(ctx, game, space, config);
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
      game.viewport.edgeMaxSlotsFor(zone),
      config.view_mode
    );
    const center_count = space.visibility === RisqVisibilityLevel.POOR ? 1 : center_slots.length;
    if (!config.entity_pass || (config.entity_pass === 'terrain' && config.zone_borders_only !== false)) {
      drawHexagon(ctx, { x: 0, y: 0 }, inner_r);
    }
    if (!config.zone_borders_only) {
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
        zoneUnitSlotLocalOffsets(zone.coordinate, r, center_count, game.viewport.unitRadius(), isWalledZone(zone)),
        active_player_id,
        config.entity_pass
      );
    }
    const a = Math.PI / 3;
    const wedges = outerZoneWedges(r);
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
      if (!config.entity_pass || (config.entity_pass === 'terrain' && config.zone_borders_only !== false)) {
        ctx.beginPath();
        for (const vertex of wedges[i]) {
          ctx.lineTo(vertex.x, vertex.y);
        }
        ctx.closePath();
        ctx.stroke();
        ctx.fill();
      }
      const rotation = a * (1 + i);
      ctx.rotate(rotation);
      const edge_slots = getZoneUnitSlots(
        zone,
        active_player_id,
        game.viewport.centerMaxSlots(),
        game.viewport.edgeMaxSlotsFor(zone),
        config.view_mode
      );
      const edge_count = space.visibility === RisqVisibilityLevel.POOR ? 1 : edge_slots.length;
      if (!config.zone_borders_only) {
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
          zoneUnitSlotLocalOffsets(zone.coordinate, r, edge_count, game.viewport.unitRadius(), isWalledZone(zone)),
          active_player_id,
          config.entity_pass
        );
      }
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
