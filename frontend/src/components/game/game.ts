import {DwgElement} from '../dwg_element';
import {ServerMessage, LobbyRoom, ConnectionMetadata, GameType, createMessage} from '../lobby/data_models';
import {apiPost} from '../../scripts/api';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';
import {capitalize, createLock, until} from '../../scripts/util';
import {MessageDialogData} from '../dialog_box/message_dialog/message_dialog';

import {Game, GameComponent, GameFromServer, serverResponseToGame} from './data_models';
import {handleMessage} from './message_handler';
import html from './game.html';

import './game.scss';
import './games/fiddlesticks/fiddlesticks';
import './games/euchre/euchre';
import './games/risq/risq';
import './game_history_dialog/game_history_dialog';
import './game_info_dialog/game_info_dialog';
import './players_dialog/players_dialog';
import '../dialog_box/confirm_dialog/confirm_dialog';

/** Function to dispatch event that will show a dialog message */
export function messageDialog(data: MessageDialogData) {
  this.dispatchEvent(new CustomEvent('show_message_dialog', {'detail': data, bubbles: true}));
}

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
  open_chatbox_button: HTMLButtonElement;
  button_game_info: HTMLButtonElement;
  button_game_history: HTMLButtonElement;
  button_room_players: HTMLButtonElement;
  button_fullscreen: HTMLButtonElement;
  maximize_img: HTMLImageElement;
  minimize_img: HTMLImageElement;
  button_exit: HTMLButtonElement;

  abort_controllers: AbortController[] = [];
  launched: boolean;
  socket: WebSocket;
  connection_metadata: ConnectionMetadata;
  game_id = 0;
  player_id = -1;
  is_player = false;
  game: Game;
  lobby_room: LobbyRoom;

  chatbox_lock = createLock();

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
    this.configureElement('open_chatbox_button');
    this.configureElement('button_game_info');
    this.configureElement('button_game_history');
    this.configureElement('button_room_players');
    this.configureElement('button_fullscreen');
    this.configureElement('maximize_img');
    this.configureElement('minimize_img');
    this.configureElement('button_exit');
  }

  protected override parsedCallback(): void {
    this.minimize_img.classList.add('hide');
    document.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.toggleChatbox();
      }
    });
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      if (!this.socketActive()) {
        console.error('Trying to send chat with invalid socket')
        return;
      }
      const message = e.detail;
      if (message.message.startsWith('\\l')) {
        message.message = message.message.slice(2).trim();
        this.socket.send(createMessage(
          message.sender ?? `client-${this.connection_metadata.client_id}`,
          'lobby-chat',
          message.message,
          message.color,
        ));
      } else if (message.message.startsWith('\\u')) {
        message.message = message.message.slice(2).trim(); // TODO: implement
      } else if (message.message.startsWith('\\r')) {
        message.message = message.message.slice(2).trim();
        this.socket.send(createMessage(
          message.sender ?? `room-${this.connection_metadata.room_id}-${this.connection_metadata.client_id}`,
          'room-chat',
          message.message,
          message.color,
        ));
      } else {
        if (message.message.startsWith('\\g')) {
          message.message = message.message.slice(2).trim();
        }
        this.socket.send(createMessage(
          message.sender ?? `game-${this.connection_metadata.client_id}`,
          'game-chat',
          message.message,
          message.color,
        ));
      }
    });
    this.open_chatbox_button.addEventListener('click', () => {
      this.toggleChatbox();
    });
    this.button_game_info.addEventListener('click', () => {
      const game_info = document.createElement('dwg-game-info-dialog');
      game_info.setData({});
      this.appendChild(game_info);
    });
    this.button_game_history.addEventListener('click', () => {
      if (!this.launched) {
        return;
      }
      const game_history = document.createElement('dwg-game-history-dialog');
      game_history.setData({updates: this.game.game_base.updates ?? new Map()});
      this.appendChild(game_history);
    });
    this.button_room_players.addEventListener('click', () => {
      const players_dialog = document.createElement('dwg-players-dialog');
      players_dialog.setData({
        players: [...this.game.game_base.players.values()].sort((a, b) => a.player_id - b.player_id),
        lobby_players: this.lobby_room.players,
        room_id: this.lobby_room.room_id,
      });
      this.appendChild(players_dialog);
    });
    this.button_fullscreen.addEventListener('click', () => {
      if (!!document.fullscreenElement) {
        document.exitFullscreen();
        this.maximize_img.classList.remove('hide');
        this.minimize_img.classList.add('hide');
      } else {
        this.requestFullscreen();
        this.maximize_img.classList.add('hide');
        this.minimize_img.classList.remove('hide');
      }
    });
    this.button_exit.addEventListener('click', () => {
      const confirm_dialog = document.createElement('dwg-confirm-dialog');
      confirm_dialog.setData({question: 'Are you sure you want to exit?'});
      confirm_dialog.addEventListener('confirmed', () => {
        this.dispatchEvent(new Event('exit_game'));
      });
      this.appendChild(confirm_dialog);
    });
    setInterval(() => {
      this.pingServer();
    }, 2500);
  }

  toggleChatbox() {
    this.chatbox_lock(async () => {
      this.chatbox_container.classList.toggle('showing');
      this.chatbox.classList.toggle('show');
      if (this.chatbox.classList.contains('show')) {
        this.chatbox.focus();
      }
    });
  }

  setPadding(padding: string) {
    this.game_container.style.setProperty('--padding', padding);
  }

  async launchGame(lobby: LobbyRoom, socket: WebSocket, connection_metadata: ConnectionMetadata, rejoining = false): Promise<boolean> {
    if ((this.launched && !rejoining) || !lobby || !lobby.game_id || !socket || socket.readyState !== WebSocket.OPEN ||
      !connection_metadata || !connection_metadata.client_id || !connection_metadata.room_id || lobby.room_id !== connection_metadata.room_id
    ) {
      console.log('Invalid state to launch game');
      return;
    }
    this.lobby_room = lobby;
    this.socket = socket;
    this.connection_metadata = connection_metadata;
    this.client_name_string.innerText = this.connection_metadata.nickname;
    this.client_ping.innerText = ` (${this.connection_metadata.ping})`;
    this.room_name.innerText = lobby.room_name;
    this.game_id = lobby.game_id;
    this.is_player = lobby.players.has(this.connection_metadata.client_id);
    try {
      for (const abort_controller of this.abort_controllers) {
        abort_controller.abort();
      }
      const abort_controller = new AbortController();
      this.abort_controllers.push(abort_controller);
      this.socket.addEventListener('message', (m) => {
        try {
          const message = JSON.parse(m.data) as ServerMessage;
          handleMessage(this, message);
        } catch(e) {
          console.log('Error parsing message: ', m, e)
        }
      }, {signal: abort_controller.signal});
      return await this.refreshGame();
    } catch(e) {
      console.log(e);
    }
    return false;
  }

  socketActive() {
    return !!this.socket && this.socket.readyState == WebSocket.OPEN;
  }

  async refreshGame() {
    const response = await apiPost<GameFromServer>(`lobby/games/get/${this.game_id}`, {
      client_id: this.connection_metadata.client_id,
      viewer: this.is_player ? "false" : "true",
    });
    if (!response.success || !response.result) {
      console.log(response.error_message);
      return false;
    }
    this.game = serverResponseToGame(response.result, this.connection_metadata.client_id);
    let game_initialized = false;
    let waiting_room_initialized = false;
    const setGame = async (component: 'dwg-fiddlesticks' | 'dwg-euchre' | 'dwg-risq') => {
      const game_el = document.createElement(component);
      this.game_container.replaceChildren(game_el);
      await until(() => game_el.fully_parsed);
      this.game_el = game_el; // assign after attaching to dom to keep TS happy
      this.player_id = -1;
      this.is_player = false;
      for (const [player_id, player] of this.game.players.entries()) {
        if (player.player.client_id === this.connection_metadata.client_id) {
          this.player_id = player_id;
          this.is_player = true;
          break;
        }
      }
      this.game_el.initialize(this, this.game).then(() => {
        game_initialized = true;
        if (waiting_room_initialized) {
          this.socket.send(createMessage(
            `client-${this.connection_metadata.client_id}`,
            'game-connected',
            '',
            this.game_id.toString(),
          ));
          this.launched = true;
          return true;
        }
      });
      game_el.addEventListener('game_update', (e: CustomEvent<string>) => {
        if (this.game.game_base.game_ended) {
          console.log('Game already over');
          return;
        }
        console.log('Sending game update', e.detail);
        this.socket.send(e.detail);
      });
    };
    switch(this.game.game_base.game_type) {
      case GameType.FIDDLESTICKS:
        await setGame('dwg-fiddlesticks');
        break;
      case GameType.EUCHRE:
        await setGame('dwg-euchre');
        break;
      case GameType.RISQ:
        await setGame('dwg-risq');
        break;
      case GameType.UNSPECIFIED:
      default:
        throw new Error(`Unknown game type: ${this.game.game_base.game_type}`);
    }
    this.game_name.innerText = capitalize(GameType[this.game.game_base.game_type].toLowerCase());
    if (this.game.game_base.game_started) {
      this.startGame();
    } else {
      this.waiting_room.classList.remove('hide');
      this.game_container.classList.remove('show');
      this.players_waiting_els.clear();
      this.players_waiting.replaceChildren();
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
    }
    this.classList.add('show');
    waiting_room_initialized = true;
    if (game_initialized) {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'game-connected',
        '',
        this.game_id.toString(),
      ));
      this.launched = true;
      return true;
    }
  }

  startGame() {
    this.game.game_base.game_started = true;
    this.waiting_room.classList.add('hide');
    this.game_container.classList.add('show');
  }

  pingServer() {
    if (!this.socketActive()) {
      return;
    }
    if (this.game?.game_base?.game_ended) {
      // nothing to do
    } else if (this.game?.game_base?.game_started) {
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
