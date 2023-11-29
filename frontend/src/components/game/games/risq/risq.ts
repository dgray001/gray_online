import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';
import {BoardTransformData, DwgCanvasBoard} from '../../util/canvas_board/canvas_board';
import {Point2D} from '../../util/objects2d';

import html from './risq.html';
import {GameRisq, RisqSpace} from './risq_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';

const HEXAGON_RADIUS = 60;

export class DwgRisq extends DwgElement {
  board: DwgCanvasBoard;

  game: GameRisq;
  canvas_center: Point2D = {x: 0, y: 0};
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
    for (const [j, row] of this.game.spaces.entries()) {
      for (const [i, space] of row.entries()) {
        space.center = this.coordinateToCanvas(space.coordinate, transform.scale);
        if (space.hovered) {
          ctx.fillStyle = "rgba(190, 190, 190, 0.2)";
        }
        drawHexagon(ctx, space.center, HEXAGON_RADIUS);
        if (space.hovered) {
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
    ctx.fillRect(this.mouse_coordinate.x - 5, this.mouse_coordinate.y - 5, 10, 10);
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
    this.mouse_coordinate = this.canvasToCoordinate(m, transform.scale);
    if (!!this.hovered_space) {
      this.hovered_space.hovered = false;
    }
    const index = this.coordinateToIndex({x: Math.round(this.mouse_coordinate.x), y: Math.round(this.mouse_coordinate.y)});
    if (index.x < 0 || index.x >= this.game.spaces.length) {
      return;
    }
    const row = this.game.spaces[index.x];
    if (index.y < 0 || index.y >= row.length) {
      return;
    }
    this.hovered_space = row[index.y];
    this.hovered_space.hovered = true;
  }

  initialize(game: GameRisq, client_id: number): void {
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
