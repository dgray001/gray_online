import type { Point2D } from '../../../../util/objects2d';
import type { DwgRisq } from '../../risq';
import type { UnitByTypeData } from '../../model/types';
import { RisqUnitType } from '../../model/types';
import { UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER } from '../zones/geometry';
import { drawUnitTypeCluster } from '../zones/draw';

const GHOST_ALPHA = 0.6;

/** Collapses pending hires into at most `capacity` slots: one per hire, then one per unit type, then one for all */
export function mercenaryGhostSlots(player_id: number, unit_ids: number[], capacity: number): UnitByTypeData[][] {
  const group = (unit_id: number, indices: number[]): UnitByTypeData => ({
    player_id,
    unit_id,
    unit_type: RisqUnitType.NONE,
    units: new Set(indices),
  });
  if (unit_ids.length <= capacity) {
    return unit_ids.map((unit_id, i) => [group(unit_id, [i])]);
  }
  const indices_by_type = new Map<number, number[]>();
  unit_ids.forEach((unit_id, i) => indices_by_type.set(unit_id, [...(indices_by_type.get(unit_id) ?? []), i]));
  const by_type = [...indices_by_type.entries()].map(([unit_id, indices]) => group(unit_id, indices));
  return by_type.length <= capacity ? by_type.map((g) => [g]) : [by_type];
}

export function drawMercenaryGhosts(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  slots: UnitByTypeData[][],
  positions: Point2D[],
  hex_r: number,
  rotation: number
) {
  const unit_r = UNIT_SLOT_CIRCLE_RADIUS_MULTIPLIER * hex_r;
  ctx.globalAlpha = GHOST_ALPHA;
  ctx.textAlign = 'left';
  ctx.textBaseline = 'top';
  for (const [i, slot] of slots.entries()) {
    ctx.save();
    ctx.translate(positions[i].x, positions[i].y);
    ctx.rotate(-rotation);
    const total = slot.reduce((sum, t) => sum + t.units.size, 0);
    drawUnitTypeCluster(ctx, risq, slot, { x: unit_r, y: unit_r }, total, 'white', 'rgba(210, 210, 210, 0.4)');
    ctx.restore();
  }
  ctx.globalAlpha = 1;
}
