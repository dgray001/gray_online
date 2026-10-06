import type { BoardTransformData } from '../../../../util/canvas_board/canvas_board';
import { defaultTransform, screenToCanvas } from '../../../../util/canvas_board/canvas_board';
import type { Point2D } from '../../../../util/objects2d';
import { addPoint2D } from '../../../../util/objects2d';
import type { RisqSession } from '../../application/session';
import { invertPair } from '../../model/coordinates';
import type { RisqRegion, RisqSpace, RisqUnit } from '../../model/types';
import { DrawRisqSpaceDetail } from '../space';
import { RisqViewMode, nextViewMode } from '../terrain';
import { getRisqZone, zoneCenterOffset } from '../zones/geometry';
import { unitSlotWorldPosition } from '../zones/slots';

const DEFAULT_HEXAGON_RADIUS = 60;
const ZOOM_REFERENCE_BOARD_SIZE = 4;
const MAX_ZOOM_CANVAS_FRACTION = 0.65;
const MIN_ZOOM_REFERENCE_POWER = 2.2;
const ZONE_DETAIL_THRESHOLD = 0.45;
const OWNERSHIP_DETAIL_THRESHOLD = 0.4;

export declare interface CanvasBounds {
  min: Point2D;
  max: Point2D;
}

/** Owns the board's hex size, canvas size, last transform, and level of detail, and converts between their frames */
export class RisqViewport {
  private hex_r = DEFAULT_HEXAGON_RADIUS;
  private hex_a = 0.5 * 1.732 * DEFAULT_HEXAGON_RADIUS;
  private canvas_center: Point2D = { x: 0, y: 0 };
  private canvas_size: DOMRect = DOMRect.fromRect();
  private transform: BoardTransformData = defaultTransform();
  private draw_detail: DrawRisqSpaceDetail = DrawRisqSpaceDetail.SPACE_DETAILS;
  private view_mode: RisqViewMode = RisqViewMode.ALL;

  constructor(
    private session: RisqSession,
    private on_view_mode_change?: () => void
  ) {}

  hexR(): number {
    return this.hex_r;
  }

  hexA(): number {
    return this.hex_a;
  }

  canvasSize(): DOMRect {
    return this.canvas_size;
  }

  lastTransform(): BoardTransformData {
    return this.transform;
  }

  setTransform(transform: BoardTransformData) {
    this.transform = transform;
  }

  drawDetail(): DrawRisqSpaceDetail {
    return this.draw_detail;
  }

  zoneView(): boolean {
    return this.draw_detail === DrawRisqSpaceDetail.ZONE_DETAILS;
  }

  viewMode(): RisqViewMode {
    return this.view_mode;
  }

  setViewMode(mode: RisqViewMode): void {
    this.view_mode = mode;
    this.on_view_mode_change?.();
  }

  cycleViewMode(): void {
    this.setViewMode(nextViewMode(this.view_mode));
  }

  /** Whether units, buildings, and their orders are drawn at the current detail and view mode */
  contentDrawn(): boolean {
    return this.view_mode !== RisqViewMode.REGION && this.draw_detail !== DrawRisqSpaceDetail.OWNERSHIP;
  }

  // Regions are hoverable/selectable in the dedicated REGION view mode (any zoom), or fully zoomed out in any other non-military view mode
  regionInteractionEnabled(): boolean {
    return (
      this.view_mode === RisqViewMode.REGION ||
      (this.draw_detail === DrawRisqSpaceDetail.OWNERSHIP && this.view_mode !== RisqViewMode.MILITARY)
    );
  }

  zoomLimits(board_width: number): { min: number; max: number } {
    const reference_radius = board_width / (1.732 * (2 * ZOOM_REFERENCE_BOARD_SIZE + 1));
    const reference_max = (MAX_ZOOM_CANVAS_FRACTION * this.canvas_size.height) / reference_radius;
    const max = (MAX_ZOOM_CANVAS_FRACTION * this.canvas_size.height) / this.hex_r;
    return { min: max / reference_max ** MIN_ZOOM_REFERENCE_POWER, max };
  }

  updateDrawDetail(scale: number, max_scale: number) {
    const board_size = this.session.getGame()?.board_size ?? ZOOM_REFERENCE_BOARD_SIZE;
    const reference_scale = (2 * ZOOM_REFERENCE_BOARD_SIZE + 1) / (2 * board_size + 1);
    scale *= reference_scale;
    max_scale *= reference_scale;
    if (scale > ZONE_DETAIL_THRESHOLD * (max_scale - 1) + 1) {
      this.draw_detail = DrawRisqSpaceDetail.ZONE_DETAILS;
    } else if (scale < 1 / (OWNERSHIP_DETAIL_THRESHOLD * (max_scale - 1) + 1)) {
      this.draw_detail = DrawRisqSpaceDetail.OWNERSHIP;
    } else {
      this.draw_detail = DrawRisqSpaceDetail.SPACE_DETAILS;
    }
  }

