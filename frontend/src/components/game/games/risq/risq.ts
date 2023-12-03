import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';
import {BoardTransformData, DwgCanvasBoard} from '../../util/canvas_board/canvas_board';
import {Point2D, equalsPoint2D, hexagonalBoardNeighbors, hexagonalBoardRows, roundAxialCoordinate} from '../../util/objects2d';
import {DwgGame} from '../../game';
import {ColorRGB} from '../../../../scripts/color_rgb';

import html from './risq.html';
import {GameRisq, RisqSpace, coordinateToIndex, getSpace, getSpaceFill} from './risq_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './space_dialog/space_dialog';

const DEFAULT_HEXAGON_RADIUS = 60;

export class DwgRisq extends DwgElement {
  board: DwgCanvasBoard;

  game: GameRisq;
  hexagon_radius = DEFAULT_HEXAGON_RADIUS;
  canvas_center: Point2D = {x: 0, y: 0};
  mouse_canvas: Point2D = {x: 0, y: 0};
  mouse_coordinate: Point2D = {x: 0, y: 0};
  hovered_space?: RisqSpace;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('board');
  }

  async initialize(abstract_game: DwgGame, game: GameRisq): Promise<void> {
    abstract_game.setPadding('0px');
    this.game = game;
    const canvas_size = {
      x: 1.732 * this.hexagon_radius * (2 * game.board_size + 1),
      y: 1.5 * this.hexagon_radius * (2 * game.board_size + 1) + 0.5 * this.hexagon_radius,
    };
    this.board.initialize({
      size: canvas_size,
      max_scale: 1.8,
      fill_space: true,
      draw: this.draw.bind(this),
      mousemove: this.mousemove.bind(this),
      mouseleave: this.mouseleave.bind(this),
      mousedown: this.mousedown.bind(this),
      mouseup: this.mouseup.bind(this),
    }).then(() => {
      const canvas_rect = this.board.getBoundingClientRect();
      this.canvas_center = {
        x: 0.5 * Math.min(canvas_size.x, canvas_rect.width),
        y: 0.5 * Math.min(canvas_size.y, canvas_rect.height),
      };
      // canvas board component should respect scale
      this.hexagon_radius = (2 * this.canvas_center.x) / (1.732 * (2 * game.board_size + 1));
    });
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
  }

  private draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    ctx.strokeStyle = 'rgba(250, 250, 250, 0.9)';
    ctx.lineWidth = 2;
    for (const row of this.game.spaces) {
      for (const space of row) {
        ctx.fillStyle = getSpaceFill(space).getString();
        space.center = this.coordinateToCanvas(space.coordinate, transform.scale);
        drawHexagon(ctx, space.center, this.hexagon_radius);
        if (space.visibility > 0) {
          // TODO: draw summary info of space
        }
      }
    }
  }

  private mousemove(m: Point2D, transform: BoardTransformData) {
    this.mouse_canvas = m;
    this.mouse_coordinate = this.canvasToCoordinate(m, transform.scale);
    const index = coordinateToIndex(this.game.board_size, roundAxialCoordinate(this.mouse_coordinate));
    const new_hovered_space = getSpace(this.game, index);
    if (!new_hovered_space) {
      this.removeHoveredFlags();
      if (!!this.hovered_space) {
        this.hovered_space.clicked = false;
        this.hovered_space = undefined;
      }
      return;
    }
    if (equalsPoint2D(new_hovered_space.coordinate, this.hovered_space?.coordinate)) {
      this.updateHoveredFlags();
      return;
    }
    this.removeHoveredFlags();
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
    this.hovered_space = new_hovered_space;
    this.updateHoveredFlags();
  }

  private mouseleave() {
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
      this.hovered_space.clicked = false;
      this.hovered_space = undefined;
    }
  }

  private mousedown(_e: MouseEvent) {
    if (!!this.hovered_space) {
      this.hovered_space.clicked = true;
    }
  }

  private mouseup(_e: MouseEvent) {
    if (!!this.hovered_space) {
      if (this.hovered_space.clicked && this.hovered_space.visibility > 0) {
        this.openSpaceDialog(this.hovered_space);
      }
      this.hovered_space.clicked = false;
    }
  }

  private openSpaceDialog(space: RisqSpace) {
    if (!space) {
      return;
    }
    const space_dialog = document.createElement('dwg-space-dialog');
    space_dialog.setData({space, game: this.game});
    this.appendChild(space_dialog);
  }

  private canvasToCoordinate(canvas: Point2D, scale: number): Point2D {
    const cy = (((canvas.y - 0.25 * this.hexagon_radius - this.canvas_center.y / scale) / (1.5 * this.hexagon_radius)) - this.game.board_size - 0.5);
    return {
      x: ((canvas.x - this.canvas_center.x / scale) / (1.732 * this.hexagon_radius)) - 0.5 * cy - this.game.board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToCanvas(coordinate: Point2D, scale: number): Point2D {
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + this.game.board_size + 0.5) * this.hexagon_radius + this.canvas_center.x / scale,
      y: 1.5 * (coordinate.y + this.game.board_size + 0.5) * this.hexagon_radius + 0.25 * this.hexagon_radius + this.canvas_center.y / scale,
    };
  }

  private removeHoveredFlags() {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = false;
    for (const neighbor of this.getBoardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = false;
    }
    for (const row of this.getBoardRows(this.hovered_space)) {
      row.hovered_row = false;
    }
  }

  private updateHoveredFlags() {
    if (!this.hovered_space) {
      return;
    }
    this.hovered_space.hovered = true;
    for (const neighbor of this.getBoardNeighbors(this.hovered_space)) {
      neighbor.hovered_neighbor = true;
    }
    for (const row of this.getBoardRows(this.hovered_space)) {
      row.hovered_row = true;
    }
  }

  private getBoardNeighbors(space: RisqSpace): RisqSpace[] {
    const neighbors: RisqSpace[] = [];
    for (const neighbor of hexagonalBoardNeighbors(space.coordinate, this.game.board_size)) {
      const index = coordinateToIndex(this.game.board_size, neighbor);
      const space = getSpace(this.game, index);
      if (!!space) {
        neighbors.push(space);
      }
    }
    return neighbors;
  }

  private getBoardRows(space: RisqSpace): RisqSpace[] {
    const rows: RisqSpace[] = [];
    for (const neighbor of hexagonalBoardRows(space.coordinate, this.game.board_size)) {
      const index = coordinateToIndex(this.game.board_size, neighbor);
      const space = getSpace(this.game, index);
      if (!!space) {
        rows.push(space);
      }
    }
    return rows;
  }
}

customElements.define('dwg-risq', DwgRisq);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq': DwgRisq;
  }
}
