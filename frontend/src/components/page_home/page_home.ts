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
  private lobby: DwgLobby;
  private game: DwgGame;
  private lobby_connector: DwgLobbyConnector;

  private client_on_mobile = false;

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
      let never_connected = true;
      socket.addEventListener('error', (e) => {
        console.log(e);
        this.tryConnectionAgain(`${never_connected ? 'Could not connect' :
          'Connection was lost'}. Check your connection and try again.`);
      });
      socket.addEventListener('open', () => {
        never_connected = false;
        this.lobby_connector.classList.add('hide');
        this.lobby.connect(e.detail.nickname, socket);
        this.game.exitGame();
      });
    });
    this.lobby.addEventListener('connection_lost', () => {
      this.tryConnectionAgain('Connection was lost. Check your connection and try again.');
    });
    this.lobby.addEventListener('game_launched', async (e: CustomEvent<LobbyRoom>) => {
      if (await this.game.launchGame(e.detail, this.lobby.getSocket(), this.lobby.getConnectionMetadata())) {
        this.lobby.classList.add('hide');
      }
    });
    this.lobby.addEventListener('rejoin_game', async (e: CustomEvent<LobbyRoom>) => {
      if (await this.game.launchGame(e.detail, this.lobby.getSocket(), this.lobby.getConnectionMetadata(), true)) {
        this.lobby.enterGame();
      }
    });
    this.game.addEventListener('exit_game', () => {
      this.game.exitGame();
      this.lobby.exitGame();
    });
    this.game.addEventListener('show_message_dialog', (e: CustomEvent<MessageDialogData>) => {
      const dialog = document.createElement('dwg-message-dialog');
      dialog.setData(e.detail);
      this.appendChild(dialog);
    });
  }

  private tryConnectionAgain(message: string): void {
    this.lobby.classList.remove('connected');
    this.lobby.classList.add('connector-open');
    this.lobby_connector.tryReconnecting(message, this.lobby.getConnectionMetadata());
  }
}

customElements.define('dwg-page-home', DwgPageHome);
