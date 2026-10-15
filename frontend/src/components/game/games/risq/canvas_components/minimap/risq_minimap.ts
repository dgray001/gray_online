import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { defaultTransform, screenToCanvas } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawHexagon } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, rotatePoint, roundAxialCoordinate } from '../../../../util/objects2d';
import type { RisqSpace } from '../../model/types';
import { RisqVisibilityLevel } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { RisqMinimapOrders } from './minimap_orders';
import { MINIMAP_BUTTON_OVERFLOW } from './corner_button';
import type { RisqMinimapCornerButton } from './corner_button';
import {
  RisqMercenaryPanelButton,
  RisqDefaultBehaviorButton,
  RisqTechTreeButton,
  RisqViewModeButton,
} from '../bottom_panel/bottom_panel_buttons';
import { RisqLinks } from '../../rendering/links';
import { drawRisqSpaceBorders } from '../../rendering/space_borders';
import { DrawRisqSpaceDetail, drawHexImage, fillHexOverlay, getSpaceFill } from '../../rendering/space';
import {
  RisqViewMode,
  buildingVisibleInViewMode,
  minimapTerrainImage,
  spaceOwnerColor,
  unitVisibleInViewMode,
} from '../../rendering/terrain';
import { clipPolygonToBox, zoneCenterOffset } from '../../rendering/zones/geometry';
import { drawZoneResourceDot, resourceDot } from '../../rendering/overview_dots';

export declare interface MinimapConfig {
  target_w: number;
  background: string;
}

/** Link lines are sized for the main board, so the minimap scales them down to hairlines */
const MINIMAP_LINK_SCALE = 3;

export class RisqMinimap implements CanvasComponent {
  readonly overflow = MINIMAP_BUTTON_OVERFLOW;
  private static PADDING = 4;

  private risq: DwgRisq;
  private config: MinimapConfig;
  private hex_r = 0;
  private content_size: Point2D = { x: 0, y: 0 };
  private side = 0;
  private p: Point2D = { x: 0, y: 0 };
  private hovering = false;
  private clicking = false;
  private last_screen_m: Point2D = { x: 0, y: 0 };
  private last_transform: BoardTransformData = defaultTransform();
  private corner_buttons: RisqMinimapCornerButton[];
  private links: RisqLinks;
  private orders: RisqMinimapOrders;

  constructor(
    risq: DwgRisq,
    config: MinimapConfig,
    private order_space: (space: RisqSpace, ctrl_held: boolean) => void
  ) {
    this.risq = risq;
    this.config = config;
    this.links = new RisqLinks(risq.session, {
      hexR: (): number => this.hex_r,
      hexA: (): number => 0.5 * 1.732 * this.hex_r,
      coordinateToCanvas: (space: Point2D): Point2D => this.coordinateToMinimapCanvas(space),
    });
    this.orders = new RisqMinimapOrders(risq, {
      hexR: (): number => this.hex_r,
      toCanvas: (coordinate: Point2D): Point2D => this.coordinateToMinimapCanvas(coordinate),
      rotation: (): number => this.last_transform.rotation,
    });
    this.corner_buttons = [
      new RisqViewModeButton(risq),
      new RisqTechTreeButton(risq),
      new RisqDefaultBehaviorButton(risq),
      new RisqMercenaryPanelButton(risq),
    ];
  }

  resolveSize(): void {
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
    for (const button of this.corner_buttons) {
      button.setMinimapBounds(this.p, this.side);
    }
  }

