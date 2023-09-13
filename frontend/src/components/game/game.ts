import {DwgElement} from '../dwg_element';

import html from './game.html';
import './game.scss';

export class DwgGame extends DwgElement {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgGame parsed!');
  }
}

customElements.define('dwg-game', DwgGame);
