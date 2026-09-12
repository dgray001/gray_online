import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { canvasToScreen } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawHexagon, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { RisqVisibilityLevel } from '../../risq_data';
import type { DwgRisq } from '../../risq';
import { drawHexImage, fillHexOverlay, getSpaceFill } from '../../risq_space';
import { RisqViewMode, spaceOwnerColor, terrainImage } from '../../risq_terrain';

export declare interface MinimapConfig {
  target_w: number;
  background: string;
}

export class RisqMinimap implements CanvasComponent {
  private static PADDING = 4;

  private risq: DwgRisq;
  private config: MinimapConfig;
  private hex_r = 0;
  private content_size: Point2D = { x: 0, y: 0 };
  private hovering = false;
  private clicking = false;
  private last_screen_m: Point2D = { x: 0, y: 0 };

  constructor(risq: DwgRisq, config: MinimapConfig) {
    this.risq = risq;
    this.config = config;
  }

  resolveSize() {
    const board_size = this.risq.getGame()?.board_size;
    if (board_size === undefined) {
      return;
    }
    this.hex_r = this.config.target_w / (1.732 * (2 * board_size + 1));
    this.content_size = {
      x: 1.732 * this.hex_r * (2 * board_size + 1),
      y: 1.5 * this.hex_r * (2 * board_size + 1) + 0.5 * this.hex_r,
    };
  }

  isHovering(): boolean {
    return this.hovering;
  }
  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }
  isClicking(): boolean {
    return this.clicking;
  }
  setClicking(clicking: boolean): void {
    this.clicking = clicking;
  }

  private minimapCanvasToCoordinate(minimap_canvas: Point2D): Point2D {
    const board_size = this.risq.getGame()?.board_size ?? 0;
    const cy = (minimap_canvas.y - 0.25 * this.hex_r) / (1.5 * this.hex_r) - board_size - 0.5;
    return {
      x: minimap_canvas.x / (1.732 * this.hex_r) - 0.5 * cy - board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToMinimapCanvas(coordinate: Point2D): Point2D {
    const board_size = this.risq.getGame()?.board_size ?? 0;
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + board_size + 0.5) * this.hex_r,
      y: 1.5 * (coordinate.y + board_size + 0.5) * this.hex_r + 0.25 * this.hex_r,
    };
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, _dt: number): void {
    const game = this.risq.getGame();
    if (!game || this.hex_r <= 0) {
      return;
    }
    configDraw(
      ctx,
      transform,
      {
        fill_style: this.config.background,
        stroke_style: 'rgb(70, 30, 5)',
        stroke_width: 2,
        fixed_position: true,
      },
      false,
      false,
      () => {
        drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
        ctx.translate(this.xi() + RisqMinimap.PADDING, this.yi() + RisqMinimap.PADDING);
        const view_mode = this.risq.viewMode();
        const draw_r = this.hex_r + 1; // slight overlap so adjacent tiles' antialiasing doesn't leave seams
        ctx.strokeStyle = 'transparent';
        ctx.lineWidth = 0;
        for (const row of game.spaces) {
          for (const space of row) {
            const minimap_canvas = this.coordinateToMinimapCanvas(space.coordinate);
            const owner_color = spaceOwnerColor(space, game.players);
            if (space.visibility === RisqVisibilityLevel.UNEXPLORED || view_mode === RisqViewMode.OWNERSHIP) {
              const fill = getSpaceFill(space, RisqViewMode.OWNERSHIP, owner_color, false);
              ctx.fillStyle = `rgb(${fill.getR()}, ${fill.getG()}, ${fill.getB()})`;
              drawHexagon(ctx, minimap_canvas, draw_r);
            } else {
              drawHexImage(ctx, this.risq.getIcon(terrainImage(space.terrain)), minimap_canvas, draw_r);
              if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
                fillHexOverlay(
                  ctx,
                  minimap_canvas,
                  this.hex_r,
                  `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, 0.25)`
                );
              }
            }
          }
        }
        ctx.fillStyle = 'white';
        ctx.strokeStyle = 'transparent';
        for (const player of game.players) {
          for (const unit of player.units.values()) {
            if (this.risq.isUnitSelected(unit.internal_id)) {
              drawCircle(ctx, this.coordinateToMinimapCanvas(unit.space_coordinate), Math.max(1.5, 0.25 * this.hex_r));
            }
          }
        }
        const left_panel = this.risq.getLeftPanel();
        const right_panel = this.risq.getRightPanel();
        const visible_x0 = left_panel.isShowing() ? left_panel.xf() : 0;
        const visible_x1 = right_panel.isOpen() ? right_panel.xi() : this.risq.canvasSize().width;
        const offset = this.risq.canvasCenter();
        const viewport_min = this.minimapCanvasFromCanvas({
          x: (transform.view.x + visible_x0 - offset.x) / transform.scale,
          y: (transform.view.y - offset.y) / transform.scale,
        });
        const viewport_max = this.minimapCanvasFromCanvas({
          x: (transform.view.x + visible_x1 - offset.x) / transform.scale,
          y: (transform.view.y + this.risq.canvasSize().height - offset.y) / transform.scale,
        });
        const clamp = (v: number, max: number) => Math.max(0, Math.min(max, v));
        const clamped_min = { x: clamp(viewport_min.x, this.content_size.x), y: clamp(viewport_min.y, this.content_size.y) };
        const clamped_max = { x: clamp(viewport_max.x, this.content_size.x), y: clamp(viewport_max.y, this.content_size.y) };
        ctx.fillStyle = 'transparent';
        ctx.strokeStyle = 'white';
        ctx.lineWidth = 1;
        drawRect(ctx, clamped_min, clamped_max.x - clamped_min.x, clamped_max.y - clamped_min.y);
        ctx.translate(-(this.xi() + RisqMinimap.PADDING), -(this.yi() + RisqMinimap.PADDING));
      }
    );
  }

  private minimapCanvasFromCanvas(canvas: Point2D): Point2D {
    const coordinate = this.risq.canvasToCoordinate(canvas, this.risq.getGame()?.board_size ?? 0);
    return this.coordinateToMinimapCanvas(coordinate);
  }

  scroll(): boolean {
    return false;
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    this.last_screen_m = canvasToScreen(m, transform);
    this.hovering =
      this.last_screen_m.x >= this.xi() &&
      this.last_screen_m.y >= this.yi() &&
      this.last_screen_m.x <= this.xf() &&
      this.last_screen_m.y <= this.yf();
    if (this.clicking && this.hovering) {
      this.jumpTo(this.last_screen_m);
    }
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (!this.hovering || e.button !== 0) {
      return false;
    }
    this.clicking = true;
    this.jumpTo(this.last_screen_m);
    return true;
  }

  mouseup(): void {
    this.clicking = false;
  }

  private jumpTo(screen: Point2D) {
    const coordinate = this.minimapCanvasToCoordinate({
      x: screen.x - this.xi() - RisqMinimap.PADDING,
      y: screen.y - this.yi() - RisqMinimap.PADDING,
    });
    this.risq.goToCoordinate(coordinate);
  }

  xi(): number {
    return 0.5 * (this.risq.canvasSize().width - this.w());
  }
  yi(): number {
    return this.risq.canvasSize().height - this.h() - 12;
  }
  xf(): number {
    return this.xi() + this.w();
  }
  yf(): number {
    return this.yi() + this.h();
  }
  w(): number {
    return this.content_size.x + 2 * RisqMinimap.PADDING;
  }
  h(): number {
    return this.content_size.y + 2 * RisqMinimap.PADDING;
  }
}
