import { DEV } from '../../../../../../scripts/util';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { multiplyPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { RisqSpace } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { drawRisqRegionBorders, drawRisqRegionLabels } from '../region';
import type { DrawRisqSpaceConfig } from '../space';
import { drawRisqSpace, drawRisqSpaceBorder } from '../space';
import { RisqViewMode } from '../terrain';
import type { RisqOrderPaths } from './order_paths';
import type { RisqViewport } from './viewport';

const DRAW_CENTER_DOT = false;

// the inset rect must stay inside the hex's incircle (radius = hex_a) so it never pokes out as the map rotates
const INSET_RATIO = 1.0392;

/** Draws one frame: visible spaces, borders, regions, order overlays, drag selection, then screen-fixed components */
export class RisqBoardRenderer {
  private last_time = Date.now();

  constructor(
    private risq: DwgRisq,
    private session: RisqSession,
    private viewport: RisqViewport,
    private order_paths: RisqOrderPaths,
    private drag_rect: () => { min: Point2D; max: Point2D } | undefined,
    private components: CanvasComponent[]
  ) {}

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, max_scale: number) {
    const game = this.session.getGame();
    if (!game) {
      return;
    }
    const now = Date.now();
    const dt = now - this.last_time;
    this.last_time = now;
    this.viewport.setTransform(transform);
    this.viewport.updateDrawDetail(transform.scale, max_scale);
    const hex_r = this.viewport.hexR();
    const inset_h = (2 * this.viewport.hexA()) / Math.sqrt(INSET_RATIO * INSET_RATIO + 1);
    const draw_config: DrawRisqSpaceConfig = {
      hex_r,
      inset_w: INSET_RATIO * inset_h,
      inset_h,
      inset_row: inset_h / 4 - 4,
      draw_detail: this.viewport.drawDetail(),
      view_mode: this.viewport.viewMode(),
      rotation: transform.rotation,
    };
    const bounds = this.viewport.visibleCanvasBounds();
    const on_screen_spaces: RisqSpace[] = [];
    for (const row of game.spaces) {
      for (const space of row) {
        if (!space) {
          continue;
        }
        space.center = this.viewport.coordinateToCanvas(space.coordinate);
        if (!this.viewport.isSpaceOnScreen(space, bounds)) {
          continue;
        }
        on_screen_spaces.push(space);
        drawRisqSpace(ctx, this.risq, space, draw_config);
      }
    }
    // borders are drawn in their own pass after every space's (opaque) fill, so a later space's fill can't paint over an earlier space's border
    for (const space of on_screen_spaces) {
      drawRisqSpaceBorder(ctx, this.risq, space, draw_config);
    }
    drawRisqRegionBorders(ctx, this.risq, on_screen_spaces, hex_r);
    if (this.viewport.viewMode() === RisqViewMode.REGION) {
      drawRisqRegionLabels(ctx, this.risq);
    }
    this.order_paths.draw(ctx);
    this.drawDragRect(ctx, transform);
    for (const component of this.components) {
      component.draw(ctx, transform, dt);
    }
    if (DRAW_CENTER_DOT && DEV) {
      ctx.fillStyle = 'red';
      ctx.strokeStyle = 'transparent';
      drawCircle(ctx, multiplyPoint2D(1 / transform.scale, transform.view), 6 / transform.scale);
    }
  }

  private drawDragRect(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    const rect = this.drag_rect();
    if (!rect) {
      return;
    }
    configDraw(
      ctx,
      transform,
      { fill_style: 'rgba(255, 255, 255, 0.15)', stroke_style: 'white', stroke_width: 1, fixed_position: true },
      false,
      false,
      () => drawRect(ctx, rect.min, rect.max.x - rect.min.x, rect.max.y - rect.min.y)
    );
  }
}
