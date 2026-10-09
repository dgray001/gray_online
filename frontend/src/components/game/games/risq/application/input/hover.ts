import type { Point2D } from '../../../../util/objects2d';
import { equalsPoint2D, roundAxialCoordinate } from '../../../../util/objects2d';
import type { RisqRegion, RisqSpace, RisqZone } from '../../model/types';
import type { RisqViewport } from '../../rendering/board/viewport';
import { RisqViewMode } from '../../rendering/terrain';
import { resolveHoveredZones, unhoverRisqZone } from '../../rendering/zones/hit_testing';
import type { RisqSession } from '../session';

/** The mouse's last board position and the space/zone under it, including their hover and click flags */
export class RisqHover {
  private mouse_canvas: Point2D = { x: 0, y: 0 };
  private mouse_screen: Point2D = { x: 0, y: 0 };
  private hovered_space?: RisqSpace;
  private hovered_zone?: RisqZone;

  constructor(
    private session: RisqSession,
    private viewport: RisqViewport
  ) {}

  space(): RisqSpace | undefined {
    return this.hovered_space;
  }

  zone(): RisqZone | undefined {
    return this.hovered_zone;
  }

  mouseCanvas(): Point2D {
    return this.mouse_canvas;
  }

  mouseScreen(): Point2D {
    return this.mouse_screen;
  }

  setMouse(canvas: Point2D, screen: Point2D): void {
    this.mouse_canvas = canvas;
    this.mouse_screen = screen;
  }

  hoveredRegion(): RisqRegion | undefined {
    if (!this.viewport.regionInteractionEnabled() || !this.hovered_space) {
      return undefined;
    }
    return this.session.getRegionForSpace(this.hovered_space.coordinate_key);
  }

  /** Drops references into a snapshot that's about to be replaced */
  forget(): void {
    this.hovered_space = undefined;
    this.hovered_zone = undefined;
  }

  /** Resolves the hovered space and zone under the mouse; returns whether the board itself is hovered */
  update(over_other_component: boolean): boolean {
    const game = this.session.getGame();
    if (!game) {
      return false;
    }
    if (!!this.hovered_space) {
      this.hovered_space.center = this.viewport.coordinateToCanvas(this.hovered_space.coordinate);
    }
    const mouse_coordinate = this.viewport.canvasToCoordinate(this.mouse_canvas, game.board_size);
    const new_hovered_space = this.session.spaceAt(roundAxialCoordinate(mouse_coordinate));
    if (over_other_component || !new_hovered_space) {
      this.removeHoveredFlags();
      if (!!this.hovered_space) {
        this.hovered_space.clicked = false;
        this.hovered_space = undefined;
        this.unhoverZone();
      }
      return false;
    }
    if (equalsPoint2D(new_hovered_space.coordinate, this.hovered_space?.coordinate)) {
      this.updateHoveredFlags();
      this.resolveZone();
      return true;
    }
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
      this.unhoverZone();
    }
    this.hovered_space = new_hovered_space;
    this.resolveZone();
    this.updateHoveredFlags();
    return true;
  }

  private resolveZone(): void {
    if (this.viewport.zoneView() && this.viewport.viewMode() !== RisqViewMode.REGION) {
      const new_hovered_zone = resolveHoveredZones(this.mouse_canvas, this.hovered_space, this.viewport.hexR());
      if (!!this.hovered_zone && !equalsPoint2D(new_hovered_zone?.coordinate, this.hovered_zone?.coordinate)) {
        this.unhoverZone();
      }
      this.hovered_zone = new_hovered_zone;
    } else {
      this.unhoverZone();
    }
  }

  private unhoverZone(): void {
    if (!!this.hovered_zone) {
      unhoverRisqZone(this.hovered_zone);
      this.hovered_zone = undefined;
    }
  }

  /** Cancels any click pending on the hovered space/zone, e.g. once a press turns into a drag */
  clearClicks(): void {
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
    if (!!this.hovered_zone) {
      this.hovered_zone.clicked = false;
      for (const part of this.hovered_zone.hovered_data) {
        part.clicked = false;
      }
    }
  }

  /** Board pan has started: drop highlight flags and pending clicks */
  dragging(): void {
    this.removeHoveredFlags();
    this.clearClicks();
  }

  leave(): void {
    this.unhoverZone();
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
      this.hovered_space.clicked = false;
      this.hovered_space = undefined;
    }
  }

  private removeHoveredFlags(): void {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = false;
    for (const neighbor of this.session.boardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = false;
    }
    for (const row of this.session.boardRows(this.hovered_space)) {
      row.hovered_row = false;
    }
  }

  private updateHoveredFlags(): void {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = true;
    for (const neighbor of this.session.boardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = true;
    }
    for (const row of this.session.boardRows(this.hovered_space)) {
      row.hovered_row = true;
    }
  }
}