  /** Board size in canvas pixels at the current hex radius */
  boardSize(board_size: number): Point2D {
    return {
      x: 1.732 * this.hex_r * (2 * board_size + 1),
      y: 1.5 * this.hex_r * (2 * board_size + 1) + 0.5 * this.hex_r,
    };
  }

  resize(board_size: Point2D, canvas_size: DOMRect, game_board_size: number): { ratio: number; center: Point2D } {
    const new_center_x = 0.5 * Math.min(board_size.x, canvas_size.width);
    const ratio = this.canvas_center.x === 0 ? 1 : new_center_x / this.canvas_center.x;
    this.canvas_center = {
      x: new_center_x,
      y: 0.5 * Math.min(board_size.y, canvas_size.height),
    };
    this.canvas_size = canvas_size;
    this.hex_r = board_size.x / (1.732 * (2 * game_board_size + 1));
    this.hex_a = 0.5 * 1.732 * this.hex_r;
    return { ratio, center: this.canvas_center };
  }

  canvasToCoordinate(canvas: Point2D, board_size: number): Point2D {
    const cy = (canvas.y - 0.25 * this.hex_r) / (1.5 * this.hex_r) - board_size - 0.5;
    return {
      x: canvas.x / (1.732 * this.hex_r) - 0.5 * cy - board_size - 0.5,
      y: cy,
    };
  }

  coordinateToCanvas(coordinate: Point2D): Point2D {
    const game = this.session.getGame();
    if (!game) {
      return { x: 0, y: 0 };
    }
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + game.board_size + 0.5) * this.hex_r,
      y: 1.5 * (coordinate.y + game.board_size + 0.5) * this.hex_r + 0.25 * this.hex_r,
    };
  }

  /** Space/zone-view-aware canvas point for an order endpoint; offset is ignored (space-to-space) outside zone view */
  orderPoint(space: Point2D, offset: Point2D | undefined, zone_view: boolean): Point2D {
    const p = this.coordinateToCanvas(space);
    return zone_view && offset ? addPoint2D(p, offset) : p;
  }

  /** Offset (relative to its space's center) of the specific unit-slot circle a unit currently occupies */
  unitAnchorOffset(unit: RisqUnit): Point2D | undefined {
    if (!this.session.getGame()) {
      return undefined;
    }
    if (unit.garrisoned_in !== undefined) {
      const building = this.session.findBuildingById(unit.garrisoned_in);
      return building ? zoneCenterOffset(building.zone_coordinate, this.hex_r) : undefined;
    }
    const zone = getRisqZone(this.session.spaceAt(unit.space_coordinate), unit.zone_coordinate);
    if (!zone) {
      return zoneCenterOffset(unit.zone_coordinate, this.hex_r);
    }
    return (
      unitSlotWorldPosition(zone, unit.zone_coordinate, this.hex_r, this.session.getPlayerId(), unit.internal_id) ??
      zoneCenterOffset(unit.zone_coordinate, this.hex_r)
    );
  }

  /** Canvas position of a region's label: the centroid of its (explored) member spaces */
  regionLabelPosition(region: RisqRegion): Point2D {
    let sx = 0;
    let sy = 0;
    for (const key of region.spaces) {
      const c = invertPair(key);
      sx += c.x;
      sy += c.y;
    }
    const n = region.spaces.length || 1;
    return this.coordinateToCanvas({ x: sx / n, y: sy / n });
  }

  visibleCanvasBounds(): CanvasBounds {
    const { width, height } = this.canvas_size;
    const corners = [
      screenToCanvas({ x: 0, y: 0 }, this.transform),
      screenToCanvas({ x: width, y: 0 }, this.transform),
      screenToCanvas({ x: 0, y: height }, this.transform),
      screenToCanvas({ x: width, y: height }, this.transform),
    ];
    return {
      min: { x: Math.min(...corners.map((c) => c.x)), y: Math.min(...corners.map((c) => c.y)) },
      max: { x: Math.max(...corners.map((c) => c.x)), y: Math.max(...corners.map((c) => c.y)) },
    };
  }

  isSpaceOnScreen(space: RisqSpace, bounds: CanvasBounds): boolean {
    const { min, max } = bounds;
    return !(
      space.center.x + this.hex_a < min.x ||
      space.center.x - this.hex_a > max.x ||
      space.center.y + this.hex_r < min.y ||
      space.center.y - this.hex_r > max.y
    );
  }

  isCircleOnScreen(c: Point2D, radius: number, bounds: CanvasBounds): boolean {
    const { min, max } = bounds;
    const closest_x = Math.min(Math.max(c.x, min.x), max.x);
    const closest_y = Math.min(Math.max(c.y, min.y), max.y);
    const dx = c.x - closest_x;
    const dy = c.y - closest_y;
    return dx * dx + dy * dy <= radius * radius;
  }
}
