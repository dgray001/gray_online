import { ColorRGB } from '../../../../scripts/color_rgb';
import { atangent } from '../../../../scripts/math';
import { drawEllipse } from '../../util/canvas_util';
import type { Point2D } from '../../util/objects2d';
import { equalsPoint2D, pointInHexagon, rotatePoint, subtractPoint2D } from '../../util/objects2d';
import type { DwgRisq } from './risq';
import { buildingImage } from './risq_buildings';
import type { RisqSpace, RisqUnit, RisqZone, UnitByTypeData } from './risq_data';
import { RisqVisibilityLevel } from './risq_data';
import { resourceImage } from './risq_resources';
import { COMBO_UNIT_ICON_SIZE, comboUnitIconKey, drawComboUnitIcon, unitImage } from './risq_unit';
import { RisqViewMode } from './risq_view_mode';

/** Multiplier for inner zone relative to whole radius */
export const INNER_ZONE_MULTIPLIER = 0.4;

const OUTER_ZONE_COORDINATES: Point2D[] = [
  { x: 2, y: 1 },
  { x: 2, y: 0 },
  { x: 1, y: 0 },
  { x: 0, y: 0 },
  { x: 0, y: 1 },
  { x: 1, y: 2 },
];

export function zoneCenterOffset(zone_coordinate: Point2D, hex_r: number): Point2D {
  const i = OUTER_ZONE_COORDINATES.findIndex((dv) => equalsPoint2D(dv, zone_coordinate));
  if (i === -1) {
    return { x: 0, y: 0 };
  }
  const angle = (Math.PI / 3) * (i + 1);
  const r = 0.75 * hex_r;
  return { x: r * Math.cos(angle), y: r * Math.sin(angle) };
}

/** Organizes units by unit id for easier processing */
export function organizeZoneUnits(units: Map<number, RisqUnit>): Map<number, Map<number, UnitByTypeData>> {
  const units_by_type = new Map<number, Map<number, UnitByTypeData>>();
  for (const unit of units.values()) {
    if (!units_by_type.has(unit.player_id)) {
      units_by_type.set(unit.player_id, new Map<number, UnitByTypeData>());
    }
    const units_by_type_data = units_by_type.get(unit.player_id)?.get(unit.unit_id);
    if (units_by_type_data) {
      units_by_type_data.units.add(unit.internal_id);
    } else {
      units_by_type.get(unit.player_id)!.set(unit.unit_id, {
        unit_id: unit.unit_id,
        player_id: unit.player_id,
        units: new Set<number>([unit.internal_id]),
      });
    }
  }
  return units_by_type;
}

/** Groups the input units by player, keeping only economic (unit_id < 11) or military types */
export function unitsByPlayerFiltered(units: Map<number, RisqUnit>, economic: boolean): Map<number, UnitByTypeData[]> {
  const units_by_type = organizeZoneUnits(units);
  const result = new Map<number, UnitByTypeData[]>();
  for (const [player_id, player_units] of units_by_type.entries()) {
    const filtered = [...player_units.values()].filter((u) => u.unit_id < 11 === economic);
    if (filtered.length > 0) {
      result.set(player_id, filtered);
    }
  }
  return result;
}

/** Groups an arbitrary, possibly multi-space list of a single player's unit ids by unit type */
export function groupUnitsByType(units: Map<number, RisqUnit>, ids: number[]): UnitByTypeData[] {
  const by_type = new Map<number, UnitByTypeData>();
  for (const id of ids) {
    const unit = units.get(id);
    if (!unit) {
      continue;
    }
    const existing = by_type.get(unit.unit_id);
    if (existing) {
      existing.units.add(id);
    } else {
      by_type.set(unit.unit_id, { player_id: unit.player_id, unit_id: unit.unit_id, units: new Set([id]) });
    }
  }
  return [...by_type.values()];
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
  const icon = (t: UnitByTypeData) => game.getIcon(unitImage(t.unit_id));
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
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 1) * r.x, -0.9 * r.y, 1.5 * r.x);
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

/** Gets zone fill for the input zone */
export function getZoneFill(
  zone: RisqZone,
  view_mode: RisqViewMode = RisqViewMode.ALL,
  owner_color: ColorRGB | undefined = undefined,
  check_hover = true,
  alpha_multiplier = 1
): ColorRGB {
  const color = new ColorRGB(0, 0, 0, 0);
  if (view_mode === RisqViewMode.OWNERSHIP) {
    if (owner_color) {
      color.setColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), 0.85);
    } else {
      color.setColor(90, 90, 90, 0.85);
    }
  } else {
    color.setColor(10, 120, 10, 0.8);
    if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
      color.addColor(owner_color.getR(), owner_color.getG(), owner_color.getB(), alpha_multiplier * 0.06);
    }
  }
  if (check_hover && zone.hovered && !zone.hovered_data.some((p) => p.hovered)) {
    if (zone.clicked) {
      color.addColor(210, 210, 210, alpha_multiplier * 0.06);
    } else {
      color.addColor(190, 190, 190, alpha_multiplier * 0.03);
    }
  }
  return color;
}

