import { err } from '../../../../../../scripts/log';
import type { Point2D } from '../../../../util/objects2d';
import type { UnitByTypeData } from '../../model/types';
import type { RisqDrawHost } from '../board/board_renderer';
import { scaledIcon, transformScale } from './image_cache';

export const UNIT_HEALTHBAR_COLOR_BACKGROUND = 'black';
export const UNIT_HEALTHBAR_COLOR_HEALTH = 'rgb(100, 250, 100)';

/** Returns image path of the unit; pass plain to get the uncolored source (no player-color marker pixels) */
export function unitImage(unit_id: number, plain = false): string {
  let filename = '';
  switch (unit_id) {
    case 1:
      filename = 'villager';
      break;
    case 11:
      filename = 'blunt_infantry';
      break;
    case 12:
      filename = 'piercing_infantry';
      break;
    case 13:
      filename = 'heavy_infantry';
      break;
    default:
      err('Trying to get unit image from unknown unit id', unit_id);
      return 'default';
  }
  if (plain) {
    filename += '_plain';
  }
  return `risq/units/${filename}`;
}

export function unitCorpseImage(unit_id: number, turns: number): string {
  return `${unitImage(unit_id)}_${turns === 1 ? 'corpse' : 'decayed'}`;
}

const COLLECTIVE_NAME_SUFFIXES = ['Infantry', 'Cavalry'];

export function unitCountLabel(display_name: string, count: number): string {
  const is_collective = COLLECTIVE_NAME_SUFFIXES.some((suffix) => display_name.endsWith(suffix));
  return `${count} ${count === 1 || is_collective ? display_name : `${display_name}s`}`;
}

export const COMBO_UNIT_ICON_SIZE = 64;

export function comboUnitIconKey(white: boolean): string {
  return white ? 'combo_unit_white' : 'combo_unit';
}

export function drawComboUnitIcon(
  ctx: CanvasRenderingContext2D,
  villager_icon: HTMLImageElement,
  unit_icon: HTMLImageElement
): void {
  const icon_size = 0.72 * COMBO_UNIT_ICON_SIZE;
  ctx.drawImage(villager_icon, 0, 0, icon_size, icon_size);
  ctx.drawImage(unit_icon, COMBO_UNIT_ICON_SIZE - icon_size, COMBO_UNIT_ICON_SIZE - icon_size, icon_size, icon_size);
}

export const UNIT_CLUSTER_ICON_SIZE = 64;

export function drawUnitCount(
  ctx: CanvasRenderingContext2D,
  text: string,
  size: number,
  x: number,
  y: number,
  width: number,
  color: string,
  anchor_top_left = false
): void {
  if (text === '1') {
    return;
  }
  const fill = ctx.fillStyle;
  ctx.fillStyle = color;
  ctx.font = `bold ${size}px serif`;
  if (anchor_top_left) {
    const align = ctx.textAlign;
    const baseline = ctx.textBaseline;
    ctx.textAlign = 'left';
    ctx.textBaseline = 'alphabetic';
    const bounds = ctx.measureText(text);
    const scale = Math.min(1, width / bounds.width);
    ctx.fillText(text, x + bounds.actualBoundingBoxLeft * scale, y + bounds.actualBoundingBoxAscent, width);
    ctx.textAlign = align;
    ctx.textBaseline = baseline;
  } else {
    ctx.fillText(text, x, y, width);
  }
  ctx.fillStyle = fill;
}

export function unitClusterIconKey(units_by_type: UnitByTypeData[], total: number): string {
  return 'unit_cluster:' + units_by_type.map((t) => `${t.unit_id}x${t.units.size}`).join(',') + `:${total}`;
}

export function drawUnitTypeCluster(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  units_by_type: UnitByTypeData[],
  r: Point2D,
  total: number,
  primary_color: string,
  secondary_color: string,
  count_at_center = false
): void {
  const draw_count = (s: string, ts: number, x: number, y: number, w: number, fill_primary = true): void => {
    if (count_at_center && fill_primary) {
      const whole_cluster = units_by_type.length === 1 || units_by_type.length > 4;
      const icon_scale = units_by_type.length === 2 ? 1.5 : 1;
      const center_x = whole_cluster ? 0 : x + (icon_scale * r.x) / 2;
      const center_y = whole_cluster ? 0 : y + (icon_scale * r.y) / 2;
      drawUnitCount(ctx, s, ts, center_x, center_y, w, primary_color, true);
    } else {
      drawUnitCount(ctx, s, ts, x, y, w, fill_primary ? primary_color : secondary_color);
    }
  };
  const scale = transformScale(ctx);
  const icon = (t: UnitByTypeData, size: number): CanvasImageSource =>
    scaledIcon(game, unitImage(t.unit_id), game.getGame()?.players[t.player_id]?.color, size * scale);
  if (units_by_type.length === 1) {
    const t = units_by_type[0];
    ctx.drawImage(icon(t, 2 * r.x), -r.x, -r.y, 2 * r.x, 2 * r.y);
    draw_count(t.units.size.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  } else if (units_by_type.length === 2) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t, 1.5 * r.x), (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 1.5 * r.x, 1.5 * r.y);
      draw_count(t.units.size.toString(), r.y, (0.5 * j - 1) * r.x, (0.5 * j - 1) * r.y, 2 * r.x);
    }
  } else if (units_by_type.length === 3 || units_by_type.length === 4) {
    for (let j = 0; j < 2; j++) {
      const t = units_by_type[j];
      ctx.drawImage(icon(t, r.x), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.9 * r.y, 1.5 * r.x);
    }
    if (units_by_type.length === 3) {
      const t = units_by_type[2];
      ctx.drawImage(icon(t, r.x), -0.5 * r.x, -0.1 * r.y, r.x, r.y);
      draw_count(t.units.size.toString(), 0.75 * r.y, -0.5 * r.x, -0.1 * r.y, 1.5 * r.x);
    } else {
      for (let j = 0; j < 2; j++) {
        const t = units_by_type[2 + j];
        ctx.drawImage(icon(t, r.x), (0.8 * j - 0.9) * r.x, -0.1 * r.y, r.x, r.y);
        draw_count(t.units.size.toString(), 0.75 * r.y, (0.8 * j - 0.9) * r.x, -0.1 * r.y, 1.5 * r.x);
      }
    }
  } else {
    for (let j = 0; j < 2; j++) {
      ctx.drawImage(icon(units_by_type[j], r.x), (0.8 * j - 0.9) * r.x, -0.9 * r.y, r.x, r.y);
    }
    ctx.drawImage(icon(units_by_type[2], r.x), -0.9 * r.x, -0.1 * r.y, r.x, r.y);
    draw_count('...', 0.8 * r.y, 0.1 * r.x, -0.1 * r.y, r.x, false);
    draw_count(total.toString(), 1.4 * r.y, -r.x, -0.7 * r.y, 2 * r.x);
  }
}
