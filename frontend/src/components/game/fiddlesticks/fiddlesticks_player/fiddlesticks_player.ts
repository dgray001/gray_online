import {DwgElement} from '../../../dwg_element';
import {FiddlesticksPlayer} from '../fiddlesticks';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  example: HTMLDivElement;

  player: FiddlesticksPlayer;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgFiddlesticksPlayer parsed!');
  }

  initialize(player: FiddlesticksPlayer) {
    this.player = player;
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
