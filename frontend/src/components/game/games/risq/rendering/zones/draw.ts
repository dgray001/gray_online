import type { DwgRisq } from '../../risq';
import type { RisqOrderPlanning } from '../../application/orders/planning';
import type { RisqZone, UnitByTypeData } from '../../model/types';
import { terrainImage, RisqViewMode } from '../terrain';
import {
  findOuterZoneIndex,
  clipToZone,
  buildingLocalOffset,
  CENTER_ZONE_APOTHEM_MULTIPLIER,
  BUILDING_CIRCLE_RADIUS_MULTIPLIER,
  edgeSlotAtAngle,
  UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER,
} from './geometry';
import type { Point2D } from '../../../../util/objects2d';
import { unitImage, comboUnitIconKey, COMBO_UNIT_ICON_SIZE, drawComboUnitIcon } from '../assets/unit';
import { ColorRGB } from '../../../../../../scripts/color_rgb';
import { RisqUnitType, RisqOrderType, RisqVisibilityLevel } from '../../model/types';
import { isForestResource, resourceImage, resourceIcon } from '../assets/resources';
import { rotatePoint } from '../../../../util/objects2d';
import { buildZoneUnitSlots } from './slots';
import { buildingImage } from '../assets/buildings';
import { drawEllipse } from '../../../../util/canvas_util';
/** Returns the space's terrain image, or (cached, at the base terrain's own native resolution) a composite with any zone terrain overrides painted on top */
export function getSpaceTerrainImage(
  game: DwgRisq,
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

export const UNIT_CLUSTER_ICON_SIZE = 64;

/** Cache key for a multi-unit-type cluster icon, sensitive to the types, their counts, and the shown total */
export function unitClusterIconKey(units_by_type: UnitByTypeData[], total: number): string {
  return 'unit_cluster:' + units_by_type.map((t) => `${t.unit_id}x${t.units.size}`).join(',') + `:${total}`;
}

/** Draws one player's unit types centered at the origin, arranging 1/2/3/4/many types each with its count */
export function drawUnitTypeCluster(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  units_by_type: UnitByTypeData[],
  r: Point2D,
  total: number,
  primary_color: string,
  secondary_color: string
) {
  const draw_count = (s: string, ts: number, x: number, y: number, w: number, fill_primary = true) => {
    const fs = ctx.fillStyle;
    ctx.fillStyle = fill_primary ? primary_color : secondary_color;
    ctx.font = `bold ${ts}px serif`;
    ctx.fillText(s, x, y, w);
    ctx.fillStyle = fs;
  };
  const icon = (t: UnitByTypeData) => {
    const color = game.getGame()?.players[t.player_id]?.color;
    return color ? game.getPlayerColoredIcon(unitImage(t.unit_id), color) : game.getIcon(unitImage(t.unit_id));
  };
  if (units_by_type.length === 1) {
    const t = units_by_type[0];
    ctx.drawImage(icon(t), -r.x, -r.y, 2 * r.x, 2 * r.y);
    draw_count(t.units.size.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  } else if (units_by_type.length === 2) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 1.5 * r.x, 1.5 * r.y);
      draw_count(t.units.size.toString(), r.y, (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 2 * r.x);
    }
  } else if (units_by_type.length === 3) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.9 * r.y, 1.5 * r.x);
    }
    const t = units_by_type[2];
    ctx.drawImage(icon(t), -0.5 * r.x, -0.1 * r.y, r.x, r.y);
    draw_count(t.units.size.toString(), 0.75 * r.y, -0.5 * r.x, -0.1 * r.y, 1.5 * r.x);
  } else if (units_by_type.length === 4) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.9 * r.y, 1.5 * r.x);
    }
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[2 + j];
      ctx.drawImage(icon(t), (0.8 * j - 0.9) * r.x, -0.1 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.1 * r.y, 1.5 * r.x);
    }
  } else {
    for (let j = 0; j < 2; j++) {
      ctx.drawImage(icon(units_by_type[j]), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
    }
    ctx.drawImage(icon(units_by_type[2]), -0.9 * r.x, -0.1 * r.y, r.x, r.y);
    draw_count('...', 0.8 * r.y, 0.1 * r.x, -0.1 * r.y, r.x, false);
    draw_count(total.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  }
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

export function unitVisibleInViewMode(unit_type: RisqUnitType, view_mode: RisqViewMode): boolean {
  if (view_mode === RisqViewMode.OWNERSHIP) {
    return false;
  }
  if (view_mode === RisqViewMode.MILITARY) {
    return unit_type !== RisqUnitType.ECONOMIC;
  }
  if (view_mode === RisqViewMode.RESOURCE) {
    return unit_type === RisqUnitType.ECONOMIC;
  }
  return true;
}

const CENTER_FOREST_TREES = 8;
const EDGE_FOREST_TREES = 6;
const FOREST_TREE_RADIUS_MULTIPLIER = 0.11;
const FOREST_TREE_RING_FRACTION = 0.4;
const CENTER_INNER_FOREST_ANGLES = [0, Math.PI / 2, Math.PI, (3 * Math.PI) / 2];
const EDGE_INNER_FOREST_ANGLES = CENTER_INNER_FOREST_ANGLES.slice(1);

/** Local-frame tree spots: a ring (center) or inward arc (edge), plus an inner square/right triangle at half that distance */
function forestTreeLocalOffsets(is_center: boolean, hex_r: number): Point2D[] {
  const building = buildingLocalOffset(is_center, hex_r);
  const gap = CENTER_ZONE_APOTHEM_MULTIPLIER - BUILDING_CIRCLE_RADIUS_MULTIPLIER;
  const center_ring_r = (BUILDING_CIRCLE_RADIUS_MULTIPLIER + FOREST_TREE_RING_FRACTION * gap) * hex_r;
  const ring_at = (angle: number): Point2D =>
    is_center
      ? { x: center_ring_r * Math.cos(angle), y: center_ring_r * Math.sin(angle) }
      : edgeSlotAtAngle(angle, hex_r, FOREST_TREE_RING_FRACTION);
  const ring_angles = is_center
    ? Array.from({ length: CENTER_FOREST_TREES }, (_, i) => (2 * Math.PI * i) / CENTER_FOREST_TREES)
    : Array.from({ length: EDGE_FOREST_TREES }, (_, i) => Math.PI / 2 + (i * Math.PI) / (EDGE_FOREST_TREES - 1));
  const inner = (is_center ? CENTER_INNER_FOREST_ANGLES : EDGE_INNER_FOREST_ANGLES).map((angle) => {
    const p = ring_at(angle);
    return { x: 0.5 * (building.x + p.x), y: 0.5 * (building.y + p.y) };
  });
  return [...inner, ...ring_angles.map(ring_at)];
}

/** Draws a forest zone's trees upright around its space's center (the current origin), clipped to the zone */
export function drawForestTrees(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  zone: RisqZone,
  view_mode: RisqViewMode,
  hex_r: number,
  map_rotation: number
) {
  const resource = zone.resource;
  const hidden_view = view_mode === RisqViewMode.MILITARY || view_mode === RisqViewMode.OWNERSHIP;
  if (!resource || !isForestResource(resource) || hidden_view) {
    return;
  }
  const direction = findOuterZoneIndex(zone.coordinate);
  const zone_rotation = direction === -1 ? 0 : (Math.PI / 3) * (direction + 1);
  const icon = game.getIcon(resourceImage(resource));
  const icon_r = FOREST_TREE_RADIUS_MULTIPLIER * hex_r;
  ctx.save();
  clipToZone(ctx, { x: 0, y: 0 }, hex_r, zone.coordinate);
  for (const offset of forestTreeLocalOffsets(direction === -1, hex_r)) {
    const p = rotatePoint(offset, zone_rotation);
    ctx.translate(p.x, p.y);
    ctx.rotate(-map_rotation);
    ctx.drawImage(icon, -icon_r, -icon_r, 2 * icon_r, 2 * icon_r);
    ctx.rotate(map_rotation);
    ctx.translate(-p.x, -p.y);
  }
  ctx.restore();
}

const BUILDING_GHOST_ALPHA = 0.5;

export function isEmptyPlot(planning: RisqOrderPlanning, zone: RisqZone): boolean {
  return !zone.resource && !zone.building && !planning.hasPlannedFoundation(zone);
}

/** Building to preview in a hovered zone: its foundation's building, else the armed build if the zone is empty */
function ghostBuildingId(game: DwgRisq, zone: RisqZone): number | undefined {
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

export function drawRisqZone(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  zone: RisqZone,
  visibility: number,
  view_mode: RisqViewMode,
  black_text: boolean,
  hex_r: number,
  rotation: number,
  building_pos: Point2D,
  unit_slot_positions: Point2D[],
  active_player_id: number
) {
  const primary_color = black_text ? 'rgb(0, 0, 0)' : 'rgb(255, 255, 255)';
  const secondary_color = black_text ? 'rgba(40, 40, 40, 0.4)' : 'rgba(210, 210, 210, 0.4)';
  const tertiary_color = black_text ? 'rgb(60, 60, 60, 0.2)' : 'rgba(190, 190, 190, 0.2)';

  function drawText(
    ctx: CanvasRenderingContext2D,
    s: string,
    ts: number,
    x: number,
    y: number,
    w: number,
    fill_primary = true
  ) {
    const fs = ctx.fillStyle;
    ctx.fillStyle = fill_primary ? primary_color : secondary_color;
    ctx.font = `bold ${ts}px serif`;
    ctx.fillText(s, x, y, w);
    ctx.fillStyle = fs;
  }

  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  const building_r = BUILDING_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const unit_r = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  const filled_slots = buildZoneUnitSlots(zone, active_player_id, unit_slot_positions.length);
  zone.unit_slots = filled_slots;

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
    }
  }
  for (const [i, part] of zone.hovered_data.entries()) {
    const selected =
      i === 0
        ? (!!zone.resource && game.selection.isResourceSelected(zone.resource.internal_id)) ||
          (!!zone.building && game.selection.isBuildingSelected(zone.building.internal_id))
        : filled_slots[i - 1].some((t) => [...t.units].some((uid) => game.selection.isUnitSelected(uid)));
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
        ctx.drawImage(resourceIcon(game, zone.resource), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
      } else {
        let building_image: string;
        let building_color: ColorRGB | undefined;

        const local_foundation = game.planning.getLocalFoundation(zone.coordinate_key);
        const server_foundation = game.getPlayer()?.planned_foundations?.get(zone.coordinate_key);

        if (zone.building) {
          building_image = buildingImage(zone.building.building_id, zone.building.under_construction);
          building_color = game.getGame()?.players[zone.building.player_id]?.color;
        } else if (local_foundation || server_foundation) {
          building_image = 'risq/buildings/construction';
          building_color = game.getPlayer()?.color;
        } else {
          building_image = buildingImage(undefined);
          building_color = undefined;
        }

        const building_icon = building_color
          ? game.getPlayerColoredIcon(building_image, building_color)
          : game.getIcon(building_image);
        if (zone.building || local_foundation || server_foundation) {
          ctx.drawImage(building_icon, -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        } else {
          ctx.strokeStyle = secondary_color;
        }
        const ghost_id = ghostBuildingId(game, zone);
        if (ghost_id !== undefined) {
          const ghost_image = buildingImage(ghost_id, false);
          const ghost_color = building_color ?? game.getPlayer()?.color;
          ctx.globalAlpha = BUILDING_GHOST_ALPHA;
          ctx.drawImage(
            ghost_color
              ? game.getPlayerColoredIcon(ghost_image, ghost_color)
              : game.getIcon(buildingImage(ghost_id, false, true)),
            -part.r.x,
            -part.r.y,
            2 * part.r.x,
            2 * part.r.y
          );
          ctx.globalAlpha = 1;
        }
        if (!!zone.building && zone.building.has_garrisoned_units) {
          const flag_icon = building_color
            ? game.getPlayerColoredIcon('risq/icons/garrison_flag', building_color)
            : game.getIcon('risq/icons/garrison_flag');
          const flag_size = 0.9 * part.r.x;
          ctx.drawImage(flag_icon, 0.5 * part.r.x, -1.4 * part.r.y, flag_size, flag_size);
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
        drawText(ctx, zone.unit_count.toString(), 1.4 * part.r.y, -part.r.x, -0.7 * part.r.y, 2 * part.r.x);
      } else {
        ctx.strokeStyle = secondary_color;
      }
    } else {
      const slot = filled_slots[i - 1].filter((t) => unitVisibleInViewMode(t.unit_type, view_mode));
      if (slot.length === 0) {
        ctx.strokeStyle = secondary_color;
      } else {
        const slot_total = slot.reduce((sum, t) => sum + t.units.size, 0);
        drawUnitTypeCluster(ctx, game, slot, part.r, slot_total, primary_color, secondary_color);
      }
    }
    ctx.rotate(rotation);
    ctx.translate(-part.c.x, -part.c.y);
    if (selected) {
      const prev_line_width = ctx.lineWidth;
      ctx.strokeStyle = 'white';
      ctx.lineWidth = 0.4;
      drawEllipse(ctx, part.c, part.r);
      ctx.lineWidth = prev_line_width;
    } else {
      drawEllipse(ctx, part.c, part.r);
    }
  }
}
