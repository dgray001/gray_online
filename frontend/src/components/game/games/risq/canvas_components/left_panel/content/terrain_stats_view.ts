import type { BoardTransformData } from '../../../../../util/canvas_board/canvas_board';
import { createTooltipState, drawTooltip, shouldShowTooltip } from '../../../../../util/canvas_components/tooltip';
import { drawText } from '../../../../../util/canvas_util';
import type { Point2D } from '../../../../../util/objects2d';
import type { RectHoverData, RisqSpace, RisqTerrainStats } from '../../../model/types';
import type { DwgRisq } from '../../../risq';
import type { PanelDrawContext } from './primitives';
import { rectHovered } from './stats_view';

const ROW_HEIGHT = 18;
type TerrainStatRow = [label: string, value: string, tooltip: string];

function terrainRows(stats: RisqTerrainStats): TerrainStatRow[] {
  const movement: TerrainStatRow[] = stats.impassable
    ? [['Movement', 'impassable', 'Units cannot enter this terrain.']]
    : [
        [
          'Within space',
          `${stats.intra_cost} stamina`,
          'Stamina cost of moving one adjacent-zone step within this space.',
        ],
        [
          'Enter space',
          `${stats.inter_cost} stamina`,
          'Stamina cost of crossing into this space. The destination terrain determines the cost.',
        ],
      ];
  return [
    ...movement,
    [
      'Construction',
      `${Math.round(stats.build_speed_modifier * 100)}%`,
      'Terrain multiplier on construction progress per stamina spent, with progress rounded to a minimum of 1 per tick.',
    ],
  ];
}

export class RisqTerrainStatsView {
  private cells: { rect: RectHoverData; text: string }[] = [];
  private hovered = -1;
  private tooltip = createTooltipState();

  clearHover(): void {
    this.hovered = -1;
  }

  mousemove(m: Point2D): void {
    this.hovered = this.cells.findIndex((cell: { rect: RectHoverData; text: string }): boolean =>
      rectHovered(m, cell.rect)
    );
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    const cell = this.cells[this.hovered];
    if (shouldShowTooltip(this.tooltip, !!cell, false, dt) && cell) {
      drawTooltip(this.tooltip, ctx, transform, risq.viewport.canvasSize(), cell.text);
    }
  }

  height(space: RisqSpace): number {
    return space.terrain_stats ? (space.terrain_stats.impassable ? 2 : 3) * ROW_HEIGHT : 0;
  }

  draw(pc: PanelDrawContext, space: RisqSpace, yi: number): void {
    this.cells = [];
    if (!space.terrain_stats) {
      this.clearHover();
      return;
    }
    const x = pc.frame.xi() + 0.1 * pc.frame.w();
    const w = 0.8 * pc.frame.w();
    for (const [i, [label, value, text]] of terrainRows(space.terrain_stats).entries()) {
      const y = yi + i * ROW_HEIGHT;
      this.cells.push({ rect: { ps: { x, y }, pe: { x: x + w, y: y + ROW_HEIGHT } }, text });
      drawText(pc.ctx, label, {
        p: { x, y: y + ROW_HEIGHT / 2 },
        w: w / 2,
        fill_style: 'black',
        baseline: 'middle',
        font: '14px serif',
      });
      drawText(pc.ctx, value, {
        p: { x: x + w, y: y + ROW_HEIGHT / 2 },
        w: w / 2,
        fill_style: 'black',
        align: 'right',
        baseline: 'middle',
        font: '14px serif',
      });
    }
  }
}
