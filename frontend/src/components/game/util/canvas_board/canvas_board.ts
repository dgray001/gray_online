import {DwgElement} from '../../../dwg_element';

import html from './canvas_board.html';

import './canvas_board.scss';

/** Data describing how the canvas should be initialized */
export declare interface CanvasBoardInitializationData {
  //
}

export class DwgCanvasBoard extends DwgElement {
  canvas: HTMLCanvasElement;
  ctx: CanvasRenderingContext2D;

  scale = 1;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('canvas');
  }

  protected override parsedCallback(): void {
  }

  initialize(data: CanvasBoardInitializationData) {
    //
  }
}

customElements.define('dwg-canvas-board', DwgCanvasBoard);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-canvas-board': DwgCanvasBoard;
  }
}
