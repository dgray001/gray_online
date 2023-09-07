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
      socket.addEventListener('error', (e) => {
        console.log(e);
        this.tryConnectionAgain("Could not connect. Check your connection and try again.");
      });
      socket.addEventListener('open', () => {
        this.lobby.setSocket(socket);
        this.lobby.connection_metadata.nickname = nickname;
        this.lobby.name_header.innerText = nickname;
        this.lobby_connector.classList.add('hide');
      });
    });
    this.lobby.addEventListener('connection_lost', () => {
      this.tryConnectionAgain("Connection was lost. Check your connection and try again.");
    });
  }

  private tryConnectionAgain(message: string): void {
    this.lobby_connector.classList.remove('hide');
    this.lobby_connector.status_message.innerText = message;
    this.lobby_connector.connect_button.disabled = false;
    this.lobby_connector.connect_button.innerText = "Connect to Lobby";
  }
}

customElements.define('dwg-page-home', DwgPageHome);
