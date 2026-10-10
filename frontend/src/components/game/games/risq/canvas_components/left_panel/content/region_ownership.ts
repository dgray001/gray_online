import { drawText } from '../../../../../util/canvas_util';
import { invertPair } from '../../../model/coordinates';
import type { RisqRegion } from '../../../model/types';
import type { PanelDrawContext } from './primitives';

export function drawRegionOwnership(pc: PanelDrawContext, region: RisqRegion, yi: number): void {
  const counts = new Map<number, number>();
  for (const key of region.spaces) {
    const owner = pc.risq.session.spaceAt(invertPair(key))?.ownership ?? -1;
    counts.set(owner, (counts.get(owner) ?? 0) + 1);
  }
  const draw_count = (name: string, count: number, color: string = 'black'): void => {
    drawText(pc.ctx, `${name}: ${count} space${count === 1 ? '' : 's'}`, {
      p: { x: pc.frame.xi() + 0.1 * pc.frame.w(), y: yi },
      w: 0.8 * pc.frame.w(),
      fill_style: color,
      align: 'left',
      font: '18px serif',
    });
    yi += 28;
  };
  for (const player of pc.risq.getGame()?.players ?? []) {
    const count = counts.get(player.player.player_id) ?? 0;
    if (count > 0) {
      draw_count(player.player.nickname, count, player.color.getString());
    }
  }
  if (region.unexplored_spaces > 0) {
    draw_count('Unexplored', region.unexplored_spaces);
  }
  draw_count('Unowned', counts.get(-1) ?? 0);
}
