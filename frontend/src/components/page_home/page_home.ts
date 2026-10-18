import { DwgElement } from '../dwg_element';
import type { DwgLobby } from '../lobby/lobby';
import type { DwgLobbyConnector } from '../lobby/lobby_connector/lobby_connector';
import type { DwgGame } from '../game/game';
import type { ServerMessage } from '../lobby/data_models';
import { websocketPath } from '../../scripts/api';
import { PREVIOUS_NICKNAME } from '../lobby/lobby_connector/lobby_connector';

import html from './page_home.html';

import './page_home.scss';
import '../lobby/lobby';
import '../game/game';
import '../lobby/lobby_connector/lobby_connector';

const SOCKET_SILENCE_MS = 3000;
const WATCHDOG_TICK_MS = 1000;
const FROZEN_TICK_GAP_MS = 2000;
const RECONNECT_DELAYS_MS = [0, 500, 1000, 2000, 3000, 5000, 5000, 5000, 5000];
const REPLACED_CLOSE_CODE = 4001;
const ABANDONED_CLOSE_CODE = 4000;

export class DwgPageHome extends DwgElement {
  private lobby!: DwgLobby;
  private game!: DwgGame;
  private lobby_connector!: DwgLobbyConnector;
  private reconnecting_mask!: HTMLDivElement;

  private socket?: WebSocket;
  private last_message_at = 0;
  private last_tick_at = Date.now();
  private reconnecting = false;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements('lobby', 'game', 'lobby_connector', 'reconnecting_mask');
  }

  protected override parsedCallback(): void {
    this.lobby_connector.addEventListener('connect', (e) => {
      const socket = e.detail.try_reconnect
        ? new WebSocket(`${websocketPath()}/reconnect/${e.detail.nickname}/${e.detail.client_id}`)
        : new WebSocket(`${websocketPath()}/connect/${e.detail.nickname}`);
      let never_connected = true;
      socket.addEventListener('error', (e) => {
        console.error(e);
        this.tryConnectionAgain(
          `${never_connected ? 'Could not connect' : 'Connection was lost'}. Check your connection and try again.`
        );
      });
      socket.addEventListener('open', () => {
        never_connected = false;
        this.lobby_connector.classList.add('hide');
        this.watchSocket(socket);
        this.lobby.connect(e.detail.nickname, socket);
        this.game.exitGame();
      });
    });
    this.lobby.addEventListener('connection_lost', () => this.socketLost());
    setInterval((): void => {
      const now = Date.now();
      const resumed_from_freeze = now - this.last_tick_at > FROZEN_TICK_GAP_MS;
      this.last_tick_at = now;
      if (resumed_from_freeze) {
        this.last_message_at = now;
      } else if (now - this.last_message_at > SOCKET_SILENCE_MS) {
        this.socketLost();
      }
    }, WATCHDOG_TICK_MS);
    this.lobby.addEventListener('game_launched', async (e) => {
      if (await this.game.launchGame(e.detail, this.lobby.getSocket(), this.lobby.getConnectionMetadata())) {
        this.lobby.classList.add('hide');
      }
    });
    this.lobby.addEventListener('rejoin_game', async (e) => {
      if (await this.game.launchGame(e.detail, this.lobby.getSocket(), this.lobby.getConnectionMetadata(), true)) {
        this.lobby.enterGame();
      }
    });
    this.lobby.addEventListener('refresh_game_lobby', (e) => {
      if (!!this.game && !!e.detail && !!e.detail.game_id) {
        this.game?.refreshRoom(e.detail);
      }
    });
    this.game.addEventListener('exit_game', () => {
      this.game.exitGame();
      this.lobby.exitGame();
    });
    this.game.addEventListener('connection_lost', () => this.socketLost());
  }

  private watchSocket(socket: WebSocket): void {
    this.socket = socket;
    this.last_message_at = Date.now();
    socket.addEventListener('message', (): void => {
      this.last_message_at = Date.now();
    });
    socket.addEventListener('close', (e: CloseEvent): void => {
      if (this.socket === socket) {
        this.socketLost(e.code);
      }
    });
  }

  private socketLost(close_code?: number): void {
    const lost_socket = this.socket;
    const client_id = this.lobby.getConnectionMetadata().client_id;
    if (!lost_socket || this.reconnecting) {
      return;
    }
    this.socket = undefined;
    lost_socket.close(ABANDONED_CLOSE_CODE);
    if (close_code === REPLACED_CLOSE_CODE || !client_id) {
      this.tryConnectionAgain('Connection was lost or opened in another tab. Check your connection and try again.');
      return;
    }
    this.reconnecting = true;
    this.reconnecting_mask.classList.add('show');
    this.reconnect(client_id, 0);
  }

  private endReconnect(): void {
    this.reconnecting = false;
    this.reconnecting_mask.classList.remove('show');
  }

  private reconnect(client_id: number, attempt: number): void {
    if (attempt >= RECONNECT_DELAYS_MS.length) {
      this.endReconnect();
      this.tryConnectionAgain('Connection was lost. Check your connection and try again.');
      return;
    }
    setTimeout((): void => {
      const socket = new WebSocket(`${websocketPath()}/reconnect/${PREVIOUS_NICKNAME}/${client_id}`);
      let opened = false;
      socket.addEventListener('open', (): void => {
        opened = true;
        this.watchSocket(socket);
        this.lobby.setSocket(socket);
        this.game.swapSocket(socket);
        socket.addEventListener('message', (m: MessageEvent<string>): void => this.confirmRejoined(socket, m));
      });
      socket.addEventListener('close', (): void => {
        if (!opened) {
          this.reconnect(client_id, attempt + 1);
        }
      });
    }, RECONNECT_DELAYS_MS[attempt]);
  }

  private confirmRejoined(socket: WebSocket, m: MessageEvent<string>): void {
    if (this.socket !== socket || !this.reconnecting || !m.data.includes('lobby-you-joined')) {
      return;
    }
    if ((JSON.parse(m.data) as ServerMessage).kind === 'lobby-you-joined') {
      this.endReconnect();
      this.game.catchUp();
    }
  }

  private tryConnectionAgain(message: string): void {
    this.game.exitGame();
    this.lobby.classList.remove('hide');
    this.game.querySelectorAll('.dwg-dialog-box').forEach((el) => el.remove());
    this.lobby.querySelectorAll('.dwg-dialog-box').forEach((el) => el.remove());
    this.lobby.classList.remove('connected');
    this.lobby.classList.add('connector-open');
    this.lobby_connector.tryReconnecting(message, this.lobby.getConnectionMetadata());
  }
}

customElements.define('dwg-page-home', DwgPageHome);