/** Draws the input risq zone */
export function drawRisqZone(
  ctx: CanvasRenderingContext2D,
  game: DwgRisq,
  zone: RisqZone,
  visibility: number,
  view_mode: RisqViewMode,
  black_text: boolean,
  r: number,
  rotation: number,
  p1: Point2D,
  p2: Point2D,
  p3: Point2D
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
  const rp = 0.5 * r;
  if (zone.hovered_data.length !== 3 || zone.reset_hovered_data) {
    const r_part = { x: 0.5 * rp, y: 0.5 * rp };
    zone.hovered_data = [
      { c: p1, r: r_part },
      { c: p2, r: r_part },
      { c: p3, r: r_part },
    ];
    zone.reset_hovered_data = false;
  } else {
    zone.hovered_data[0].c = p1;
    zone.hovered_data[1].c = p2;
    zone.hovered_data[2].c = p3;
  }
  for (const [i, part] of zone.hovered_data.entries()) {
    ctx.strokeStyle = 'transparent';
    if (part.hovered) {
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
    switch (i) {
      case 0: // resources / building
        if (!!zone.resource && view_mode !== RisqViewMode.MILITARY && view_mode !== RisqViewMode.OWNERSHIP) {
          ctx.drawImage(game.getIcon(resourceImage(zone.resource)), -part.r.x, -part.r.y, 2 * part.r.x, 2 * part.r.y);
        } else {
          ctx.drawImage(
            game.getIcon(buildingImage(zone.building?.building_id, zone.building?.under_construction)),
            -part.r.x,
            -part.r.y,
            2 * part.r.x,
            2 * part.r.y
          );
        }
        break;
      case 1: // economic units
      case 2: // military units
        if (i === 1 && (view_mode === RisqViewMode.MILITARY || view_mode === RisqViewMode.OWNERSHIP)) {
          ctx.strokeStyle = secondary_color;
          break;
        }
        if (i === 2 && (view_mode === RisqViewMode.RESOURCE || view_mode === RisqViewMode.OWNERSHIP)) {
          ctx.strokeStyle = secondary_color;
          break;
        }
        if (visibility === RisqVisibilityLevel.POOR) {
          if (i === 2 && !!zone.unit_count) {
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
          break;
        }
        const units_by_player_and_type = i === 1 ? zone.economic_units_by_type : zone.military_units_by_type;
        if (units_by_player_and_type.size === 0) {
          ctx.strokeStyle = secondary_color;
          break;
        } else if (units_by_player_and_type.size > 1) {
          // TODO: handle displaying multiplayer units => either battle or allies
          break;
        }
        const units_by_type = [...units_by_player_and_type.values()][0];
        if (units_by_type.length === 0) {
          ctx.strokeStyle = secondary_color;
        } else {
          drawUnitTypeCluster(ctx, game, units_by_type, part.r, zone.units.size, primary_color, secondary_color);
        }
        break;
      default:
        console.error('No implemented');
        break;
    }
    ctx.rotate(rotation);
    ctx.translate(-part.c.x, -part.c.y);
    drawEllipse(ctx, part.c, part.r);
  }
}

/** Resolves hover logic for the zones of a risq space */
export function resolveHoveredZones(
  p: Point2D,
  space: RisqSpace | undefined,
  r: number,
  override_center?: Point2D,
  ignore_parts?: boolean
): RisqZone | undefined {
  if (!space?.zones) {
    return undefined;
  }

  const resolve_zone_dependencies = (m: Point2D, zone: RisqZone, rotate: number) => {
    zone.hovered = true;
    if (ignore_parts) {
      return;
    }
    const p = rotatePoint(m, rotate);
    for (const part of zone.hovered_data) {
      const dx = p.x - part.c.x;
      const dy = p.y - part.c.y;
      if ((dx * dx) / (part.r.x * part.r.x) + (dy * dy) / (part.r.y * part.r.y) <= 1) {
        part.hovered = true;
      } else {
        part.hovered = false;
      }
    }
  };

  const m = subtractPoint2D({ x: p.x, y: p.y }, override_center ?? space.center);
  let new_hovered_zone: RisqZone | undefined = undefined;
  if (pointInHexagon(m, INNER_ZONE_MULTIPLIER * r)) {
    new_hovered_zone = space.zones[1][1];
    resolve_zone_dependencies(m, new_hovered_zone, 0);
  } else if (pointInHexagon(m, r)) {
    const angle = atangent(m.y, m.x);
    let index = Math.floor((angle + Math.PI / 6) / (Math.PI / 3));
    let direction_vector: Point2D = { x: 0, y: 0 };
    switch (index) {
      case 6:
        index = 0;
      case 0:
        direction_vector = { x: 1, y: 2 };
        break;
      case 1:
        direction_vector = { x: 0, y: 1 };
        break;
      case 2:
        direction_vector = { x: 0, y: 0 };
        break;
      case 3:
        direction_vector = { x: 1, y: 0 };
        break;
      case 4:
        direction_vector = { x: 2, y: 0 };
        break;
      case 5:
        direction_vector = { x: 2, y: 1 };
        break;
      default:
        console.error('Unknown zone hovered', angle);
        return;
    }
    new_hovered_zone = space.zones[direction_vector.x][direction_vector.y];
    resolve_zone_dependencies(m, new_hovered_zone, -(Math.PI / 3) * (1 + 5 - index));
  }
  return new_hovered_zone;
}

/** Removes all hovered flags from the risq zone */
export function unhoverRisqZone(zone: RisqZone) {
  if (!zone) {
    return;
  }
  zone.clicked = false;
  zone.hovered = false;
  for (const part of zone.hovered_data) {
    part.clicked = false;
    part.hovered = false;
  }
}
