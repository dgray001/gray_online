import {DwgElement} from '../dwg_element';
import {ServerMessage, LobbyRoom, ConnectionMetadata, GameType, createMessage} from '../lobby/data_models';
import {Game, GameComponent, GameFromServer, serverResponseToGame} from './data_models';
import {apiPost} from '../../scripts/api';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';
import {capitalize} from '../../scripts/util';

import {handleMessage} from './message_handler';
import html from './game.html';

import './game.scss';
import './fiddlesticks/fiddlesticks';
import './game_history_dialog/game_history_dialog';

export class DwgGame extends DwgElement {
  client_name_string: HTMLSpanElement;
  client_ping: HTMLSpanElement;
  room_name: HTMLDivElement;
  game_name: HTMLDivElement;
  waiting_room: HTMLDivElement;
  players_waiting: HTMLDivElement;
  players_waiting_els = new Map<number, HTMLDivElement>();
  game_container: HTMLDivElement;
  game_el: GameComponent;
  chatbox_container: HTMLDivElement;
  chatbox: DwgChatbox;
  button_game_history: HTMLButtonElement;
  button_room_players: HTMLButtonElement;
  button_exit: HTMLButtonElement;

  launched: boolean;
  socket: WebSocket;
  connection_metadata: ConnectionMetadata;
  game: Game;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('client_name_string');
    this.configureElement('client_ping');
    this.configureElement('room_name');
    this.configureElement('game_name');
    this.configureElement('waiting_room');
    this.configureElement('players_waiting');
    this.configureElement('game_container');
    this.configureElement('chatbox_container');
    this.configureElement('chatbox');
    this.configureElement('button_game_history');
    this.configureElement('button_room_players');
    this.configureElement('button_exit');
  }

  protected override parsedCallback(): void {
    document.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.chatbox_container.classList.toggle('showing');
        this.chatbox.classList.toggle('show');
      }
    });
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      const message = e.detail;
      this.socket.send(createMessage(
        message.sender ?? `game-${this.connection_metadata.client_id}`,
        'game-chat',
        message.message,
        message.color,
      ));
    });
    this.button_game_history.addEventListener('click', () => {
      if (!this.launched) {
        return;
      }
      const game_history = document.createElement('dwg-game-history-dialog');
      console.log(this.game.game_base.updates);
      game_history.setData({updates: this.game.game_base.updates ?? new Map()});
      this.appendChild(game_history);
    });
    this.button_room_players.addEventListener('click', () => {
      // TODO: game players dialog box
    });
    this.button_exit.addEventListener('click', () => {
      // TODO: add confirm dialog
      this.dispatchEvent(new Event("exit_game"));
    });
    setInterval(() => {
      this.pingServer();
    }, 2500);
  }

  async launchGame(lobby: LobbyRoom, socket: WebSocket, connection_metadata: ConnectionMetadata): Promise<boolean> {
    if (this.launched || !lobby || !lobby.game_id || !socket || socket.readyState !== WebSocket.OPEN ||
      !connection_metadata || !connection_metadata.client_id || !connection_metadata.room_id || lobby.room_id !== connection_metadata.room_id
    ) {
      // TODO: try to sync state
      return;
    }
    this.socket = socket;
    this.connection_metadata = connection_metadata;
    this.client_name_string.innerText = this.connection_metadata.nickname;
    this.client_ping.innerText = ` (${this.connection_metadata.ping})`;
    this.room_name.innerText = lobby.room_name;
    try {
      this.socket.addEventListener('message', (m) => {
        try {
          const message = JSON.parse(m.data) as ServerMessage;
          handleMessage(this, message);
        } catch(e) {
          console.log('Error parsing message: ', m, e)
        }
      });
      const response = await apiPost<GameFromServer>(`lobby/games/get/${lobby.game_id}`, {
        client_id: this.connection_metadata.client_id,
        viewer: "false", // TODO: add viewer support here somehow
      });
      if (response.success && !!response.result) {
        this.game = serverResponseToGame(response.result, this.connection_metadata.client_id);
        const setGame = (component: string) => {
          const game_el = document.createElement(component);
          this.game_container.appendChild(game_el);
          this.game_el = game_el as unknown as GameComponent;
          this.game_el.initialize(this.game, this.connection_metadata.client_id);
          game_el.addEventListener('game_update', (e: CustomEvent<string>) => {
            this.socket.send(e.detail);
          });
        };
        switch(this.game.game_base.game_type) {
          case GameType.FIDDLESTICKS:
            setGame('dwg-fiddlesticks');
            break;
          case GameType.UNSPECIFIED:
          default:
            throw new Error(`Unknown game type: ${this.game.game_base.game_type}`);
        }
        this.game_name.innerText = capitalize(GameType[this.game.game_base.game_type].toLowerCase());
        for (const [id, player] of this.game.game_base.players) {
          const el = document.createElement('div');
          const name = document.createElement('div');
          const status = document.createElement('div');
          el.appendChild(name);
          el.appendChild(status);
          el.classList.add('player-waiting');
          name.innerText = player.nickname;
          status.innerText = player.connected ? 'Connected' : 'Connecting ...';
          if (player.connected) {
            status.classList.add('connected');
          }
          this.players_waiting_els.set(id, status);
          this.players_waiting.appendChild(el);
        }
        this.classList.add('show');
        this.socket.send(createMessage(
          `client-${this.connection_metadata.client_id}`,
          'game-connected',
          '',
          lobby.game_id.toString(),
        ));
        this.launched = true;
        return true;
      } else {
        console.log(response.error_message);
      }
    } catch(e) {
      console.log(e);
    }
    return false;
  }

  socketActive() {
    return !!this.socket && this.socket.readyState == WebSocket.OPEN;
  }

  pingServer() {
    if (!this.socketActive()) {
      return;
    }
    if (this.game?.game_base?.game_started) {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'game-resend-last-update',
      ));
    } else {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'game-resend-waiting-room',
      ));
    }
  }
}

customElements.define('dwg-game', DwgGame);
