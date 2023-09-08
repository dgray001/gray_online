import {DwgElement} from '../../dwg_element';

import html from './lobby_room.html';
import './lobby_room.scss';

export class DwgLobbyRoom extends DwgElement {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example', 'example');
  }

  protected override parsedCallback(): void {
    console.log('DwgLobbyRoom parsed!');
  }
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);
