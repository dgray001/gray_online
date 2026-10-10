import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawLine, drawRect } from '../../../../util/canvas_util';
import type { DwgRisq } from '../../risq';
import type { IdleCategory } from '../../application/orders/idle_category';
import { RisqActionButton } from '../left_panel/actions/action_button';

export const IDLE_FILTERS: [IdleCategory, string, string][] = [
  ['economic', 'icons/villager64', 'Idle economic units'],
  ['military', 'icons/unit64', 'Idle military units'],
  ['unit_production', 'icons/building64', 'Idle unit-producing buildings'],
  ['other_buildings', 'icons/research64', 'Idle non-unit-producing buildings'],
];

export class RisqIdleFilterButton extends RisqActionButton {
  constructor(
    private risq: DwgRisq,
    private category: IdleCategory,
    image_path: string,
    description: string,
    col: number
  ) {
    super({ row: 0, col, image_path, description }, 28);
  }
  override execute(): void {
    this.risq.toggleIdleCategory(this.category);
  }
  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    super.draw(ctx, transform, dt);
    configDraw(
      ctx,
      transform,
      { fill_style: 'white', stroke_style: 'black', stroke_width: 1, fixed_position: true },
      false,
      false,
      (): void => {
        const p = { x: this.xi(), y: this.yi() };
        const size = Math.min(10, this.w());
        drawRect(ctx, p, size, size);
        if (this.risq.planning.idleCategoryEnabled(this.category)) {
          drawLine(ctx, { x: p.x + size * 0.2, y: p.y + size * 0.5 }, { x: p.x + size * 0.4, y: p.y + size * 0.8 });
          drawLine(ctx, { x: p.x + size * 0.4, y: p.y + size * 0.8 }, { x: p.x + size * 0.8, y: p.y + size * 0.2 });
        }
      }
    );
  }
}
