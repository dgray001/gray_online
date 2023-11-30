import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';
import {BoardTransformData, DwgCanvasBoard} from '../../util/canvas_board/canvas_board';
import {Point2D, equalsPoint2D, hexagonalBoardNeighbors, hexagonalBoardRows, roundAxialCoordinate} from '../../util/objects2d';
import {DwgGame} from '../../game';

import html from './risq.html';
import {GameRisq, RisqSpace} from './risq_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';

const HEXAGON_RADIUS = 60;

export class DwgRisq extends DwgElement {
  board: DwgCanvasBoard;

  game: GameRisq;
  canvas_center: Point2D = {x: 0, y: 0};
  mouse_canvas: Point2D = {x: 0, y: 0};
  mouse_coordinate: Point2D = {x: 0, y: 0};
  hovered_space?: RisqSpace;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('board');
  }

  private draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    ctx.strokeStyle = "rgba(250, 250, 250, 0.9)";
    ctx.lineWidth = 2;
    for (const row of this.game.spaces) {
      for (const space of row) {
        space.center = this.coordinateToCanvas(space.coordinate, transform.scale);
        if (space.visibility > 0) {
          ctx.fillStyle = "green";
        } else {
          ctx.fillStyle = "transparent";
        }
        drawHexagon(ctx, space.center, HEXAGON_RADIUS);
        ctx.fill();
        if (space.hovered) {
          if (space.clicked) {
            ctx.fillStyle = "rgba(210, 210, 210, 0.4)";
          } else {
            ctx.fillStyle = "rgba(190, 190, 190, 0.2)";
          }
          drawHexagon(ctx, space.center, HEXAGON_RADIUS);
          ctx.fill();
        }
      }
    }
    ctx.fillStyle = "red";
    ctx.fillRect(
      (this.canvas_center.x + transform.view.x) / transform.scale - 5,
      (this.canvas_center.y + transform.view.y) / transform.scale - 5,
    10, 10);
    ctx.fillStyle = "blue";
    ctx.fillRect(this.mouse_canvas.x - 5, this.mouse_canvas.y - 5, 10, 10);
  }

  private canvasToCoordinate(canvas: Point2D, scale: number): Point2D {
    const cy = (((canvas.y - 0.25 * HEXAGON_RADIUS - this.canvas_center.y / scale) / (1.5 * HEXAGON_RADIUS)) - this.game.board_size - 0.5);
    return {
      x: ((canvas.x - this.canvas_center.x / scale) / (1.732 * HEXAGON_RADIUS)) - 0.5 * cy - this.game.board_size - 0.5,
      y: cy,
    };
  }

  private coordinateToCanvas(coordinate: Point2D, scale: number): Point2D {
    return {
      x: 1.732 * (coordinate.x + 0.5 * coordinate.y + this.game.board_size + 0.5) * HEXAGON_RADIUS + this.canvas_center.x / scale,
      y: 1.5 * (coordinate.y + this.game.board_size + 0.5) * HEXAGON_RADIUS + 0.25 * HEXAGON_RADIUS + this.canvas_center.y / scale,
    };
  }

  private coordinateToIndex(coordinate: Point2D): Point2D {
    return {
      x: coordinate.y + this.game.board_size,
      y: coordinate.x - Math.max(-this.game.board_size, -(this.game.board_size + coordinate.y)),
    };
  }

  private mousemove(m: Point2D, transform: BoardTransformData) {
    this.mouse_canvas = m;
    this.mouse_coordinate = this.canvasToCoordinate(m, transform.scale);
    const index = this.coordinateToIndex(roundAxialCoordinate(this.mouse_coordinate));
    const new_hovered_space = this.getSpace(index);
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
      const index = this.coordinateToIndex(neighbor);
      const space = this.getSpace(index);
      if (!!space) {
        neighbors.push(space);
      }
    }
    return neighbors;
  }

  private getBoardRows(space: RisqSpace): RisqSpace[] {
    const rows: RisqSpace[] = [];
    for (const neighbor of hexagonalBoardRows(space.coordinate, this.game.board_size)) {
      const index = this.coordinateToIndex(neighbor);
      const space = this.getSpace(index);
      if (!!space) {
        rows.push(space);
      }
    }
    return rows;
  }

  private mousedown(e: MouseEvent) {
    if (!!this.hovered_space) {
      this.hovered_space.clicked = true;
    }
  }

  private mouseup(e: MouseEvent) {
    if (!!this.hovered_space) {
      this.hovered_space.clicked = false;
    }
  }

  private getSpace(index: Point2D): RisqSpace|undefined {
    if (index.x < 0 || index.x >= this.game.spaces.length) {
      return undefined;
    }
    const row = this.game.spaces[index.x];
    if (index.y < 0 || index.y >= row.length) {
      return undefined;
    }
    return row[index.y];
  }

  initialize(abstract_game: DwgGame, game: GameRisq, client_id: number): void {
    abstract_game.setPadding('0px');
    this.game = game;
    const canvas_size = {
      x: 1.732 * HEXAGON_RADIUS * (2 * game.board_size + 1),
      y: 1.5 * HEXAGON_RADIUS * (2 * game.board_size + 1) + 0.5 * HEXAGON_RADIUS,
    };
    this.board.initialize({
      size: canvas_size,
      max_scale: 1.8,
      draw: this.draw.bind(this),
      mousemove: this.mousemove.bind(this),
      mousedown: this.mousedown.bind(this),
      mouseup: this.mouseup.bind(this),
    }).then(() => {
      const canvas_rect = this.board.getBoundingClientRect();
      this.canvas_center = {
        x: 0.5 * Math.min(canvas_size.x, canvas_rect.width),
        y: 0.5 * Math.min(canvas_size.y, canvas_rect.height),
      };
    });
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
  }
}

customElements.define('dwg-risq', DwgRisq);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-risq': DwgRisq;
  }
}
