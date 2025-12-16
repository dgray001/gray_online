import { DwgElement } from '../dwg_element';
import type { DwgGame } from '../game/game';

import html from './page_dev.html';

import './page_dev.scss';
import '../game/game';

export class DwgPageDev extends DwgElement {
  private game!: DwgGame;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('game');
  }

  protected override parsedCallback(): void {
    console.log('DwgPageDev parsed!');
  }
}

customElements.define('dwg-page-dev', DwgPageDev);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-dev': DwgPageDev;
  }
}
