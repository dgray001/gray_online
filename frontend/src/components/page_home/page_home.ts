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
      const socket = new WebSocket(`ws://127.0.0.1:6807/api/lobby/connect/:${nickname}`);
      socket.addEventListener('open', () => {
        this.lobby.setSocket(socket);
        this.lobby.connection_metadata.nickname = nickname;
        this.lobby.name_header.innerText = nickname;
        this.lobby_connector.classList.add('hide');
      });
    });
  }
}

customElements.define('dwg-page-home', DwgPageHome);
