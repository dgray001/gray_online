import type { BoardTransformData } from '../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../util/canvas_components/canvas_component';
import type { Point2D } from '../../../util/objects2d';
import { drawRect, drawText } from '../../../util/canvas_util';
import type { DwgRisq } from '../risq';
import type { RisqCost } from '../risq_data';
import { RisqResourceType } from '../risq_data';
import { resourceTypeImage } from '../risq_resources';

export declare interface RisqTooltipData {
  title: string;
  description?: string;
  cost?: RisqCost;
  stamina_cost?: number;
  // icon path, value; stats of the thing being made/researched (e.g. health, attack)
  stats?: [string, number][];
  // TODO: optional hotkey row, shown below stats once buttons have assigned hotkeys
}

const PADDING = 4;
const ROW_HEIGHT = 16;
const ICON_SIZE = ROW_HEIGHT - 2;
const ICON_TEXT_GAP = 4;
const STAT_GAP = 6;
const TITLE_COST_GAP = 10;
const FONT = '12px serif';

function costStats(data: RisqTooltipData): [string, string][] {
  const entries: [string, number][] = [];
  if (!!data.cost) {
    entries.push(
      [resourceTypeImage(RisqResourceType.FOOD), data.cost.food],
      [resourceTypeImage(RisqResourceType.WOOD), data.cost.wood],
      [resourceTypeImage(RisqResourceType.STONE), data.cost.stone],
      [resourceTypeImage(RisqResourceType.GOLD), data.cost.gold]
    );
  }
  if (data.stamina_cost !== undefined) {
    entries.push(['risq/icons/stamina', data.stamina_cost]);
  }
  return entries.filter(([, value]) => !!value).map(([icon, value]) => [icon, value.toString()]);
}

function measureStatsRow(ctx: CanvasRenderingContext2D, stats: [string, string][]): number {
  if (!stats.length) {
    return 0;
  }
  return (
    stats.reduce((sum, [, text]) => sum + ICON_SIZE + ICON_TEXT_GAP + ctx.measureText(text).width, 0) +
    STAT_GAP * (stats.length - 1)
  );
}

function drawStatsRow(ctx: CanvasRenderingContext2D, risq: DwgRisq, stats: [string, string][], x: number, y: number) {
  let xi = x;
  for (const [icon, text] of stats) {
    ctx.drawImage(risq.getIcon(icon), xi, y, ICON_SIZE, ICON_SIZE);
    const text_x = xi + ICON_SIZE + ICON_TEXT_GAP;
    const text_w = ctx.measureText(text).width;
    drawText(ctx, text, {
      p: { x: text_x, y: y + 0.5 * ICON_SIZE },
      w: text_w + 2,
      fill_style: 'white',
      align: 'left',
      baseline: 'middle',
      font: FONT,
    });
    xi = text_x + text_w + STAT_GAP;
  }
}

/** Draws a tooltip with its bottom-left corner anchored at p, given the input risq to fetch icons from */
export function drawRisqTooltip(
  ctx: CanvasRenderingContext2D,
  transform: BoardTransformData,
  risq: DwgRisq,
  p: Point2D,
  data: RisqTooltipData
) {
  configDraw(ctx, transform, { fill_style: 'transparent', stroke_width: 0, fixed_position: true }, false, false, () => {
    ctx.font = FONT;
    const cost_stats = costStats(data);
    const stats = (data.stats ?? []).map(([icon, value]) => [icon, value.toString()] as [string, string]);
    const title_w = ctx.measureText(data.title).width;
    const row1_w = title_w + (cost_stats.length ? TITLE_COST_GAP + measureStatsRow(ctx, cost_stats) : 0);
    const desc_w = data.description ? ctx.measureText(data.description).width : 0;
    const stats_w = measureStatsRow(ctx, stats);
    const w = Math.max(row1_w, desc_w, stats_w) + 2 * PADDING;
    const num_rows = 1 + (data.description ? 1 : 0) + (stats.length ? 1 : 0);
    const h = num_rows * ROW_HEIGHT + 2 * PADDING;
    const box_p = { x: p.x, y: p.y - 2 - h };
    ctx.fillStyle = 'rgba(0, 0, 0, 0.85)';
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    ctx.lineWidth = 1;
    drawRect(ctx, box_p, w, h, 3);
    let row_y = box_p.y + PADDING + 0.5 * ROW_HEIGHT;
    drawText(ctx, data.title, {
      p: { x: box_p.x + PADDING, y: row_y },
      w: title_w + 2,
      fill_style: 'white',
      align: 'left',
      baseline: 'middle',
      font: FONT,
    });
    if (cost_stats.length) {
      const costs_w = measureStatsRow(ctx, cost_stats);
      drawStatsRow(ctx, risq, cost_stats, box_p.x + w - PADDING - costs_w, row_y - 0.5 * ICON_SIZE);
    }
    if (data.description) {
      row_y += ROW_HEIGHT;
      drawText(ctx, data.description, {
        p: { x: box_p.x + PADDING, y: row_y },
        w: w - 2 * PADDING,
        fill_style: 'white',
        align: 'left',
        baseline: 'middle',
        font: FONT,
      });
    }
    if (stats.length) {
      row_y += ROW_HEIGHT;
      drawStatsRow(ctx, risq, stats, box_p.x + PADDING, row_y - 0.5 * ICON_SIZE);
    }
  });
}
