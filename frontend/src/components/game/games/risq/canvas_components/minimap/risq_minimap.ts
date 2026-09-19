import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { screenToCanvas } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawHexagon, drawRect } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { rotatePoint } from '../../../../util/objects2d';
import { RisqVisibilityLevel } from '../../risq_data';
import type { DwgRisq } from '../../risq';
import { drawHexImage, fillHexOverlay, getSpaceFill } from '../../risq_space';
import { RisqViewMode, spaceOwnerColor, terrainImage } from '../../risq_terrain';

export declare interface MinimapConfig {
  target_w: number;
  background: string;
}

/** Clips a convex polygon against an axis-aligned box (Sutherland-Hodgman) */
function clipPolygonToBox(polygon: Point2D[], min: Point2D, max: Point2D): Point2D[] {
  const lerp = (a: Point2D, b: Point2D, t: number): Point2D => ({ x: a.x + t * (b.x - a.x), y: a.y + t * (b.y - a.y) });
  const clip_edge = (
    points: Point2D[],
    inside: (p: Point2D) => boolean,
    intersect: (a: Point2D, b: Point2D) => Point2D
  ): Point2D[] => {
    const out: Point2D[] = [];
    for (let i = 0; i < points.length; i++) {
      const curr = points[i];
      const prev = points[(i + points.length - 1) % points.length];
      if (inside(curr)) {
        if (!inside(prev)) {
          out.push(intersect(prev, curr));
        }
        out.push(curr);
      } else if (inside(prev)) {
        out.push(intersect(prev, curr));
      }
    }
    return out;
  };
  let result = polygon;
  result = clip_edge(
    result,
    (p) => p.x >= min.x,
    (a, b) => lerp(a, b, (min.x - a.x) / (b.x - a.x))
  );
  result = clip_edge(
    result,
    (p) => p.x <= max.x,
    (a, b) => lerp(a, b, (max.x - a.x) / (b.x - a.x))
  );
  result = clip_edge(
    result,
    (p) => p.y >= min.y,
    (a, b) => lerp(a, b, (min.y - a.y) / (b.y - a.y))
  );
  result = clip_edge(
    result,
    (p) => p.y <= max.y,
    (a, b) => lerp(a, b, (max.y - a.y) / (b.y - a.y))
  );
  return result;
}

export class RisqMinimap implements CanvasComponent {
  private static PADDING = 4;

  private risq: DwgRisq;
  private config: MinimapConfig;
  private hex_r = 0;
  private content_size: Point2D = { x: 0, y: 0 };
  private side = 0;
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
    this.side = Math.max(this.content_size.x, this.content_size.y) + 2 * RisqMinimap.PADDING;
  }

  private contentOrigin(): Point2D {
    return {
      x: this.xi() + 0.5 * (this.side - this.content_size.x),
      y: this.yi() + 0.5 * (this.side - this.content_size.y),
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
        const origin = this.contentOrigin();
        const content_center = { x: 0.5 * this.content_size.x, y: 0.5 * this.content_size.y };
        ctx.translate(origin.x + content_center.x, origin.y + content_center.y);
        ctx.rotate(transform.rotation);
        ctx.translate(-content_center.x, -content_center.y);
        const view_mode = this.risq.viewMode();
        const draw_r = this.hex_r + 1; // slight overlap so adjacent tiles' antialiasing doesn't leave seams
        ctx.strokeStyle = 'transparent';
        ctx.lineWidth = 0;
        for (const row of game.spaces) {
          for (const space of row) {
            const minimap_canvas = this.coordinateToMinimapCanvas(space.coordinate);
            const owner_color = spaceOwnerColor(space.ownership, game.players);
            if (space.visibility === RisqVisibilityLevel.UNEXPLORED || view_mode === RisqViewMode.OWNERSHIP) {
              const fill = getSpaceFill(space, RisqViewMode.OWNERSHIP, owner_color, false);
              ctx.fillStyle = `rgb(${fill.getR()}, ${fill.getG()}, ${fill.getB()})`;
              drawHexagon(ctx, minimap_canvas, draw_r);
            } else {
              drawHexImage(ctx, this.risq.getIcon(terrainImage(space.terrain_id)), minimap_canvas, draw_r);
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
            if (unit.garrisoned_in === undefined && this.risq.isUnitSelected(unit.internal_id)) {
              drawCircle(ctx, this.coordinateToMinimapCanvas(unit.space_coordinate), Math.max(1.5, 0.25 * this.hex_r));
            }
          }
        }
        ctx.translate(content_center.x, content_center.y);
        ctx.rotate(-transform.rotation);
        ctx.translate(-(origin.x + content_center.x), -(origin.y + content_center.y));
        const left_panel = this.risq.getLeftPanel();
        const right_panel = this.risq.getRightPanel();
        const visible_x0 = left_panel.isShowing() ? left_panel.xf() : 0;
        const visible_x1 = right_panel.isOpen() ? right_panel.xi() : this.risq.canvasSize().width;
        const canvas_h = this.risq.canvasSize().height;
        const display_center = { x: origin.x + content_center.x, y: origin.y + content_center.y };
        const corners = [
          { x: visible_x0, y: 0 },
          { x: visible_x1, y: 0 },
          { x: visible_x1, y: canvas_h },
          { x: visible_x0, y: canvas_h },
        ].map((screen) => {
          const local = this.minimapCanvasFromCanvas(screenToCanvas(screen, transform));
          const rotated = rotatePoint(
            { x: local.x - content_center.x, y: local.y - content_center.y },
            transform.rotation
          );
          return { x: display_center.x + rotated.x, y: display_center.y + rotated.y };
        });
        const clipped = clipPolygonToBox(corners, { x: this.xi(), y: this.yi() }, { x: this.xf(), y: this.yf() });
        if (clipped.length >= 3) {
          ctx.fillStyle = 'transparent';
          ctx.strokeStyle = 'white';
          ctx.lineWidth = 1;
          ctx.beginPath();
          clipped.forEach((p, i) => (i === 0 ? ctx.moveTo(p.x, p.y) : ctx.lineTo(p.x, p.y)));
          ctx.closePath();
          ctx.stroke();
        }
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

  mousemove(_canvas: Point2D, screen: Point2D, _transform: BoardTransformData): boolean {
    this.last_screen_m = screen;
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
    const origin = this.contentOrigin();
    const coordinate = this.minimapCanvasToCoordinate({
      x: screen.x - origin.x,
      y: screen.y - origin.y,
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
    return this.side;
  }
  h(): number {
    return this.side;
  }
}
