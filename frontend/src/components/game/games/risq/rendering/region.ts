import { drawRect } from '../../../util/canvas_util';
import type { Point2D } from '../../../util/objects2d';
import type { RisqRegion } from '../model/types';
import { RisqResourceType } from '../model/types';
import { resourceTypeImage } from './assets/resources';
import type { RisqDrawHost } from './draw_host';
import { regionLabelAnchor } from './region_label_anchor';
import type { DrawRisqSpaceConfig } from './space';
import { DrawRisqSpaceDetail } from './space';
import { spaceOwnerColor } from './terrain';

const MIN_OVERVIEW_ROW_SCREEN_PX = 14;
const OVERVIEW_ROW_CANVAS_PX = 16;

declare interface RegionLabelPlacement {
  center: Point2D;
  row_h: number;
  offset_y: number; // upright offset from center to the middle of the two-row block
}

function labelPlacement(game: RisqDrawHost, region: RisqRegion, config: DrawRisqSpaceConfig): RegionLabelPlacement {
  const centroid = game.viewport.regionLabelPosition(region);
  if (config.draw_detail === DrawRisqSpaceDetail.OWNERSHIP) {
    const min_row = MIN_OVERVIEW_ROW_SCREEN_PX / game.viewport.lastTransform().scale;
    return { center: centroid, row_h: Math.max(OVERVIEW_ROW_CANVAS_PX, min_row), offset_y: 0 };
  }
  const anchor = regionLabelAnchor(region, (c) => game.viewport.coordinateToCanvas(c), centroid);
  // a lone space keeps its first inset row free for the per-space gold amount
  const offset_y = anchor.shared_spaces === 1 ? config.inset_row + config.inset_gap : 0;
  return { center: anchor.point, row_h: config.inset_row, offset_y };
}

/** Two rows (name, gold bonus) of equal font size on a small dark box, drawn upright at the placement */
function drawRegionLabel(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  region: RisqRegion,
  placement: RegionLabelPlacement,
  config: DrawRisqSpaceConfig
): void {
  const { center, row_h, offset_y } = placement;
  const owner_color = region.owner >= 0 ? spaceOwnerColor(region.owner, game.getGame()?.players ?? []) : undefined;
  const bonus_text = `+${region.gold_bonus}/turn`;
  ctx.save();
  ctx.translate(center.x, center.y);
  ctx.rotate(-config.rotation);
  ctx.font = `bold ${row_h}px serif`;
  const bonus_w = 1.25 * row_h + ctx.measureText(bonus_text).width;
  const w = Math.max(ctx.measureText(region.name).width, bonus_w);
  const h = 2 * row_h + config.inset_gap;
  const pad = 0.3 * row_h;
  ctx.fillStyle = 'rgba(0, 0, 0, 0.65)';
  ctx.strokeStyle = owner_color ? owner_color.getString() : 'rgba(255, 255, 255, 0.6)';
  ctx.lineWidth = 0.1 * row_h;
  drawRect(ctx, { x: -w / 2 - pad, y: offset_y - h / 2 - pad }, w + 2 * pad, h + 2 * pad, 0.4 * row_h);
  const row_offset = 0.5 * (row_h + config.inset_gap);
  ctx.textBaseline = 'middle';
  ctx.textAlign = 'center';
  ctx.fillStyle = 'white';
  ctx.fillText(region.name, 0, offset_y - row_offset);
  const gold_icon = game.getIcon(resourceTypeImage(RisqResourceType.GOLD));
  ctx.drawImage(gold_icon, -bonus_w / 2, offset_y + row_offset - row_h / 2, row_h, row_h);
  ctx.textAlign = 'left';
  ctx.fillStyle = 'rgb(255, 215, 0)';
  ctx.fillText(bonus_text, -bonus_w / 2 + 1.25 * row_h, offset_y + row_offset);
  ctx.restore();
}

/** Draws each known region's name and gold bonus; used by the ownership view mode */
export function drawRisqRegionLabels(
  ctx: CanvasRenderingContext2D,
  game: RisqDrawHost,
  config: DrawRisqSpaceConfig
): void {
  const drawn = new Set<RisqRegion>();
  for (const region of game.session.getRegionLookup().values()) {
    if (drawn.has(region)) {
      continue;
    }
    drawn.add(region);
    drawRegionLabel(ctx, game, region, labelPlacement(game, region, config), config);
  }
}
