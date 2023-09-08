import {DwgElement} from '../../dwg_element';

import html from './lobby_users.html';
import './lobby_users.scss';

export class DwgLobbyUsers extends DwgElement {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgLobbyUsers parsed!');
  }
}

customElements.define('dwg-lobby-users', DwgLobbyUsers);
