import { drawText } from '../../../../util/canvas_util';
import type { ReplayEvent, ReplayHiredUnit } from '../../model/replay_event';
import { hiredUnitLabel } from '../../model/replay_event';
import type { DwgRisq } from '../../risq';
import { unitImage } from '../../rendering/assets/unit';

function label(
  ctx: CanvasRenderingContext2D,
  value: string,
  x: number,
  y: number,
  width: number,
  color: string = 'black'
): void {
  drawText(ctx, value, { p: { x, y }, w: width, font: '18px sans-serif', fill_style: color });
}

export function drawHiredDetails(
  ctx: CanvasRenderingContext2D,
  risq: DwgRisq,
  event: ReplayEvent,
  x: number,
  y: number,
  width: number
): void {
  const units = event.hired_units ?? [];
  const player = risq.getGame()?.players[event.player_id];
  const header = event.location_name ? 78 : 54;
  label(ctx, 'Hired', x, y, width);
  label(
    ctx,
    player?.player.nickname ?? `Player ${event.player_id}`,
    x,
    y + 24,
    width,
    player?.color.getString() ?? 'black'
  );
  if (event.location_name) {
    label(ctx, event.location_name, x, y + 48, width);
  }
  units.forEach((unit: ReplayHiredUnit, index: number): void => {
    ctx.drawImage(risq.getIcon(unitImage(unit.unit_id, true)), x, y + header + index * 32, 24, 24);
    label(ctx, hiredUnitLabel(unit), x + 32, y + header + 2 + index * 32, width - 32);
  });
  label(ctx, 'End of turn', x, y + header + units.length * 32, width);
}