  setPosition(p: Point2D): void {
    this.p = p;
    for (const button of this.corner_buttons) {
      button.setMinimapBounds(p, this.side);
    }
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
    if (!hovering) {
      this.corner_buttons.forEach((button) => button.setHovering(false));
    }
  }
  isClicking(): boolean {
    return this.clicking || this.corner_buttons.some((button) => button.isClicking());
  }
  setClicking(clicking: boolean): void {
    this.clicking = clicking;
    if (!clicking) {
      this.corner_buttons.forEach((button) => button.setClicking(false));
    }
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

  private drawResourceDots(ctx: CanvasRenderingContext2D, space: RisqSpace, selected_points: Point2D[]): void {
    if (space.visibility < RisqVisibilityLevel.FOG) {
      return;
    }
    for (const zone of space.zones?.flat() ?? []) {
      if (!zone.resource) {
        continue;
      }
      const center = this.coordinateToMinimapCanvas(space.coordinate);
      const point = addPoint2D(center, zoneCenterOffset(zone.coordinate, this.hex_r));
      const scale = this.risq.viewport.viewMode() === RisqViewMode.RESOURCE ? 1.4 : 1;
      drawZoneResourceDot(ctx, this.risq.getImageCache(), zone, center, this.hex_r, scale);
      if (this.risq.selection.isResourceSelected(zone.resource.internal_id)) {
        selected_points.push(point);
      }
    }
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    const game = this.risq.getGame();
    if (!game || this.hex_r <= 0) {
      return;
    }
    this.last_transform = transform;
    configDraw(
      ctx,
      transform,
      {
        fill_style: this.config.background,
        stroke_style: 'transparent',
        stroke_width: 0,
        fixed_position: true,
      },
      false,
      false,
      () => {
        drawCircle(ctx, { x: this.xc(), y: this.yc() }, 0.5 * this.side);
        ctx.save();
        ctx.beginPath();
        ctx.arc(this.xc(), this.yc(), 0.5 * this.side, 0, 2 * Math.PI);
        ctx.clip();
        const origin = this.contentOrigin();
        const content_center = { x: 0.5 * this.content_size.x, y: 0.5 * this.content_size.y };
        ctx.translate(origin.x + content_center.x, origin.y + content_center.y);
        ctx.rotate(transform.rotation);
        ctx.translate(-content_center.x, -content_center.y);
        const view_mode = this.risq.viewport.viewMode();
        const draw_r = this.hex_r + 1; // slight overlap so adjacent tiles' antialiasing doesn't leave seams
        ctx.strokeStyle = 'transparent';
        ctx.lineWidth = 0;
        for (const row of game.spaces) {
          for (const space of row) {
            if (!space) {
              continue;
            }
            const minimap_canvas = this.coordinateToMinimapCanvas(space.coordinate);
            const owner_color = spaceOwnerColor(space.ownership, game.players);
            const region_owned = (this.risq.session.getRegionForSpace(space.coordinate_key)?.owner ?? -1) >= 0;
            if (space.visibility === RisqVisibilityLevel.UNEXPLORED || view_mode === RisqViewMode.OWNERSHIP) {
              const fill = getSpaceFill(space, RisqViewMode.OWNERSHIP, owner_color, false);
              if (region_owned) {
                fill.dBrightness(-0.22);
              }
              ctx.fillStyle = `rgb(${fill.getR()}, ${fill.getG()}, ${fill.getB()})`;
              drawHexagon(ctx, minimap_canvas, draw_r);
            } else {
              drawHexImage(ctx, this.risq.getIcon(minimapTerrainImage(space.terrain_id)), minimap_canvas, draw_r);
              if (view_mode !== RisqViewMode.RESOURCE && !!owner_color) {
                fillHexOverlay(
                  ctx,
                  minimap_canvas,
                  this.hex_r,
                  `rgba(${owner_color.getR()}, ${owner_color.getG()}, ${owner_color.getB()}, ${region_owned ? 0.45 : 0.25})`
                );
              }
            }
          }
        }
        const all_spaces = game.spaces.flat().filter((s): s is RisqSpace => !!s);
        drawRisqSpaceBorders(
          ctx,
          this.risq,
          all_spaces,
          {
            hex_r: this.hex_r,
            draw_detail: DrawRisqSpaceDetail.OWNERSHIP,
            view_mode,
            regions_only: true,
            black_regions: false,
            min_width: 1,
            border_scale: 1,
          },
          (s) => this.coordinateToMinimapCanvas(s.coordinate)
        );
        this.links.draw(ctx, MINIMAP_LINK_SCALE);
        const selected_points: Point2D[] = [];
        const selected_resource_points: Point2D[] = [];
        const selected_units = this.risq.selection.selectedUnitIds();

        ctx.imageSmoothingEnabled = true;
        if (view_mode !== RisqViewMode.MILITARY && view_mode !== RisqViewMode.OWNERSHIP) {
          for (const space of all_spaces) {
            this.drawResourceDots(ctx, space, selected_resource_points);
          }
        }
        this.orders.drawSelectedOrders(ctx);
        ctx.imageSmoothingEnabled = false;
        for (const player of game.players) {
          const color_key = `minimap_dot_${player.color.getR()}_${player.color.getG()}_${player.color.getB()}`;
          const dot_img = this.risq.getImageCache().getImage(color_key, 1, [], (c) => {
            c.fillStyle = `rgb(${player.color.getR()}, ${player.color.getG()}, ${player.color.getB()})`;
            c.fillRect(0, 0, 1, 1);
          });

          for (const entity of [...player.units.values(), ...player.buildings.values()]) {
            const is_unit = 'unit_id' in entity;
            const point = this.orders.entityPoint(entity);
            const shown = is_unit
              ? unitVisibleInViewMode(entity.unit_type, view_mode)
              : buildingVisibleInViewMode(entity, view_mode);
            if (!point || !shown) {
              continue;
            }

            const radius = Math.max(1, 0.175 * this.hex_r);
            if (dot_img) {
              ctx.drawImage(dot_img, point.x - radius, point.y - radius, radius * 2, radius * 2);
            }

            if (
              is_unit
                ? selected_units.has(entity.internal_id)
                : this.risq.selection.isBuildingSelected(entity.internal_id)
            ) {
              selected_points.push(point);
            }
          }
        }

        const white_dot = this.risq.getImageCache().getImage('minimap_dot_white', 1, [], (c) => {
          c.fillStyle = 'white';
          c.fillRect(0, 0, 1, 1);
        });

        for (const point of selected_points) {
          const radius = Math.max(1.5, 0.25 * this.hex_r);
          if (white_dot) {
            ctx.drawImage(white_dot, point.x - radius, point.y - radius, radius * 2, radius * 2);
          }
        }
        ctx.imageSmoothingEnabled = true;
        for (const point of selected_resource_points) {
          const image = resourceDot(this.risq.getImageCache(), 'selected');
          const scale = view_mode === RisqViewMode.RESOURCE ? 1.4 : 1;
          const radius = scale * Math.max(1.5, 0.25 * this.hex_r);
          if (image) {
            ctx.drawImage(image, point.x - radius, point.y - radius, radius * 2, radius * 2);
          }
        }
        ctx.translate(content_center.x, content_center.y);
        ctx.rotate(-transform.rotation);
        ctx.translate(-(origin.x + content_center.x), -(origin.y + content_center.y));
        const left_panel = this.risq.left_panel;
        const right_panel = this.risq.right_panel;
        const hud = this.risq.hudScale();
        const visible_x0 = left_panel.isShowing() ? left_panel.xf() * hud : 0;
        const visible_x1 = right_panel.isOpen() ? right_panel.xi() * hud : this.risq.viewport.canvasSize().width;
        const canvas_h = this.risq.viewport.canvasSize().height;
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
        ctx.restore();
      }
    );
    for (const button of this.corner_buttons) {
      button.draw(ctx, transform, dt);
    }
  }

  dataRefreshed(): void {
    for (const button of this.corner_buttons) {
      button.dataRefreshed();
    }
  }

  drawTooltip(ctx: CanvasRenderingContext2D, transform: BoardTransformData, risq: DwgRisq, dt: number): void {
    for (const button of this.corner_buttons) {
      button.drawTooltip(ctx, transform, risq, dt);
    }
  }

  private minimapCanvasFromCanvas(canvas: Point2D): Point2D {
    const coordinate = this.risq.viewport.canvasToCoordinate(canvas, this.risq.getGame()?.board_size ?? 0);
    return this.coordinateToMinimapCanvas(coordinate);
  }

  scroll(): boolean {
    return false;
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.last_screen_m = screen;
    const on_map = Math.hypot(screen.x - this.xc(), screen.y - this.yc()) <= 0.5 * this.side;
    const on_button = this.corner_buttons.map((button) => button.mousemove(canvas, screen, transform)).some(Boolean);
    this.hovering = on_map || on_button;
    if (this.clicking && on_map) {
      this.jumpTo(this.last_screen_m);
    }
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.corner_buttons.map((button) => button.mousedown(e)).some(Boolean)) {
      return true;
    }
    if (!this.hovering || this.corner_buttons.some((button) => button.isHovering())) {
      return false;
    }
    if (e.button === 2) {
      const coordinate = roundAxialCoordinate(this.coordinateAtScreen(this.last_screen_m));
      const space = this.risq.session.spaceAt(coordinate);
      if (space) {
        this.order_space(space, e.ctrlKey);
      }
      return true;
    }
    if (e.button !== 0) {
      return false;
    }
    this.clicking = true;
    this.jumpTo(this.last_screen_m);
    return true;
  }

