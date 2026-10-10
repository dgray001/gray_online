import type { Point2D } from '../../../../util/objects2d';
import type { RisqZone, UnitByTypeData } from '../../model/types';
import type { RisqDrawHost } from '../draw_host';

export function drawControlGroupLabel(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  zone: RisqZone,
  slot: UnitByTypeData[] | undefined,
  r: Point2D
): void {
  const player_id = game.getPlayerId();
  const ids = slot
    ? slot
        .filter((group) => group.player_id === player_id)
        .flatMap((group) => [...group.units].filter((id) => game.selection.isUnitSelected(id)))
    : zone.building?.player_id === player_id
      ? [zone.building.internal_id]
      : [];
  const groups = game.control_groups?.memberships(slot ? 'unit' : 'building', ids);
  if (!groups?.length) {
    return;
  }
  const transform = ctx.getTransform();
  const scale = Math.hypot(transform.a, transform.b);
  const gap = 2 / scale;
  ctx.fillStyle = 'white';
  ctx.font = `bold ${10 / scale}px serif`;
  ctx.textAlign = 'right';
  ctx.textBaseline = 'bottom';
  ctx.fillText(groups.map((number) => number % 10).join(','), -Math.SQRT1_2 * r.x - gap, -Math.SQRT1_2 * r.y - gap);
}
