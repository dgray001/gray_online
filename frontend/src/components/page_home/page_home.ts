import {DwgElement} from '../dwg_element';
import {DwgLobby} from '../lobby/lobby';
import {DwgLobbyConnector} from '../lobby_connector/lobby_connector';
import '../lobby/lobby';
import '../lobby_connector/lobby_connector';

import html from './page_home.html';
import './page_home.scss';

export class DwgPageHome extends DwgElement {
  lobby: DwgLobby;
  lobby_connector: DwgLobbyConnector;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('lobby');
    this.configureElement('lobby_connector');
  }

  protected override parsedCallback(): void {
    this.lobby_connector.addEventListener('submitted', () => {
      const nickname = this.lobby_connector.nickname.value;
      console.log(nickname);
    });
  }
}

customElements.define('dwg-page-home', DwgPageHome);
