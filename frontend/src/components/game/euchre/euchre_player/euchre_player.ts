import {DwgElement} from '../../../dwg_element';
import {EuchrePlayer} from '../euchre_data';

import html from './euchre_player.html';

import './euchre_player.scss';

export class DwgEuchrePlayer extends DwgElement {
  example: HTMLDivElement;

  initialized = false;
  player: EuchrePlayer;
  client_player = false;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgEuchrePlayer parsed!');
  }

  initialize(player: EuchrePlayer) {
    this.player = player;
    this.initialized = true;
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
  }
}

customElements.define('dwg-euchre-player', DwgEuchrePlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-euchre-player': DwgEuchrePlayer;
  }
}
