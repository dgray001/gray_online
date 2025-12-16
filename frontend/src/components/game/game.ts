import {DwgElement} from '../dwg_element';
import {ServerMessage, LobbyRoom, ConnectionMetadata, GameType, createMessage} from '../lobby/data_models';
import {apiPost} from '../../scripts/api';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';
import {capitalize, createLock, until} from '../../scripts/util';
import {MessageDialogData} from '../dialog_box/message_dialog/message_dialog';
import {getUrlParam} from '../../scripts/url';

import {Game, GameComponent, GameFromServer, serverResponseToGame, GamePlayer, GameViewer, GameHtmlTag} from './data_models';
import {handleMessage} from './message_handler';
import html from './game.html';

import './game.scss';
import './game_history_dialog/game_history_dialog';
import './game_info_dialog/game_info_dialog';
import './players_dialog/players_dialog';
import '../dialog_box/message_dialog/message_dialog';
import '../dialog_box/confirm_dialog/confirm_dialog';

const SERVER_PING_TIME = 3500; // time between game refreshes

/** Function to dispatch event that will show a dialog message */
export function messageDialog(data: MessageDialogData) {
  this.dispatchEvent(new CustomEvent('show_message_dialog', {detail: data, bubbles: true}));
}

export class DwgGame extends DwgElement {
  private client_name_string: HTMLSpanElement;
  private client_ping: HTMLSpanElement;
  private room_name: HTMLDivElement;
  private game_name: HTMLDivElement;
  private waiting_room: HTMLDivElement;
  private players_waiting: HTMLDivElement;
  private players_waiting_els = new Map<number, HTMLDivElement>();
  private game_container: HTMLDivElement;
  private game_el: GameComponent;
  private chatbox_container: HTMLDivElement;
  private chatbox: DwgChatbox;
  private open_chatbox_button: HTMLButtonElement;
  private button_game_info: HTMLButtonElement;
  private button_game_history: HTMLButtonElement;
  private button_room_players: HTMLButtonElement;
  private button_fullscreen: HTMLButtonElement;
  private maximize_img: HTMLImageElement;
  private minimize_img: HTMLImageElement;
  private button_exit: HTMLButtonElement;

  private bundles_attached = new Set<string>();
  private abort_controllers: AbortController[] = [];
  private launched: boolean;
  private socket: WebSocket;
  private connection_metadata: ConnectionMetadata;
  private game_id = 0;
  private player_id = -1;
  private is_player = false;
  private game: Game;
  private lobby_room: LobbyRoom;
  private ping_interval: NodeJS.Timeout;

  private chatbox_lock = createLock();

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

  isPlayer(): boolean {
    return this.is_player;
  }

  playerId(): number {
    return this.player_id;
  }

  getPlayers(): Map<number, GamePlayer>|undefined {
    return this.game?.game_base?.players;
  }

  getViewers(): Map<number, GameViewer>|undefined {
    return this.game?.game_base?.viewers;
  }

  addChat(message: ChatMessage, you_sent = false) {
    this.chatbox.addChat(message, you_sent);
  }

  getLaunched(): boolean {
    return this.launched;
  }

  getGameEl(): GameComponent|undefined {
    return this.game_el
  }

  getSocket(): WebSocket {
    return this.socket;
  }

  getConnectionMetadata(): ConnectionMetadata {
    return this.connection_metadata;
  }

  getGame(): Game|undefined {
    return this.game;
  }

