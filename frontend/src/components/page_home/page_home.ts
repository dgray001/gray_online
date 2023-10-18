import {DwgElement} from '../dwg_element';
import {DwgLobby} from '../lobby/lobby';
import {DwgLobbyConnector} from '../lobby/lobby_connector/lobby_connector';
import {DwgGame} from '../game/game';
import {LobbyRoom} from '../lobby/data_models';
import '../lobby/lobby';
import '../game/game';
import '../lobby/lobby_connector/lobby_connector';

import html from './page_home.html';
import './page_home.scss';

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
      let client_id_time = 0;
      let client_id = 0;
      try {
        client_id_time = parseInt(localStorage.getItem("client_id_time"));
        client_id = parseInt(localStorage.getItem("client_id"));
      } catch(e) {} // if localstorage isn't accessible
      const socket = (!!client_id && !!client_id_time && (Date.now() - client_id_time < 1000 * 60 * 60 * 24)) ?
        new WebSocket(`ws://${location.hostname}:6807/api/lobby/reconnect/${nickname}/${client_id}`) :
        new WebSocket(`ws://${location.hostname}:6807/api/lobby/connect/${nickname}`);
      socket.addEventListener('error', (e) => {
        console.log(e);
        this.tryConnectionAgain("Could not connect. Check your connection and try again.");
      });
      socket.addEventListener('open', () => {
        this.lobby.setSocket(socket);
        this.lobby.setNickname(nickname);
        this.lobby.setPing(0);
        this.lobby_connector.classList.add('hide');
        this.lobby.classList.remove('connector-open');
      });
    });
    this.lobby.addEventListener('connection_lost', () => {
      this.tryConnectionAgain("Connection was lost. Check your connection and try again.");
    });
    this.lobby.addEventListener('game_launched', async (e: CustomEvent<LobbyRoom>) => {
      if (await this.game.launchGame(e.detail, this.lobby.socket, this.lobby.connection_metadata)) {
        this.lobby.classList.add('hide');
      }
    });
    this.game.addEventListener('exit_game', () => {
      this.game.classList.remove('show');
      this.lobby.classList.remove('hide');
    });
  }

  private tryConnectionAgain(message: string): void {
    this.lobby.classList.remove('connected');
    this.lobby_connector.classList.remove('hide');
    this.lobby.classList.add('connector-open');
    this.lobby_connector.status_message.innerText = message;
    this.lobby_connector.connect_button.disabled = false;
    this.lobby_connector.connect_button.innerText = "Reconnect to Lobby";
  }
}

customElements.define('dwg-page-home', DwgPageHome);
