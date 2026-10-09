import { capitalize } from '../../../../../../../scripts/util';
import { drawRect, drawText } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import type { RisqBuilding, RisqUnit, UnitByTypeData } from '../../../model/types';
import { RisqUnitType } from '../../../model/types';
import {
  UNIT_HEALTHBAR_COLOR_BACKGROUND,
  UNIT_HEALTHBAR_COLOR_HEALTH,
  unitCountLabel,
  unitImage,
} from '../../../rendering/assets/unit';
import type { DwgRisq } from '../../../risq';
import { ACTION_GRID_COLS, PANEL_PADDING } from '../layout';
import { unitResolver } from '../selection_queries';
import type { PanelDrawContext } from './primitives';
import { drawHeaderImage, drawName, drawOwnerSeparators, drawSeparator, drawSubtitle } from './primitives';

/** Draws a unit's icon and health bar, and records its rect as the unit's hover target */
export function drawUnitImage(pc: PanelDrawContext, unit: RisqUnit, p: Point2D, s: number): void {
  const { ctx, risq } = pc;
  const color = risq.getGame()?.players[unit.player_id]?.color;
  const icon = color
    ? risq.getPlayerColoredIcon(unitImage(unit.unit_id), color)
    : risq.getIcon(unitImage(unit.unit_id));
  ctx.drawImage(icon, p.x, p.y, s, s);
  ctx.strokeStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
  ctx.lineWidth = 0.4;
  ctx.fillStyle = UNIT_HEALTHBAR_COLOR_BACKGROUND;
  drawRect(ctx, { x: p.x, y: p.y + 0.8 * s }, s, 0.18 * s);
  if (unit.combat_stats.max_health > 0 && unit.combat_stats.health > 0) {
    ctx.fillStyle = UNIT_HEALTHBAR_COLOR_HEALTH;
    drawRect(ctx, { x: p.x, y: p.y + 0.8 * s }, (unit.combat_stats.health / unit.combat_stats.max_health) * s, 0.2 * s);
  }
  if (unit.hover_data.hovered) {
    ctx.fillStyle = unit.hover_data.clicked ? 'rgba(250, 250, 250, 0.4)' : 'rgba(220, 220, 220, 0.2)';
    ctx.strokeStyle = 'transparent';
    drawRect(ctx, p, s, s);
  }
  unit.hover_data.ps = p;
  unit.hover_data.pe = { x: p.x + s, y: p.y + s };
}

export function garrisonedUnits(risq: DwgRisq, building: RisqBuilding): RisqUnit[] {
  const resolve = unitResolver(risq);
  return (building.garrisoned_units ?? [])
    .map((internal_id) => resolve(building.player_id, internal_id))
    .filter((u): u is RisqUnit => !!u);
}

/** Garrisoned units fill the garrison band's grid cells in order, with empty outlined cells for the rest */
export function drawGarrisonedUnits(pc: PanelDrawContext, building: RisqBuilding): void {
  const s = pc.layout.grid_s;
  const P = PANEL_PADDING;
  const top = pc.layout.separator_below_stats + P;
  const units = garrisonedUnits(pc.risq, building);
  const slot_p = (i: number): Point2D => ({
    x: pc.layout.grid_x0 + (i % ACTION_GRID_COLS) * (s + P),
    y: top + Math.floor(i / ACTION_GRID_COLS) * (s + P),
  });
  for (const [i, unit] of units.entries()) {
    drawUnitImage(pc, unit, slot_p(i), s);
  }
  pc.ctx.fillStyle = 'transparent';
  pc.ctx.strokeStyle = 'black';
  pc.ctx.lineWidth = 1;
  for (let i = units.length; i < building.garrison_capacity; i++) {
    drawRect(pc.ctx, slot_p(i), s, s);
  }
}

