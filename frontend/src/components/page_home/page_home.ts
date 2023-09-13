import {DwgElement} from '../dwg_element';
import {DwgLobby} from '../lobby/lobby';
import {DwgLobbyConnector} from '../lobby/lobby_connector/lobby_connector';
import {DwgGame} from '../game/game';
import '../lobby/lobby';
import '../game/game';
import '../lobby/lobby_connector/lobby_connector';

import html from './page_home.html';
import './page_home.scss';
import { LobbyRoom } from '../lobby/data_models';

export class DwgPageHome extends DwgElement {
  lobby: DwgLobby;
  game: DwgGame;
  lobby_connector: DwgLobbyConnector;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('lobby');
    this.configureElement('game');
    this.configureElement('lobby_connector');
  }

  protected override parsedCallback(): void {
    this.lobby_connector.addEventListener('submitted', () => {
      const nickname = this.lobby_connector.nickname.value;
      const socket = new WebSocket(`ws://127.0.0.1:6807/api/lobby/connect/${nickname}`);
      socket.addEventListener('error', (e) => {
        console.log(e);
        this.tryConnectionAgain("Could not connect. Check your connection and try again.");
      });
      socket.addEventListener('open', () => {
        this.lobby.setSocket(socket);
        this.lobby.setNickname(nickname);
        this.lobby.setPing(0);
        this.lobby_connector.classList.add('hide');
      });
    });
    this.lobby.addEventListener('connection_lost', () => {
      this.tryConnectionAgain("Connection was lost. Check your connection and try again.");
    });
    this.lobby.addEventListener('game_launched', (e: CustomEvent<number>) => {
      this.game.launchGame(e.detail, this.lobby.socket);
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