  mouseup(e: MouseEvent): void {
    this.corner_buttons.forEach((button) => button.mouseup(e));
    this.clicking = false;
  }

  private coordinateAtScreen(screen: Point2D): Point2D {
    const origin = this.contentOrigin();
    const content_center = { x: 0.5 * this.content_size.x, y: 0.5 * this.content_size.y };
    const display_center = { x: origin.x + content_center.x, y: origin.y + content_center.y };
    const local = rotatePoint(
      { x: screen.x - display_center.x, y: screen.y - display_center.y },
      -this.last_transform.rotation
    );
    return this.minimapCanvasToCoordinate({
      x: local.x + content_center.x,
      y: local.y + content_center.y,
    });
  }

  private jumpTo(screen: Point2D): void {
    this.risq.goToCoordinate(this.coordinateAtScreen(screen));
  }

  xi(): number {
    return this.p.x;
  }
  yi(): number {
    return this.p.y;
  }
  xf(): number {
    return this.xi() + this.w();
  }
  yf(): number {
    return this.yi() + this.h();
  }
  xc(): number {
    return this.xi() + 0.5 * this.side;
  }
  yc(): number {
    return this.yi() + 0.5 * this.side;
  }
  w(): number {
    return this.side;
  }
  h(): number {
    return this.side;
  }
}