  protected override parsedCallback(): void {
    this.minimize_img.classList.add('hide');
    document.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.toggleChatbox();
      }
    });
    this.chatbox.style.setProperty('--gray-color', 'rgba(220, 220, 220, 0.9)');
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      if (!this.socketActive()) {
        console.error('Trying to send chat with invalid socket')
        return;
      }
      const message = e.detail;
      if (message.message.startsWith('\\s')) {
        this.setSetting(message.message.slice(2).trim());
      } else if (message.message.startsWith('\\l')) {
        message.message = message.message.slice(2).trim();
        this.socketSend(createMessage(
          message.sender ?? `client-${this.connection_metadata.client_id}`,
          'lobby-chat',
          message.message,
          message.color,
        ));
      } else if (message.message.startsWith('\\u')) {
        message.message = message.message.slice(2).trim(); // TODO: implement
      } else if (message.message.startsWith('\\r')) {
        message.message = message.message.slice(2).trim();
        this.socketSend(createMessage(
          message.sender ?? `room-${this.connection_metadata.room_id}-${this.connection_metadata.client_id}`,
          'room-chat',
          message.message,
          message.color,
        ));
      } else {
        if (message.message.startsWith('\\g')) {
          message.message = message.message.slice(2).trim();
        }
        this.socketSend(createMessage(
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
      game_info.setData({game: this.game, room: this.lobby_room});
      this.appendChild(game_info);
    });
    this.button_game_history.addEventListener('click', () => {
      if (!this.launched) {
        return;
      }
      const game_history = document.createElement('dwg-game-history-dialog');
      game_history.setData({game: this.game_el, updates: this.game.game_base.updates ?? new Map()});
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

  private setSetting(s: string) {
    switch(s) {
      case 'help':
        this.chatbox.addChat({
          message: 'There are no commands; this feature is not yet implemented',
          color: 'gray',
        }, true);
        return;
      default:
        break;
    }
    const split = s.split('=', 2).map(s => s.trim());
    if (split.length !== 2) {
      return;
    }
    switch(split[0]) {
      default:
        this.chatbox.addChat({
          message: `Unrecognized setting key "${split[0]}", try "help" to see a list of commands`,
          color: 'gray',
        }, true);
        break;
    }
  }

  refreshRoom(room: LobbyRoom) {
    if (!room || !room.game_id || !this.connection_metadata?.room_id || room.room_id !== this.connection_metadata?.room_id) {
      console.error('Invalid game state');
      this.exitGame();
      return;
    }
    this.lobby_room = room;
    this.room_name.innerText = room.room_name;
    this.game_id = room.game_id;
    this.is_player = room.players.has(this.connection_metadata.client_id);
  }

  async launchGame(room: LobbyRoom, socket: WebSocket, connection_metadata: ConnectionMetadata, rejoining = false): Promise<boolean> {
    if ((this.launched && !rejoining) || !room || !room.game_id || !socket || socket.readyState !== WebSocket.OPEN ||
      !connection_metadata || !connection_metadata.client_id || !connection_metadata.room_id || room.room_id !== connection_metadata.room_id
    ) {
      console.log('Invalid state to launch game');
      return;
    }
    this.socket = socket;
    this.connection_metadata = connection_metadata;
    this.refreshRoom(room);
    this.client_name_string.innerText = this.connection_metadata.nickname;
    this.client_ping.innerText = ` (${this.connection_metadata.ping})`;
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

  socketSend(message: string) {
    if (!this.socketActive()) {
      return;
    }
    this.socket?.send(message);
  }

  private clientId(): number {
    return this.connection_metadata?.client_id ? this.connection_metadata.client_id : -1;
  }

  async refreshGame(): Promise<boolean> {
    const response = await apiPost<GameFromServer>(`lobby/games/get/${this.game_id}`, {
      client_id: this.clientId(),
      viewer: this.is_player ? "false" : "true",
    });
    if (!response.success || !response.result) {
      console.log(response.error_message);
      return false;
    }
    this.game = serverResponseToGame(response.result, this.clientId());
    let game_initialized = false;
    let waiting_room_initialized = false;
    const setGame = async (component: GameHtmlTag) => {
      if (!this.bundles_attached.has(component)) {
        const script = document.createElement('script');
        script.setAttribute('src', `/dist/${component.replace('dwg-', '').replace('-', '_')}.bundle.js?v=${getUrlParam('v')}`);
        script.async = false;
        let script_loaded = false;
        script.addEventListener('load', () => {
          script_loaded = true;
        });
        document.body.appendChild(script);
        await until(() => script_loaded);
        this.bundles_attached.add(component);
      }
      const game_el = document.createElement(component);
      this.game_container.replaceChildren(game_el);
      await until(() => game_el.fully_parsed);
      this.game_el = game_el; // assign after attaching to dom to keep TS happy
      this.player_id = -1;
      this.is_player = false;
      for (const [player_id, player] of this.game.players.entries()) {
        if (player.player.client_id === this.clientId()) {
          this.player_id = player_id;
          this.is_player = true;
          break;
        }
      }
      this.game_el.initialize(this, this.game).then(() => {
        game_initialized = true;
        if (waiting_room_initialized) {
          this.socketSend(createMessage(
            `client-${this.clientId()}`,
            'game-connected',
            '',
            this.game_id.toString(),
          ));
          this.launched = true;
        }
      });
      game_el.addEventListener('game_update', (e: CustomEvent<string>) => {
        if (this.game.game_base.game_ended) {
          console.log('Game already over');
          return;
        }
        console.log('Sending game update', e.detail);
        this.socketSend(e.detail);
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
      case GameType.TEST_GAME:
        await setGame('dwg-test-game');
        break;
      case GameType.UNSPECIFIED:
      default:
        throw new Error(`Unknown game type: ${this.game.game_base.game_type}`);
    }
    this.game_name.innerText = capitalize(GameType[this.game.game_base.game_type].toLowerCase().replace('_', ' '));
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
      this.socketSend(createMessage(
        `client-${this.clientId()}`,
        'game-connected',
        '',
        this.game_id.toString(),
      ));
      this.ping_interval = setInterval(() => {
        if (!this.launched) {
          return;
        }
        this.pingServer();
      }, SERVER_PING_TIME);
      this.launched = true;
      return true;
    }
    return false;
  }

  playerDisconnected(player_id: number) {
    const el = this.players_waiting_els.get(player_id);
    if (!!el) {
      el.innerText = 'Connecting ...';
      el.classList.remove('connected');
    }
    const player = this.getPlayers()?.get(player_id);
    if (!!player) {
      player.connected = false;
    }
  }

  playerConnected(player_id: number) {
    const el = this.players_waiting_els.get(player_id);
    if (!!el) {
      el.innerText = 'Connected';
      el.classList.add('connected');
    }
    const player = this.getPlayers()?.get(player_id);
    if (!!player) {
      player.connected = true;
    }
  }

  startGame() {
    if (!this.game) {
      return;
    }
    this.game.game_base.game_started = true;
    this.waiting_room.classList.add('hide');
    this.game_container.classList.add('show');
  }

  exitGame() {
    this.launched = false;
    this.classList.remove('show');
    this.socketSend(createMessage(
      `client-${this.clientId()}`,
      'game-exit',
    ));
    this.socket = undefined;
    this.connection_metadata = undefined;
    this.lobby_room = undefined;
    this.game_id = 0;
    this.game = undefined;
    clearInterval(this.ping_interval);
  }

  pingServer() {
    if (!this.socketActive()) {
      this.dispatchEvent(new CustomEvent<string>('connection_lost', {detail: 'Lost connection in game'}));
      return;
    }
    if (this.game?.game_base?.game_ended) {
      // nothing to do
    } else if (this.game?.game_base?.game_started) {
      this.socketSend(createMessage(
        `client-${this.clientId()}`,
        'game-resend-last-update',
      ));
    } else {
      this.socketSend(createMessage(
        `client-${this.clientId()}`,
        'game-resend-waiting-room',
      ));
    }
  }
}

if (!customElements.get('dwg-game')) {
  customElements.define('dwg-game', DwgGame);
}

declare global {
  interface HTMLElementTagNameMap {
    'dwg-game': DwgGame;
  }
}
