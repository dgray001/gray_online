import {DwgElement} from '../../../dwg_element';
import {UpdateMessage} from '../../data_models';
import {drawHexagon} from '../../util/canvas_util';

import html from './risq.html';
import {GameRisq} from './risq_data';

import './risq.scss';

export class DwgRisq extends DwgElement {
  board: HTMLCanvasElement;

  ctx: CanvasRenderingContext2D;
  scale = 1;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('board');
  }

  protected override parsedCallback(): void {
    if (!this.board.getContext) {
      console.error('Browser does not support canvas; cannot draw board');
      return;
    }
    this.ctx = this.board.getContext('2d');
    setInterval(() => {this.draw();}, 100);
    this.board.addEventListener('wheel', (e: WheelEvent) => {
      const zoom = 1 + e.deltaY/200;
      this.setScale(this.scale * zoom);
    });
  }

  static MAX_SCALE = 2.5;
  private setScale(scale: number) {
    if (scale < 1/DwgRisq.MAX_SCALE) {
      scale = 1/DwgRisq.MAX_SCALE;
    } else if (scale > DwgRisq.MAX_SCALE) {
      scale = DwgRisq.MAX_SCALE;
    }
    this.scale = scale;
    this.ctx.resetTransform();
    this.ctx.scale(this.scale, this.scale);
  }

  private draw() {
    this.ctx.fillStyle = "black";
    this.ctx.fillRect(0, 0, 700/this.scale, 700/this.scale);
    this.ctx.strokeStyle = "green";
    this.ctx.lineWidth = 4;
    drawHexagon(this.ctx, 100/this.scale, 120/this.scale, 50);
  }

  initialize(game: GameRisq, client_id: number): void {
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
