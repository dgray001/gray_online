import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { defaultTransform, screenToCanvas } from '../../../../util/canvas_board/canvas_board';
import type { CanvasComponent } from '../../../../util/canvas_components/canvas_component';
import { configDraw } from '../../../../util/canvas_components/canvas_component';
import { drawCircle, drawHexagon } from '../../../../util/canvas_util';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D, multiplyPoint2D, rotatePoint } from '../../../../util/objects2d';
import type { RisqSpace, RisqUnit, RisqZone } from '../../model/types';
import { RisqResourceType, RisqVisibilityLevel } from '../../model/types';
import type { DwgRisq } from '../../risq';
import { MINIMAP_BUTTON_OVERFLOW } from './corner_button';
import type { RisqMinimapCornerButton } from './corner_button';
import {
  RisqMercenaryPanelButton,
  RisqDefaultBehaviorButton,
  RisqTechTreeButton,
  RisqViewModeButton,
} from '../bottom_panel/bottom_panel_buttons';
import { drawRisqSpaceBorders } from '../../rendering/space_borders';
import { DrawRisqSpaceDetail, drawHexImage, fillHexOverlay, getSpaceFill } from '../../rendering/space';
import { RisqViewMode, spaceOwnerColor, terrainImage } from '../../rendering/terrain';
import {
  centerUnitMaxZoom,
  centerUnitSlotCapacity,
  edgeUnitSlotCapacity,
  findOuterZoneIndex,
  getRisqZone,
  zoneCenterOffset,
  zoneUnitSlotOffsets,
  zoneVertices,
} from '../../rendering/zones/geometry';
import { buildZoneUnitSlots } from '../../rendering/zones/slots';
import { isForestResource, resourceType } from '../../rendering/assets/resources';

const RESOURCE_DOT_COLORS: Record<RisqResourceType | 'selected', string> = {
  [RisqResourceType.ERROR]: 'black',
  [RisqResourceType.FOOD]: '#a0c692',
  [RisqResourceType.WOOD]: 'rgb(25, 85, 35)',
  [RisqResourceType.STONE]: 'rgb(160, 160, 160)',
  [RisqResourceType.GOLD]: 'rgb(220, 180, 35)',
  selected: 'white',
};