function drawUnitCountBadge(ctx: CanvasRenderingContext2D, count: number, p: Point2D, s: number): void {
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

function drawUnitCountBlock(
  pc: PanelDrawContext,
  player_id: number,
  unit_id: number,
  count: number,
  p: Point2D,
  s: number,
  units: UnitByTypeData[]
): void {
  const color = pc.risq.getGame()?.players[player_id]?.color;
  const icon = color ? pc.risq.getPlayerColoredIcon(unitImage(unit_id), color) : pc.risq.getIcon(unitImage(unit_id));
  pc.ctx.drawImage(icon, p.x, p.y, s, s);
  drawUnitCountBadge(pc.ctx, count, p, s);
  drawGroupHighlight(pc, units, p, s);
}

function drawGroupHighlight(pc: PanelDrawContext, units: UnitByTypeData[], p: Point2D, s: number): void {
  pc.ctx.fillStyle = pc.groupTile?.(units, p, s) ?? 'transparent';
  pc.ctx.strokeStyle = 'transparent';
  drawRect(pc.ctx, p, s, s);
}

function chunkBlocks<T>(items: T[], size: number): T[][] {
  const rows: T[][] = [];
  for (let i = 0; i < items.length; i += size) {
    rows.push(items.slice(i, i + size));
  }
  return rows;
}

function unitGroupTitle(risq: DwgRisq, groups: [number, UnitByTypeData[]][], total: number): string {
  const types = groups.flatMap(([, units]) => units).filter((t) => t.units.size > 0);
  if (new Set(types.map((t) => t.unit_id)).size === 1) {
    const sample = unitResolver(risq)(types[0].player_id, [...types[0].units][0]);
    return unitCountLabel(sample?.display_name ?? 'Unit', total);
  }
  const unit_types = new Set(types.map((t) => t.unit_type));
  const units_word = total === 1 ? 'Unit' : 'Units';
  if (unit_types.size === 1) {
    return `${total} ${capitalize(RisqUnitType[types[0].unit_type].replace(/_/g, ' '))} ${units_word}`;
  }
  return unit_types.has(RisqUnitType.ECONOMIC) ? `${total} ${units_word}` : `${total} Military ${units_word}`;
}

function unitGroupSubtitle(risq: DwgRisq, groups: [number, UnitByTypeData[]][]): string | undefined {
  const player_ids = groups.filter(([, units]) => units.some((u) => u.units.size > 0)).map(([id]) => id);
  if (player_ids.length > 1) {
    return `Units from ${player_ids.length} players`;
  }
  if (player_ids[0] === undefined || player_ids[0] === risq.getPlayer()?.player.player_id) {
    return undefined;
  }
  return `Owned by ${risq.getGame()?.players[player_ids[0]]?.player.nickname ?? 'another player'}`;
}

interface UnitRef {
  player_id: number;
  internal_id: number;
}

/** Draws a unit selection, coarsening from individual units to per-type, per-category, then per-player blocks until it fits */
export function drawUnitsGeneric(pc: PanelDrawContext, groups: [number, UnitByTypeData[]][]): void {
  const multi_player = groups.length > 1;
  const single_owner_player_id = multi_player ? undefined : groups[0]?.[0];
  const total_units = groups.reduce((sum, [, units]) => sum + units.reduce((s, u) => s + u.units.size, 0), 0);
  let content_yi = pc.frame.yi() + drawName(pc, unitGroupTitle(pc.risq, groups, total_units));
  const subtitle = unitGroupSubtitle(pc.risq, groups);
  if (subtitle) {
    content_yi += drawSubtitle(pc, subtitle, content_yi);
  }
  const gap = PANEL_PADDING;
  const block_size = pc.layout.grid_s;
  const grid_bottom = pc.layout.separator_below_stats;
  const max_rows = Math.floor((grid_bottom - gap - content_yi) / (block_size + gap));
  const separators = (): boolean => drawOwnerSeparators(pc, single_owner_player_id);
  const per_player_refs: [number, UnitRef[]][] = groups
    .map(([player_id, units]): [number, UnitRef[]] => [
      player_id,
      units.flatMap((u) => [...u.units].map((internal_id) => ({ player_id, internal_id }))),
    ])
    .filter(([, refs]) => refs.length > 0);
  const tier1_row_groups = per_player_refs.map(([, refs]) => chunkBlocks(refs, ACTION_GRID_COLS));
  const tier1_rows = tier1_row_groups.reduce((sum, rows) => sum + rows.length, 0);
  if (tier1_rows > 0 && tier1_rows <= max_rows) {
    drawPerPlayerUnitRows(pc, per_player_refs, tier1_row_groups, tier1_rows, multi_player);
    separators();
    return;
  }
  const tier2_row_groups = chunkBlocks(
    per_player_refs.flatMap(([, refs]) => refs),
    ACTION_GRID_COLS
  );
  if (tier2_row_groups.length > 0 && tier2_row_groups.length <= max_rows) {
    layoutRows(pc, tier2_row_groups, (ref, p) => drawUnitRef(pc, ref, p));
    separators();
    return;
  }
  const id_blocks = groups.flatMap(([player_id, units]) =>
    units.filter((u) => u.units.size > 0).map((u) => ({ player_id, unit_id: u.unit_id, count: u.units.size }))
  );
  const tier3_row_groups = chunkBlocks(id_blocks, ACTION_GRID_COLS);
  if (tier3_row_groups.length > 0 && tier3_row_groups.length <= max_rows) {
    layoutRows(pc, tier3_row_groups, (b, p) =>
      drawUnitCountBlock(
        pc,
        b.player_id,
        b.unit_id,
        b.count,
        p,
        block_size,
        groups.find(([id]) => id === b.player_id)![1].filter((u) => u.unit_id === b.unit_id)
      )
    );
    separators();
    return;
  }
  const type_blocks = unitTypeBlocks(pc.risq, groups);
  const tier4_row_groups = chunkBlocks(type_blocks, ACTION_GRID_COLS);
  if (type_blocks.length > 0 && (tier4_row_groups.length <= max_rows || !multi_player)) {
    layoutRows(pc, tier4_row_groups, (b, p) =>
      drawUnitCountBlock(
        pc,
        b.player_id,
        b.representative_unit_id,
        b.count,
        p,
        block_size,
        groups.find(([id]) => id === b.player_id)![1].filter((u) => u.unit_type === b.unit_type)
      )
    );
    separators();
    return;
  }
  // only reachable for a multi-player selection
  const player_blocks = groups
    .map(([player_id, units]) => ({ player_id, count: units.reduce((s, u) => s + u.units.size, 0) }))
    .filter((b) => b.count > 0);
  layoutRows(pc, chunkBlocks(player_blocks, ACTION_GRID_COLS), (block, p) => {
    const color = pc.risq.getGame()?.players[block.player_id]?.color;
    pc.ctx.fillStyle = color ? color.getString() : 'rgba(255, 255, 255, 0.3)';
    pc.ctx.strokeStyle = 'black';
    pc.ctx.lineWidth = 1;
    drawRect(pc.ctx, p, block_size, block_size);
    drawUnitCountBadge(pc.ctx, block.count, p, block_size);
    drawGroupHighlight(pc, groups.find(([id]) => id === block.player_id)![1], p, block_size);
  });
  separators();
}

/** Rows are stacked upward from the stats separator, so the last row sits right above it */
function layoutRows<T>(pc: PanelDrawContext, row_groups: T[][], draw_item: (item: T, p: Point2D) => void): void {
  const gap = PANEL_PADDING;
  const block_size = pc.layout.grid_s;
  let by = pc.layout.separator_below_stats - row_groups.length * (block_size + gap);
  for (const row of [...row_groups].reverse()) {
    let bx = pc.layout.grid_x0;
    for (const item of row) {
      draw_item(item, { x: bx, y: by });
      bx += block_size + gap;
    }
    by += block_size + gap;
  }
}

function drawUnitRef(pc: PanelDrawContext, ref: UnitRef, p: Point2D): void {
  const unit = unitResolver(pc.risq)(ref.player_id, ref.internal_id);
  if (unit) {
    drawUnitImage(pc, unit, p, pc.layout.grid_s);
  }
}

/** Individual units with one row-run per player, tinted per player when more than one player is shown */
function drawPerPlayerUnitRows(
  pc: PanelDrawContext,
  per_player_refs: [number, UnitRef[]][],
  row_groups: UnitRef[][][],
  total_rows: number,
  multi_player: boolean
): void {
  const gap = PANEL_PADDING;
  const block_size = pc.layout.grid_s;
  const content_w = ACTION_GRID_COLS * block_size + (ACTION_GRID_COLS - 1) * gap;
  let by = pc.layout.separator_below_stats - total_rows * (block_size + gap);
  for (const [i, [player_id]] of [...per_player_refs.entries()].reverse()) {
    const rows = row_groups[i];
    if (multi_player) {
      const color = pc.risq.getGame()?.players[player_id]?.color;
      pc.ctx.fillStyle = color
        ? `rgba(${color.getR()}, ${color.getG()}, ${color.getB()}, 0.12)`
        : 'rgba(255, 255, 255, 0.06)';
      pc.ctx.strokeStyle = 'transparent';
      drawRect(
        pc.ctx,
        { x: pc.layout.grid_x0 - 0.5 * gap, y: by - 0.5 * gap },
        content_w + gap,
        rows.length * (block_size + gap)
      );
    }
    for (const row of [...rows].reverse()) {
      let bx = pc.layout.grid_x0;
      for (const ref of row) {
        drawUnitRef(pc, ref, { x: bx, y: by });
        bx += block_size + gap;
      }
      by += block_size + gap;
    }
  }
}

interface TypeBlock {
  player_id: number;
  unit_type: RisqUnitType;
  representative_unit_id: number;
  count: number;
}

/** One block per (player, unit_type), merging unit_ids that share a type */
function unitTypeBlocks(risq: DwgRisq, groups: [number, UnitByTypeData[]][]): TypeBlock[] {
  const resolve = unitResolver(risq);
  const type_blocks_by_key = new Map<string, TypeBlock>();
  for (const [player_id, units] of groups) {
    for (const u of units) {
      if (u.units.size < 1) {
        continue;
      }
      const unit_type = resolve(player_id, [...u.units][0])?.unit_type ?? RisqUnitType.NONE;
      const key = `${player_id}:${unit_type}`;
      const existing = type_blocks_by_key.get(key);
      if (existing) {
        existing.count += u.units.size;
      } else {
        type_blocks_by_key.set(key, { player_id, unit_type, representative_unit_id: u.unit_id, count: u.units.size });
      }
    }
  }
  return [...type_blocks_by_key.values()];
}

export function drawUnit(pc: PanelDrawContext, unit: RisqUnit): void {
  let yi = pc.frame.yi() + drawName(pc, unit.display_name);
  yi += drawHeaderImage(pc, yi, unitImage(unit.unit_id), pc.risq.getGame()?.players[unit.player_id]?.color);
  drawSeparator(pc, yi);
  pc.stats.draw(pc, pc.frame.yi() + 0.25 * pc.frame.h() + 6, unit);
  drawOwnerSeparators(pc, unit.player_id);
}
