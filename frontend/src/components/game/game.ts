import {DwgElement} from '../dwg_element';

import html from './game.html';
import './game.scss';

export class DwgGame extends DwgElement {
  example: HTMLDivElement;

  launched: boolean;
  socket: WebSocket;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgGame parsed!');
  }

  launchGame(game_id: number, socket: WebSocket) {
    if (this.launched) {
      return;
    }
    try {
      // TODO: send request for game object
      this.classList.add('show');
    } catch(e) {}
  }
}

customElements.define('dwg-game', DwgGame);