function minimapTerrainImage(terrain_id: number): string {
  switch (terrain_id) {
    case 1:
    case 2:
    case 3:
    case 4:
    case 5:
      return 'risq/terrains/minimap/grass';
    case 6:
    case 7:
    case 13:
      return 'risq/terrains/minimap/cobblestone';
    case 8:
    case 9:
    case 10:
    case 11:
    case 12:
      return 'risq/terrains/minimap/dirt';
    case 14:
    case 22:
    case 23:
      return 'risq/terrains/minimap/straw';
    case 15:
    case 16:
    case 17:
    case 18:
    case 19:
      return 'risq/terrains/minimap/sand';
    case 20:
    case 21:
      return 'risq/terrains/minimap/snow';
    case 24:
    case 25:
      return 'risq/terrains/minimap/farmland';
    case 51:
    case 52:
    case 53:
    case 54:
    case 55:
      return 'risq/terrains/minimap/grass_hills';
    case 56:
    case 57:
    case 58:
    case 59:
    case 60:
      return 'risq/terrains/minimap/dirt_hills';
    case 61:
    case 62:
      return 'risq/terrains/minimap/snow_hills';
    case 63:
    case 64:
      return 'risq/terrains/minimap/farmland_hills';
    case 101:
    case 102:
    case 103:
    case 104:
    case 105:
      return 'risq/terrains/minimap/grass_mountains';
    case 106:
    case 107:
    case 108:
    case 109:
    case 110:
      return 'risq/terrains/minimap/dirt_mountains';
    case 111:
    case 112:
      return 'risq/terrains/minimap/snow_mountains';
    case 151:
    case 152:
    case 153:
      return 'risq/terrains/minimap/swamp';
    case 201:
    case 202:
    case 203:
    case 204:
    case 205:
    case 206:
      return 'risq/terrains/minimap/shallows';
    case 251:
    case 252:
    case 253:
    case 254:
      return 'risq/terrains/minimap/water';
    case 301:
    case 302:
    case 303:
    case 304:
    case 305:
    case 306:
      return 'risq/terrains/minimap/deep_water';
    default:
      return terrainImage(terrain_id);
  }
}

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
  private unit_offsets = new WeakMap<RisqZone, Map<number, Point2D>>();
  private corner_buttons: RisqMinimapCornerButton[];

  constructor(risq: DwgRisq, config: MinimapConfig) {
    this.risq = risq;
    this.config = config;
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

  private resourceDot(type: RisqResourceType | 'selected'): HTMLCanvasElement | undefined {
    const size = 16;
    return this.risq
      .getImageCache()
      .getImage(`minimap_resource_${type}`, size, [], (ctx: CanvasRenderingContext2D): void => {
        ctx.fillStyle = RESOURCE_DOT_COLORS[type];
        ctx.strokeStyle = 'transparent';
        drawCircle(ctx, { x: size / 2, y: size / 2 }, size / 2);
      });
  }

  private drawForestZone(ctx: CanvasRenderingContext2D, zone: RisqZone, center: Point2D): void {
    ctx.save();
    ctx.fillStyle = RESOURCE_DOT_COLORS[RisqResourceType.WOOD];
    ctx.beginPath();
    for (const point of zoneVertices(zone.coordinate, this.hex_r)) {
      ctx.lineTo(center.x + point.x, center.y + point.y);
    }
    ctx.closePath();
    ctx.fill();
    ctx.restore();
  }

  private drawResourceDots(ctx: CanvasRenderingContext2D, space: RisqSpace, selected_points: Point2D[]): void {
    if (space.visibility < RisqVisibilityLevel.FOG) {
      return;
    }
    for (const zone of space.zones?.flat() ?? []) {
      if (!zone.resource) {
        continue;
      }
      const forest = isForestResource(zone.resource);
      const image = forest ? undefined : this.resourceDot(resourceType(zone.resource));
      if (!forest && !image) {
        continue;
      }
      const center = this.coordinateToMinimapCanvas(space.coordinate);
      const point = addPoint2D(center, zoneCenterOffset(zone.coordinate, this.hex_r));
      const scale = this.risq.viewport.viewMode() === RisqViewMode.RESOURCE ? 1.4 : 1;
      const radius = scale * Math.max(1, 0.175 * this.hex_r);
      if (forest) {
        this.drawForestZone(ctx, zone, center);
      } else if (image) {
        ctx.drawImage(image, point.x - radius, point.y - radius, radius * 2, radius * 2);
      }
      if (this.risq.selection.isResourceSelected(zone.resource.internal_id)) {
        selected_points.push(point);
      }
    }
  }

  private zoneUnitOffsets(zone: RisqZone): Map<number, Point2D> {
    const cached = this.unit_offsets.get(zone);
    if (cached) {
      return cached;
    }
    const unit_r = 1 / (2 * centerUnitMaxZoom(1, 1));
    const capacity =
      findOuterZoneIndex(zone.coordinate) === -1 ? centerUnitSlotCapacity(1, unit_r) : edgeUnitSlotCapacity(1, unit_r);
    const slots = buildZoneUnitSlots(zone, this.risq.getPlayerId(), capacity);
    const offsets = zoneUnitSlotOffsets(zone.coordinate, 1, slots.length, unit_r);
    const positions = new Map<number, Point2D>();
    for (const [i, slot] of slots.entries()) {
      for (const group of slot) {
        for (const id of group.units) {
          positions.set(id, offsets[i]);
        }
      }
    }
    this.unit_offsets.set(zone, positions);
    return positions;
  }

  private unitOffset(unit: RisqUnit): Point2D | undefined {
    if (unit.garrisoned_in !== undefined) {
      return undefined;
    }
    const zone = getRisqZone(this.risq.session.spaceAt(unit.space_coordinate), unit.zone_coordinate);
    return zone ? this.zoneUnitOffsets(zone).get(unit.internal_id) : undefined;
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
            if (
              space.visibility === RisqVisibilityLevel.UNEXPLORED ||
              view_mode === RisqViewMode.OWNERSHIP ||
              view_mode === RisqViewMode.REGION
            ) {
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
          },
          (s) => this.coordinateToMinimapCanvas(s.coordinate)
        );
        const selected_points: Point2D[] = [];
        const selected_resource_points: Point2D[] = [];
        const selected_units = this.risq.selection.selectedUnitIds();

        ctx.imageSmoothingEnabled = true;
        if (
          view_mode !== RisqViewMode.MILITARY &&
          view_mode !== RisqViewMode.OWNERSHIP &&
          view_mode !== RisqViewMode.REGION
        ) {
          for (const space of all_spaces) {
            this.drawResourceDots(ctx, space, selected_resource_points);
          }
        }
        ctx.imageSmoothingEnabled = false;
        for (const player of game.players) {
          const color_key = `minimap_dot_${player.color.getR()}_${player.color.getG()}_${player.color.getB()}`;
          const dot_img = this.risq.getImageCache().getImage(color_key, 1, [], (c) => {
            c.fillStyle = `rgb(${player.color.getR()}, ${player.color.getG()}, ${player.color.getB()})`;
            c.fillRect(0, 0, 1, 1);
          });

          for (const entity of [...player.units.values(), ...player.buildings.values()]) {
            const is_unit = 'unit_id' in entity;
            const location = is_unit ? this.risq.session.unitLocation(entity) : entity;
            if (!location) {
              continue;
            }
            const offset =
              (is_unit ? this.unitOffset(entity) : undefined) ?? zoneCenterOffset(location.zone_coordinate, 1);
            const point = addPoint2D(
              this.coordinateToMinimapCanvas(location.space_coordinate),
              multiplyPoint2D(this.hex_r, offset)
            );

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
          const image = this.resourceDot('selected');
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
        const visible_x0 = left_panel.isShowing() ? left_panel.xf() : 0;
        const visible_x1 = right_panel.isOpen() ? right_panel.xi() : this.risq.viewport.canvasSize().width;
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
      button.dataRefreshed();
      button.draw(ctx, transform, dt);
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
    if (!this.hovering || this.corner_buttons.some((button) => button.isHovering()) || e.button !== 0) {
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

  private jumpTo(screen: Point2D): void {
    const origin = this.contentOrigin();
    const content_center = { x: 0.5 * this.content_size.x, y: 0.5 * this.content_size.y };
    const display_center = { x: origin.x + content_center.x, y: origin.y + content_center.y };
    const local = rotatePoint(
      { x: screen.x - display_center.x, y: screen.y - display_center.y },
      -this.last_transform.rotation
    );
    const coordinate = this.minimapCanvasToCoordinate({
      x: local.x + content_center.x,
      y: local.y + content_center.y,
    });
    this.risq.goToCoordinate(coordinate);
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
