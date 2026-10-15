import { isImageReady } from '../../../../../../scripts/image';
import { DEV } from '../../../../../../scripts/util';
import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { multiplyPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import type { GameRisq, RisqPlayer, RisqSpace } from '../../model/types';
import type { ColorRGB } from '../../../../../../scripts/color_rgb';
import type { RisqArmedState } from '../../application/input/armed_state';
import type { RisqHover } from '../../application/input/hover';
import type { RisqOrderPlanning } from '../../application/orders/planning';
import type { RisqSelection } from '../../application/selection/selection';
import type { RisqControlGroups } from '../../application/selection/control_groups';
import type { RisqImageCache } from '../assets/image_cache';
import type { RisqCorpseLayout } from '../zones/corpses';
import { drawRisqRegionLabels } from '../region';
import { RisqLinks } from '../links';
import type { DrawRisqSpaceConfig } from '../space';
import { drawRisqSpaceBase, drawRisqSpaceContent } from '../space';
import { drawRisqSpaceBorders } from '../space_borders';
import { RisqViewMode } from '../terrain';
import type { RisqOrderOverlays } from './order_overlays';
import type { RisqViewport } from './viewport';

const DRAW_CENTER_DOT = false;

// the inset rect must stay inside the hex's incircle (radius = hex_a) so it never pokes out as the map rotates
const INSET_RATIO = 1.0392;

export interface RisqDrawHost {
  readonly control_groups?: Pick<RisqControlGroups, 'memberships'>;
  readonly session: Pick<RisqSession, 'getRegionForSpace' | 'getRegionLookup' | 'getResourceConfigs' | 'isReplaying'>;
  readonly planning: Pick<RisqOrderPlanning, 'getLocalFoundation' | 'hasPlannedFoundation' | 'idleIds'>;
  readonly selection: Pick<
    RisqSelection,
    'isUnitSelected' | 'isBuildingSelected' | 'isResourceSelected' | 'isRegionSelected'
  >;
  readonly armed: Pick<RisqArmedState, 'getArmedOrder' | 'getArmedBuilding'>;
  readonly hover: Pick<RisqHover, 'hoveredRegion' | 'space'>;
  readonly viewport: Pick<
    RisqViewport,
    | 'regionLabelPosition'
    | 'coordinateToCanvas'
    | 'lastTransform'
    | 'unitRadius'
    | 'corpseRadius'
    | 'centerMaxSlots'
    | 'edgeMaxSlots'
  >;
  readonly corpse_layout: RisqCorpseLayout;
  getIcon(name: string): HTMLImageElement;
  getPlayerColoredIcon(name: string, color: ColorRGB): HTMLImageElement | HTMLCanvasElement;
  getImageCache(): RisqImageCache;
  getGame(): GameRisq | undefined;
  getPlayer(): RisqPlayer | undefined;
  getPlayerId(): number;
}

export class RisqBoardRenderer {
  private last_time = Date.now();
  private links: RisqLinks;

  constructor(
    private risq: RisqDrawHost,
    private session: RisqSession,
    private viewport: RisqViewport,
    private order_overlays: Pick<RisqOrderOverlays, 'draw' | 'drawRanges'>,
    private drag_rect: () => { min: Point2D; max: Point2D } | undefined,
    private components: CanvasComponent[],
    private hud_scale: () => number = (): number => 1
  ) {
    this.links = new RisqLinks(session, viewport);
  }

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
    const inset_gap = 2 / transform.scale;
    const draw_config: DrawRisqSpaceConfig = {
      hex_r,
      inset_w: INSET_RATIO * inset_h,
      inset_h,
      inset_row: (inset_h - 3 * inset_gap) / 4,
      inset_gap,
      draw_detail: this.viewport.drawDetail(),
      view_mode: this.viewport.viewMode(),
      rotation: transform.rotation,
      entity_pass: 'terrain',
      zone_borders_only: false,
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
    const zone_border_config: DrawRisqSpaceConfig = { ...draw_config, zone_borders_only: true };
    for (const [i, space] of on_screen_spaces.entries()) {
      drawRisqSpaceContent(ctx, this.risq, space, zone_border_config, space_text_colors[i]);
    }
    drawRisqSpaceBorders(ctx, this.risq, on_screen_spaces, draw_config);
    this.links.draw(ctx, transform.scale);
    for (const [i, space] of on_screen_spaces.entries()) {
      drawRisqSpaceContent(ctx, this.risq, space, draw_config, space_text_colors[i]);
    }
    this.order_overlays.drawRanges(ctx);
    for (const selected of [false, true]) {
      this.order_overlays.draw(ctx, selected);
      const pass_config: DrawRisqSpaceConfig = { ...draw_config, entity_pass: selected ? 'selected' : 'unselected' };
      for (const [i, space] of on_screen_spaces.entries()) {
        drawRisqSpaceContent(ctx, this.risq, space, pass_config, space_text_colors[i]);
      }
    }
    if (this.viewport.viewMode() === RisqViewMode.OWNERSHIP && !this.viewport.zoneView()) {
      drawRisqRegionLabels(ctx, this.risq, draw_config);
    }
    this.drawDragRect(ctx, transform);
    this.drawHud(ctx, transform, dt);
    if (DRAW_CENTER_DOT && DEV) {
      ctx.fillStyle = 'red';
      ctx.strokeStyle = 'transparent';
      drawCircle(ctx, multiplyPoint2D(1 / transform.scale, transform.view), 6 / transform.scale);
    }
  }

  /** Components cancel the board matrix themselves, so scaling it first scales the whole HUD uniformly */
  private drawHud(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    const board = ctx.getTransform();
    const scale = this.hud_scale();
    ctx.save();
    ctx.setTransform(scale, 0, 0, scale, 0, 0);
    ctx.transform(board.a, board.b, board.c, board.d, board.e, board.f);
    for (const component of this.components) {
      component.draw(ctx, transform, dt);
    }
    ctx.restore();
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
