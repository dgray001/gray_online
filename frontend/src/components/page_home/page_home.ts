import {DwgElement} from '../dwg_element';
import {DwgLobby} from '../lobby/lobby';
import {ConnectData, DwgLobbyConnector} from '../lobby/lobby_connector/lobby_connector';
import {DwgGame} from '../game/game';
import {LobbyRoom} from '../lobby/data_models';
import {websocketPath} from '../../scripts/api';
import {clientOnMobile} from '../../scripts/util';
import {MessageDialogData} from '../dialog_box/message_dialog/message_dialog';

import html from './page_home.html';

import './page_home.scss';
import '../lobby/lobby';
import '../game/game';
import '../lobby/lobby_connector/lobby_connector';

export class DwgPageHome extends DwgElement {
  lobby: DwgLobby;
  game: DwgGame;
  lobby_connector: DwgLobbyConnector;

  client_on_mobile = false;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('lobby');
    this.configureElement('game');
    this.configureElement('lobby_connector');
  }

  protected override parsedCallback(): void {
    this.client_on_mobile = clientOnMobile();
    if (this.client_on_mobile) {
      document.body.classList.add('mobile');
    }
    this.lobby_connector.addEventListener('connect', (e: CustomEvent<ConnectData>) => {
      const socket = e.detail.try_reconnect ?
        new WebSocket(`${websocketPath()}/reconnect/${e.detail.nickname}/${e.detail.client_id}`) :
        new WebSocket(`${websocketPath()}/connect/${e.detail.nickname}`);
      socket.addEventListener('error', (e) => {
        console.log(e);
        this.tryConnectionAgain("Could not connect. Check your connection and try again.");
      });
      socket.addEventListener('open', () => {
        this.lobby_connector.classList.add('hide');
        this.lobby.classList.remove('connector-open');
        this.lobby.setNickname(e.detail.nickname);
        this.lobby.setPing(0);
        this.lobby.setSocket(socket);
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
    this.lobby.addEventListener('rejoin_game', async (e: CustomEvent<LobbyRoom>) => {
      if (await this.game.launchGame(e.detail, this.lobby.socket, this.lobby.connection_metadata, true)) {
        this.lobby.classList.add('hide');
      }
    });
    this.game.addEventListener('exit_game', () => {
      this.game.classList.remove('show');
      this.lobby.classList.remove('hide');
    });
    this.game.addEventListener('show_message_dialog', (e: CustomEvent<MessageDialogData>) => {
      const dialog = document.createElement('dwg-message-dialog');
      dialog.setData(e.detail);
      this.appendChild(dialog);
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
