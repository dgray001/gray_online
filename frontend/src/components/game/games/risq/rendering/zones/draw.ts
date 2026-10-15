import type { RisqDisplaySettings } from '../../../../../../scripts/settings_store';
import { getSettings } from '../../../../../../scripts/settings_store';
import type { RisqDrawHost } from '../board/board_renderer';
import type { RisqBuilding, RisqResource, RisqZone, UnitByTypeData } from '../../model/types';
import { RESOURCE_DOT_COLORS } from '../overview_dots';
import { terrainImage, RisqViewMode, buildingVisibleInViewMode, unitVisibleInViewMode } from '../terrain';
import { findOuterZoneIndex, clipToZone, BUILDING_CIRCLE_RADIUS_MULTIPLIER } from './geometry';
import type { Point2D } from '../../../../util/objects2d';
import {
  comboUnitIconKey,
  COMBO_UNIT_ICON_SIZE,
  drawComboUnitIcon,
  drawUnitCount,
  drawUnitTypeCluster,
} from '../assets/unit';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { RisqOrderType, RisqResourceType, RisqVisibilityLevel } from '../../model/types';
import { resourceIcon, resourceType } from '../assets/resources';
import { drawControlGroupLabel, getZoneUnitSlots } from './slots';
import { buildingImage, rubbleImage } from '../assets/buildings';
import { scaledIcon, transformScale } from '../assets/image_cache';
import { drawEllipse } from '../../../../util/canvas_util';
/** Returns the space's terrain image, or (cached, at the base terrain's own native resolution) a composite with any zone terrain overrides painted on top */
export function getSpaceTerrainImage(
  game: RisqDrawHost,
  base_terrain_id: number,
  space_zones: RisqZone[]
): CanvasImageSource {
  const base_icon = game.getIcon(terrainImage(base_terrain_id));
  const overrides = space_zones.filter((z) => !!z.terrain_override);
  if (overrides.length === 0) {
    return base_icon;
  }
  const key = `space_terrain:${base_terrain_id}:${overrides
    .map((z) => `${findOuterZoneIndex(z.coordinate)}=${z.terrain_override}`)
    .sort()
    .join(',')}`;
  const w = base_icon.naturalWidth;
  const h = base_icon.naturalHeight;
  const c = { x: w / 2, y: h / 2 };
  const r = h / 2;
  const override_icons = overrides.map((z) => game.getIcon(terrainImage(z.terrain_override)));
  const composite = game.getImageCache().getRectImage(key, w, h, [base_icon, ...override_icons], (ctx) => {
    ctx.drawImage(base_icon, 0, 0, w, h);
    overrides.forEach((zone, i) => {
      ctx.save();
      clipToZone(ctx, c, r, zone.coordinate);
      ctx.drawImage(override_icons[i], 0, 0, w, h);
      ctx.restore();
    });
  });
  return composite ?? base_icon;
}

export function getZoneFill(
  zone: RisqZone,
  view_mode: RisqViewMode = RisqViewMode.ALL,
  owner_color: ColorRGB | undefined = undefined,
  check_hover = true,
  alpha_multiplier = 1,
  plot_empty = false
): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (view_mode === RisqViewMode.OWNERSHIP) {
    if (owner_color) {
      color.setColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), 0.85);
    } else {
      color.setColor(90, 90, 90, 0.85);
    }
  } else if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
    color.addColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), alpha_multiplier * 0.06);
  }
  const part_hovered = zone.hovered_data.some((p, i) => p.hovered && !(plot_empty && i === 0));
  if (check_hover && zone.hovered && !part_hovered) {
    if (zone.clicked) {
      color.addColor(210, 210, 210, alpha_multiplier * 0.06);
    } else {
      color.addColor(190, 190, 190, alpha_multiplier * 0.03);
    }
  }
  return color;
}

export function visibleZoneUnitSlot(zone: RisqZone, index: number, view_mode: RisqViewMode): UnitByTypeData[] {
  return (zone.unit_slots?.[index] ?? []).filter((group) => unitVisibleInViewMode(group.unit_type, view_mode));
}

