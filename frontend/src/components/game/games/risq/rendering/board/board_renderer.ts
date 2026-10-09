import { isImageReady } from '../../../../../../scripts/image';
import { DEV } from '../../../../../../scripts/util';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { multiplyPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { GameRisq, RisqSpace } from '../../model/types';
import type { RisqDrawHost } from '../draw_host';
import { drawRisqRegionLabels } from '../region';
import type { DrawRisqSpaceConfig } from '../space';
import { drawRisqSpaceBase, drawRisqSpaceContent } from '../space';
import { drawRisqSpaceBorders } from '../space_borders';
import { RisqViewMode } from '../terrain';
import type { RisqOrderOverlays } from './order_overlays';
import type { RisqViewport } from './viewport';

const DRAW_CENTER_DOT = false;

// the inset rect must stay inside the hex's incircle (radius = hex_a) so it never pokes out as the map rotates
const INSET_RATIO = 1.0392;

export class RisqBoardRenderer {
  private last_time = Date.now();

  constructor(
    private risq: RisqDrawHost,
    private session: RisqSession,
    private viewport: RisqViewport,
    private order_overlays: Pick<RisqOrderOverlays, 'draw'>,
    private drag_rect: () => { min: Point2D; max: Point2D } | undefined,
    private components: CanvasComponent[]
  ) {}

  private drawBackgroundImage(ctx: CanvasRenderingContext2D, game: GameRisq): void {
    if (!game.background_image || !game.background_top_left || !game.background_top_right) {
      return;
    }

    const image = this.risq.getIcon(`risq/maps/${game.background_image}`);
    if (!isImageReady(image)) {
      return;
    }

    const top_left_coordinate = { x: game.background_top_left[0], y: game.background_top_left[1] };
    const top_right_coordinate = { x: game.background_top_right[0], y: game.background_top_right[1] };

    const top_left = this.viewport.coordinateToCanvas(top_left_coordinate);
    const top_right = this.viewport.coordinateToCanvas(top_right_coordinate);

    const width = Math.hypot(top_right.x - top_left.x, top_right.y - top_left.y);
    const height = width * (image.naturalHeight / image.naturalWidth);
    const angle = Math.atan2(top_right.y - top_left.y, top_right.x - top_left.x);

    ctx.save();
    ctx.translate(top_left.x, top_left.y);
    ctx.rotate(angle);
    ctx.drawImage(image, 0, 0, width, height);
    ctx.restore();
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, max_scale: number): void {
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
    this.drawBackgroundImage(ctx, game);
    const on_screen_spaces: RisqSpace[] = [];
    const space_text_colors: boolean[] = [];
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
        space_text_colors.push(drawRisqSpaceBase(ctx, this.risq, space, draw_config));
      }
    }
    drawRisqSpaceBorders(ctx, this.risq, on_screen_spaces, draw_config);
    for (const [i, space] of on_screen_spaces.entries()) {
      drawRisqSpaceContent(ctx, this.risq, space, draw_config, space_text_colors[i]);
    }
    if (this.viewport.viewMode() === RisqViewMode.REGION) {
      drawRisqRegionLabels(ctx, this.risq);
    }
    this.order_overlays.draw(ctx);
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

  private drawDragRect(ctx: CanvasRenderingContext2D, transform: BoardTransformData): void {
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
