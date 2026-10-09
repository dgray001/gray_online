import { drawText } from '../../../util/canvas_util';
import type { RisqRegion } from '../model/types';
import { RisqResourceType } from '../model/types';
import { resourceTypeImage } from './assets/resources';
import type { RisqDrawHost } from './draw_host';
import { spaceOwnerColor } from './terrain';

/** Draws each known region's name and gold bonus at its centroid; used by the REGION view mode */
export function drawRisqRegionLabels(ctx: CanvasRenderingContext2D, game: RisqDrawHost): void {
  const game_data = game.getGame();
  if (!game_data) {
    return;
  }
  const drawn = new Set<RisqRegion>();
  for (const region of game.session.getRegionLookup().values()) {
    if (drawn.has(region)) {
      continue;
    }
    drawn.add(region);
    const p = game.viewport.regionLabelPosition(region);
    const owner_color = region.owner >= 0 ? spaceOwnerColor(region.owner, game_data.players) : undefined;
    drawText(ctx, region.name, {
      p: { x: p.x, y: p.y - 8 },
      w: 220,
      fill_style: owner_color ? owner_color.getString() : 'white',
      stroke_style: 'black',
      stroke_width: 3,
      align: 'center',
      baseline: 'middle',
      font: 'bold 16px serif',
    });
    const bonus_text = `+${region.gold_bonus}/turn`;
    ctx.save();
    ctx.font = '13px serif';
    const bonus_width = ctx.measureText(bonus_text).width;
    ctx.restore();
    const bonus_left = p.x - (bonus_width + 20) / 2;
    ctx.drawImage(game.getIcon(resourceTypeImage(RisqResourceType.GOLD)), bonus_left, p.y + 2, 16, 16);
    drawText(ctx, bonus_text, {
      p: { x: bonus_left + 20, y: p.y + 10 },
      w: 220,
      fill_style: 'rgb(255, 215, 0)',
      stroke_style: 'black',
      stroke_width: 2,
      align: 'left',
      baseline: 'middle',
      font: '13px serif',
    });
  }
}
