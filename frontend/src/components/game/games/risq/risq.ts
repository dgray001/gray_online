import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';
import {BoardTransformData, DwgCanvasBoard} from '../../util/canvas_board/canvas_board';
import {Point2D} from '../../util/objects2d';

import html from './risq.html';
import {GameRisq} from './risq_data';

import './risq.scss';
import '../../util/canvas_board/canvas_board';

const HEXAGON_RADIUS = 50;

export class DwgRisq extends DwgElement {
  board: DwgCanvasBoard;

  board_size = 12; // radius from center hexagon
  canvas_center: Point2D = {x: 0, y: 0};

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('board');
  }

  private draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData) {
    ctx.strokeStyle = "green";
    ctx.lineWidth = 4;
    for (let j = -this.board_size; j <= this.board_size; j++) {
      const i_start = Math.max(-this.board_size, -(this.board_size + j));
      const i_end = Math.min(this.board_size, (this.board_size - j));
      for (let i = i_start; i <= i_end; i++) {
        const center = {
          x: 1.732 * (i + 0.5 * j + this.board_size + 0.5) * HEXAGON_RADIUS + this.canvas_center.x / transform.scale,
          y: 1.5 * (j + this.board_size + 0.5) * HEXAGON_RADIUS + 0.25 * HEXAGON_RADIUS + this.canvas_center.y / transform.scale,
        };
        drawHexagon(ctx, center, HEXAGON_RADIUS);
      }
    }
    ctx.fillStyle = "red";
    ctx.fillRect(
      (this.canvas_center.x + transform.view.x) / transform.scale - 5,
      (this.canvas_center.y + transform.view.y) / transform.scale - 5,
    10, 10);
  }

  initialize(game: GameRisq, client_id: number): void {
    // TODO: set board size from game
    const canvas_size = {
      x: 1.732 * HEXAGON_RADIUS * (2 * this.board_size + 1),
      y: 1.5 * HEXAGON_RADIUS * (2 * this.board_size + 1) + 0.5 * HEXAGON_RADIUS,
    };
    this.board.initialize({
      size: canvas_size,
      max_scale: 1.8,
      draw: this.draw.bind(this),
    }).then(() => {
      const canvas_rect = this.board.getBoundingClientRect();
      this.canvas_center = {
        x: 0.5 * Math.min(canvas_size.x, canvas_rect.width),
        y: 0.5 * Math.min(canvas_size.y, canvas_rect.height),
      };
      console.log(canvas_rect, this.canvas_center);
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