const BUILDING_GHOST_ALPHA = 0.5;

export function isEmptyPlot(planning: RisqDrawHost['planning'], zone: RisqZone): boolean {
  return !zone.resource && !zone.building && !planning.hasPlannedFoundation(zone);
}

/** Building to preview in a hovered zone: its foundation's building, else the armed build if the zone is empty */
function ghostBuildingId(game: RisqDrawHost, zone: RisqZone): number | undefined {
  if (!zone.hovered || zone.resource) {
    return undefined;
  }
  const foundation =
    game.planning.getLocalFoundation(zone.coordinate_key)?.building_id ??
    game.getPlayer()?.planned_foundations?.get(zone.coordinate_key)?.building_id ??
    (zone.building?.under_construction ? zone.building.building_id : undefined);
  if (foundation !== undefined) {
    return foundation;
  }
  const armed =
    game.armed.getArmedOrder() === RisqOrderType.OrderType_UnitBuild ? game.armed.getArmedBuilding() : undefined;
  return zone.building ? undefined : armed?.id;
}

function drawIdleMarker(ctx: CanvasRenderingContext2D, r: Point2D, size_px: number, pulse: boolean): void {
  if (size_px <= 0) {
    return;
  }
  const scale = Math.hypot(ctx.getTransform().a, ctx.getTransform().b);
  const pulse_mult = pulse ? 1 + 0.25 * Math.sin(Date.now() / 150) : 1;
  const s = (size_px * pulse_mult) / scale;
  const x = Math.SQRT1_2 * r.x;
  const y = -Math.SQRT1_2 * r.y;
  ctx.save();
  ctx.fillStyle = 'rgb(240, 200, 40)';
  ctx.strokeStyle = 'rgb(40, 30, 0)';
  ctx.lineWidth = 1.5 / scale;
  ctx.beginPath();
  ctx.arc(x, y, s / 2, 0, 2 * Math.PI);
  ctx.fill();
  ctx.stroke();
  if (size_px >= 10) {
    ctx.fillStyle = 'black';
    ctx.font = `bold ${Math.round(s * 0.7)}px sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText('Z', x, y);
  }
  ctx.restore();
}

function renderOverheadBars(
  ctx: CanvasRenderingContext2D,
  r: Point2D,
  hp: { current: number; max: number } | undefined,
  sec: { current: number; max: number; color?: string } | undefined,
  show_hp: boolean,
  show_sec: boolean,
  size: number
): void {
  ctx.save();
  const scale = Math.hypot(ctx.getTransform().a, ctx.getTransform().b);
  const h = size / scale;
  const inset = size <= 2 ? 0 : Math.min(1 / scale, h / 4);
  const w = 1.4 * r.x;
  let y = -r.y - h - 2 / scale;
  if (show_sec && sec) {
    ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
    ctx.fillRect(-w / 2, y, w, h);
    ctx.fillStyle = sec.color ?? 'rgb(140, 220, 255)';
    const ratio = Math.max(0, Math.min(1, sec.current / sec.max));
    ctx.fillRect(-w / 2 + inset, y + inset, (w - 2 * inset) * ratio, h - 2 * inset);
    y -= h + 1.5 / scale;
  }
  if (show_hp && hp) {
    ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
    ctx.fillRect(-w / 2, y, w, h);
    const ratio = Math.max(0, Math.min(1, hp.current / hp.max));
    ctx.fillStyle = ratio > 0.5 ? 'rgb(60, 200, 60)' : ratio > 0.25 ? 'rgb(230, 190, 40)' : 'rgb(220, 50, 40)';
    ctx.fillRect(-w / 2 + inset, y + inset, (w - 2 * inset) * ratio, h - 2 * inset);
  }
  ctx.restore();
}

function drawOverheadBars(
  ctx: CanvasRenderingContext2D,
  r: Point2D,
  hp: { current: number; max: number } | undefined,
  sec: { current: number; max: number; color?: string } | undefined,
  selected: boolean,
  display: RisqDisplaySettings = getSettings().risq_display
): void {
  const show_hp =
    !!hp &&
    hp.max > 0 &&
    display.healthbars !== 'never' &&
    (display.healthbars === 'always' ||
      (display.healthbars === 'damaged_only' && hp.current < hp.max) ||
      (display.healthbars === 'selected_only' && selected));
  const show_sec =
    !!sec &&
    sec.max > 0 &&
    display.secondary_bars !== 'never' &&
    (display.secondary_bars === 'always' ||
      (display.secondary_bars === 'active_only' && sec.current < sec.max) ||
      (display.secondary_bars === 'selected_only' && selected));
  if (!show_hp && !show_sec) {
    return;
  }
  renderOverheadBars(ctx, r, hp, sec, show_hp, show_sec, display.healthbar_size);
}

function resourceDepletion(resource: RisqResource): { current: number; max: number; color: string } {
  return {
    current: resource.resources_left,
    max: resource.max_resources,
    color: RESOURCE_DOT_COLORS[resourceType(resource)],
  };
}

function buildingSecondaryBar(
  b: RisqBuilding,
  display: RisqDisplaySettings
): { current: number; max: number; color: string } | undefined {
  const progress_color = 'rgb(140, 220, 255)';
  if (b.under_construction) {
    const total = b.construction_stamina_total ?? 0;
    return { current: total - (b.stamina_remaining ?? 0), max: total, color: progress_color };
  }
  if (b.renewing && b.renew_stamina) {
    return { current: b.renew_stamina - (b.renew_stamina_remaining ?? 0), max: b.renew_stamina, color: progress_color };
  }
  if (b.resource_capacity) {
    const color = RESOURCE_DOT_COLORS[b.resource_category ?? RisqResourceType.ERROR];
    return { current: b.resources_left ?? 0, max: b.resource_capacity, color };
  }
  const stamina_hidden = display.secondary_bars === 'active_only';
  return !stamina_hidden && (b.current_stamina ?? 0) < (b.max_stamina ?? 0)
    ? { current: b.current_stamina, max: b.max_stamina, color: progress_color }
    : undefined;
}

function drawSlotOverheadBars(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  slot: UnitByTypeData[],
  r: Point2D,
  selected: boolean,
  display?: RisqDisplaySettings
): void {
  let hp = 0;
  let max_hp = 0;
  let stamina = 0;
  let max_stamina = 0;
  for (const t of slot) {
    const units = game.getGame()?.players[t.player_id]?.units;
    if (!units) {
      continue;
    }
    for (const uid of t.units) {
      const u = units.get(uid);
      if (u?.combat_stats) {
        hp += u.combat_stats.health;
        max_hp += u.combat_stats.max_health;
        stamina += u.current_stamina ?? 0;
        max_stamina += u.max_stamina ?? 0;
      }
    }
  }
  if (max_hp > 0) {
    const show_stamina = (display ?? getSettings().risq_display).secondary_bars !== 'active_only';
    const sec = show_stamina && max_stamina > 0 ? { current: stamina, max: max_stamina } : undefined;
    drawOverheadBars(ctx, r, { current: hp, max: max_hp }, sec, selected, display);
  }
}

export type RisqEntityPass = 'terrain' | 'unselected' | 'selected';

export function drawRisqZone(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  visibility: number,
  view_mode: RisqViewMode,
  black_text: boolean,
  hex_r: number,
  rotation: number,
  building_pos: Point2D,
  unit_slot_positions: Point2D[],
  active_player_id: number,
  entity_pass?: RisqEntityPass
): void {
  const primary_color = black_text ? 'rgb(0, 0, 0)' : 'rgb(255, 255, 255)';
  const secondary_color = black_text ? 'rgba(40, 40, 40, 0.4)' : 'rgba(210, 210, 210, 0.4)';
  const tertiary_color = black_text ? 'rgb(60, 60, 60, 0.2)' : 'rgba(190, 190, 190, 0.2)';

  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const unit_r = game.viewport.unitRadius();
  const unit_slots = getZoneUnitSlots(
    zone,
    active_player_id,
    game.viewport.centerMaxSlots(),
    game.viewport.edgeMaxSlots(),
    view_mode
  );
  const filled_slots =
    visibility === RisqVisibilityLevel.POOR
      ? zone.unit_count && view_mode !== RisqViewMode.OWNERSHIP
        ? [[]]
        : []
      : unit_slots;

  const target_len = 1 + filled_slots.length;
  if (zone.hovered_data.length !== target_len || zone.reset_hovered_data) {
    zone.hovered_data = [
      { c: building_pos, r: { x: building_r, y: building_r } },
      ...filled_slots.map((_, i) => ({ c: unit_slot_positions[i], r: { x: unit_r, y: unit_r } })),
    ];
    zone.reset_hovered_data = false;
  } else {
    zone.hovered_data[0].c = building_pos;
    for (let i = 0; i < filled_slots.length; i++) {
      zone.hovered_data[i + 1].c = unit_slot_positions[i];
      zone.hovered_data[i + 1].r = { x: unit_r, y: unit_r };
    }
  }
  if (entity_pass === 'terrain') {
    return;
  }
  const display = getSettings().risq_display;
  const show_idle = display.idle_indicator_size > 0 && !game.session.isReplaying();
  const idle_ids = show_idle ? game.planning.idleIds() : undefined;
  const scale = transformScale(ctx);
  for (const [i, part] of zone.hovered_data.entries()) {
    const selected =
      i === 0
        ? (!!zone.resource && game.selection.isResourceSelected(zone.resource.internal_id)) ||
          (!!zone.building && game.selection.isBuildingSelected(zone.building.internal_id))
        : filled_slots[i - 1].some((t) => [...t.units].some((uid) => game.selection.isUnitSelected(uid)));
    if (entity_pass && selected !== (entity_pass === 'selected')) {
      continue;
    }
    ctx.strokeStyle = 'transparent';
    if (part.hovered && !(i === 0 && isEmptyPlot(game.planning, zone))) {
      if (part.clicked) {
        ctx.fillStyle = secondary_color;
      } else {
        ctx.fillStyle = tertiary_color;
      }
    } else {
      ctx.fillStyle = 'transparent';
    }
    ctx.translate(part.c.x, part.c.y);
    ctx.rotate(-rotation);
    if (i === 0) {
      if (!!zone.resource && view_mode !== RisqViewMode.MILITARY && view_mode !== RisqViewMode.OWNERSHIP) {
        const resource_icon = game
          .getImageCache()
          .getScaledSprite(resourceIcon(game, zone.resource), 2 * part.r.x * scale);
        ctx.drawImage(resource_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        drawOverheadBars(ctx, part.r, undefined, resourceDepletion(zone.resource), selected, display);
      } else if (!zone.building || buildingVisibleInViewMode(zone.building, view_mode)) {
        let building_image: string;
        let building_color: ColorRGB | undefined;

        const local_foundation = game.planning.getLocalFoundation(zone.coordinate_key);
        const server_foundation = game.getPlayer()?.planned_foundations?.get(zone.coordinate_key);

        if (zone.building) {
          building_image = buildingImage(
            zone.building.building_id,
            zone.building.under_construction,
            false,
            zone.building.combat_stats
          );
          building_color = game.getGame()?.players[zone.building.player_id]?.color;
        } else if (zone.destroyed_building) {
          building_image = rubbleImage(zone.destroyed_building, zone.destroyed_building_turns ?? 1);
          building_color = undefined;
        } else if (local_foundation || server_foundation) {
          building_image = 'risq/buildings/construction';
          building_color = game.getPlayer()?.color;
        } else {
          building_image = buildingImage(undefined);
          building_color = undefined;
        }

        const building_icon = scaledIcon(game, building_image, building_color, 2 * part.r.x * scale);
        if (zone.building || zone.destroyed_building || local_foundation || server_foundation) {
          ctx.drawImage(building_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        } else {
          ctx.strokeStyle = secondary_color;
        }
        const ghost_id = ghostBuildingId(game, zone);
        if (ghost_id !== undefined) {
          const ghost_image = buildingImage(ghost_id, false);
          const ghost_color = building_color ?? game.getPlayer()?.color;
          ctx.globalAlpha = BUILDING_GHOST_ALPHA;
          const ghost_icon = ghost_color
            ? scaledIcon(game, ghost_image, ghost_color, 2 * part.r.x * scale)
            : scaledIcon(game, buildingImage(ghost_id, false, true), undefined, 2 * part.r.x * scale);
          ctx.drawImage(ghost_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
          ctx.globalAlpha = 1;
        }
        if (!!zone.building && zone.building.has_garrisoned_units) {
          const flag_size = 0.9 * part.r.x;
          const flag_icon = scaledIcon(game, 'risq/icons/garrison_flag', building_color, flag_size * scale);
          ctx.drawImage(flag_icon, 0.5 * part.r.x, -1.4 * part.r.y, flag_size, flag_size);
        }
        if (zone.building && zone.building.player_id === game.getPlayerId()) {
          if (idle_ids?.buildings.has(zone.building.internal_id)) {
            drawIdleMarker(ctx, part.r, display.idle_indicator_size, display.idle_indicator_pulse);
          }
        }
        if (zone.building) {
          const b = zone.building;
          const hp = b.combat_stats ? { current: b.combat_stats.health, max: b.combat_stats.max_health } : undefined;
          drawOverheadBars(ctx, part.r, hp, buildingSecondaryBar(b, display), selected, display);
        }
      }
    } else if (visibility === RisqVisibilityLevel.POOR) {
      if (i === 1 && !!zone.unit_count && view_mode !== RisqViewMode.OWNERSHIP) {
        const villager_img = game.getIcon('icons/villager64');
        const unit_img = game.getIcon('icons/unit64');
        const combo_icon = game
          .getImageCache()
          .getImage(comboUnitIconKey(false), COMBO_UNIT_ICON_SIZE, [villager_img, unit_img], (combo_ctx) =>
            drawComboUnitIcon(combo_ctx, villager_img, unit_img)
          );
        if (combo_icon) {
          ctx.drawImage(combo_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        }
        drawUnitCount(ctx, zone.unit_count.toString(), 1.4 * part.r.y, 0, 0, 2 * part.r.x, primary_color, true);
      } else {
        ctx.strokeStyle = secondary_color;
      }
    } else {
      const slot = visibleZoneUnitSlot(zone, i - 1, view_mode);
      if (slot.length === 0) {
        ctx.strokeStyle = secondary_color;
      } else {
        const slot_total = slot.reduce((sum, t) => sum + t.units.size, 0);
        drawUnitTypeCluster(ctx, game, slot, part.r, slot_total, primary_color, secondary_color, true);
        if (
          idle_ids &&
          slot.some((t) => t.player_id === game.getPlayerId() && [...t.units].some((uid) => idle_ids.units.has(uid)))
        ) {
          drawIdleMarker(ctx, part.r, display.idle_indicator_size, display.idle_indicator_pulse);
        }
        drawSlotOverheadBars(ctx, game, slot, part.r, selected, display);
      }
    }
    ctx.rotate(rotation);
    ctx.translate(-part.c.x, -part.c.y);
    if (selected) {
      const prev_line_width = ctx.lineWidth;
      ctx.strokeStyle = 'white';
      ctx.lineWidth = display.selection_outline_thickness / Math.hypot(ctx.getTransform().a, ctx.getTransform().b);
      drawEllipse(ctx, part.c, part.r);
      ctx.lineWidth = prev_line_width;
      ctx.save();
      ctx.translate(part.c.x, part.c.y);
      ctx.rotate(-rotation);
      drawControlGroupLabel(ctx, game, zone, i === 0 ? undefined : visibleZoneUnitSlot(zone, i - 1, view_mode), part.r);
      ctx.restore();
    } else {
      drawEllipse(ctx, part.c, part.r);
    }
  }
}
