import { DwgElement } from '../dwg_element';
import { DialogSize } from '../dialog_box/dialog_box';
import type { ServerMessage, LobbyRoom, ConnectionMetadata } from '../lobby/data_models';
import { GameType, createMessage } from '../lobby/data_models';
import { apiPost } from '../../scripts/api';
import { Sounds } from '../../sounds/Sounds';
import type { ChatMessage, DwgChatbox } from '../chatbox/chatbox';
import { capitalize, createLock, until } from '../../scripts/util';
import type { MessageDialogData } from '../dialog_box/message_dialog/message_dialog';
import { getUrlParam } from '../../scripts/url';

import type {
  Game,
  GameComponent,
  GameFromServer,
  GamePlayer,
  GameViewer,
  GameHtmlTag,
  UpdateMessage,
} from './data_models';
import { serverResponseToGame } from './data_models';
import { handleMessage } from './message_handler';
import html from './game.html';

import './game.scss';
import './game_history_dialog/game_history_dialog';
import './game_info_dialog/game_info_dialog';
import './players_dialog/players_dialog';
import '../dialog_box/message_dialog/message_dialog';
import '../dialog_box/confirm_dialog/confirm_dialog';
import '../dialog_box/settings_dialog/settings_dialog';

const SERVER_PING_TIME = 3500; // time between game refreshes

/** Function to dispatch event that will show a dialog message */
export function messageDialog(this: EventTarget, data: MessageDialogData): void {
  this.dispatchEvent(new CustomEvent('show_message_dialog', { detail: data, bubbles: true }));
}

export class DwgGame extends DwgElement {
  private client_name_string!: HTMLSpanElement;
  private client_ping!: HTMLSpanElement;
  private room_name!: HTMLDivElement;
  private game_name!: HTMLDivElement;
  private waiting_room!: HTMLDivElement;
  private players_waiting!: HTMLDivElement;
  private players_waiting_els = new Map<number, HTMLDivElement>();
  private game_container!: HTMLDivElement;
  private game_el!: GameComponent & HTMLElement;
  private chatbox_container!: HTMLDivElement;
  private chatbox!: DwgChatbox;
  private open_chatbox_button!: HTMLButtonElement;
  private button_game_info!: HTMLButtonElement;
  private button_game_history!: HTMLButtonElement;
  private button_room_players!: HTMLButtonElement;
  private button_settings!: HTMLButtonElement;
  private button_fullscreen!: HTMLButtonElement;
  private maximize_img!: HTMLImageElement;
  private minimize_img!: HTMLImageElement;
  private button_exit!: HTMLButtonElement;

  private bundles_attached = new Set<string>();
  private abort_controller?: AbortController;
  private launched = false;
  private socket?: WebSocket;
  private connection_metadata?: ConnectionMetadata;
  private game_id = 0;
  private player_id = -1;
  private is_player = false;
  private game?: Game;
  private lobby_room?: LobbyRoom;
  private ping_interval?: NodeJS.Timeout;

  message_error_count = 0;
  running_updates = false;
  refreshing?: Promise<boolean>;
  private resyncing?: Promise<void>;
  update_queue: UpdateMessage[] = [];
  private pending_start = false;

  private chatbox_lock = createLock();

  private handleKeyup = (e: KeyboardEvent): void => {
    if (e.key === 'Enter') {
      if (!this.isOpen()) {
        return;
      }
      this.toggleChatbox();
    }
  };

  constructor() {
    super();
    this.html_string = html;
    this.addEventListener('show_message_dialog', (event) => {
      event.stopPropagation();
      const dialog = document.createElement('dwg-message-dialog');
      dialog.setData({
        ...event.detail,
        size:
          event.detail.size ??
          (this.game?.game_base.game_type === GameType.FIDDLESTICKS ? DialogSize.SMALL : undefined),
      });
      this.appendChild(dialog);
    });
    this.configureElements(
      'client_name_string',
      'client_ping',
      'room_name',
      'game_name',
      'waiting_room',
      'players_waiting',
      'game_container',
      'chatbox_container',
      'chatbox',
      'open_chatbox_button',
      'button_game_info',
      'button_game_history',
      'button_room_players',
      'button_settings',
      'button_fullscreen',
      'maximize_img',
      'minimize_img',
      'button_exit'
    );
  }

  isPlayer(): boolean {
    return this.is_player;
  }

  playerId(): number {
    return this.player_id;
  }

  getPlayers(): Map<number, GamePlayer> | undefined {
    return this.game?.game_base?.players;
  }

  getViewers(): Map<number, GameViewer> | undefined {
    return this.game?.game_base?.viewers;
  }

  addChat(message: ChatMessage, you_sent = false): void {
    this.chatbox.addChat(message, you_sent);
  }

  getLaunched(): boolean {
    return this.launched;
  }

  isOpen(): boolean {
    return this.classList.contains('show');
  }

  getGameEl(): GameComponent | undefined {
    return this.game_el;
  }

  getSocket(): WebSocket | undefined {
    return this.socket;
  }

  getConnectionMetadata(): ConnectionMetadata | undefined {
    return this.connection_metadata;
  }

  getGame(): Game | undefined {
    return this.game;
  }

  protected override parsedCallback(): void {
    this.minimize_img.classList.add('hide');
    document.addEventListener('keyup', this.handleKeyup);
    this.chatbox.style.setProperty('--gray-color', 'rgba(220, 220, 220, 0.9)');
    this.chatbox.addEventListener('chat_sent', (e) => {
      if (!this.socketActive()) {
        console.error('Trying to send chat with invalid socket');
        return;
      }
      const message = e.detail;
      if (message.message.startsWith('\\s')) {
        this.setSetting(message.message.slice(2).trim());
      } else if (message.message.startsWith('\\l')) {
        message.message = message.message.slice(2).trim();
        this.socketSend(
          createMessage(
            message.sender ?? `client-${this.connection_metadata?.client_id}`,
            'lobby-chat',
            message.message,
            message.color
          )
        );
      } else if (message.message.startsWith('\\u')) {
        message.message = message.message.slice(2).trim(); // TODO: implement
      } else if (message.message.startsWith('\\r')) {
        message.message = message.message.slice(2).trim();
        this.socketSend(
          createMessage(
            message.sender ?? `room-${this.connection_metadata?.room_id}-${this.connection_metadata?.client_id}`,
            'room-chat',
            message.message,
            message.color
          )
        );
      } else {
        if (message.message.startsWith('\\g')) {
          message.message = message.message.slice(2).trim();
        }
        this.socketSend(
          createMessage(
            message.sender ?? `game-${this.connection_metadata?.client_id}`,
            'game-chat',
            message.message,
            message.color
          )
        );
        Sounds.playTaunt(message.message);
      }
    });
    this.open_chatbox_button.addEventListener('click', () => {
      this.toggleChatbox();
    });
    this.button_game_info.addEventListener('click', () => {
      if (!this.game || !this.lobby_room) {
        return;
      }
      const game_info = document.createElement('dwg-game-info-dialog');
      game_info.setData({
        game: this.game,
        room: this.lobby_room,
        size: this.game.game_base.game_type === GameType.FIDDLESTICKS ? DialogSize.LARGE : undefined,
      });
      this.appendChild(game_info);
    });
    this.button_game_history.addEventListener('click', () => {
      if (!this.launched || !this.game) {
        return;
      }
      const game_history = document.createElement('dwg-game-history-dialog');
      game_history.setData({
        size: this.game.game_base.game_type === GameType.FIDDLESTICKS ? DialogSize.XLARGE : undefined,
        game: this.game_el,
        updates: this.game.game_base.updates ?? new Map(),
      });
      this.appendChild(game_history);
    });
    this.button_room_players.addEventListener('click', () => {
      if (!this.game || !this.lobby_room) {
        return;
      }
      const players_dialog = document.createElement('dwg-players-dialog');
      players_dialog.setData({
        size: this.game.game_base.game_type === GameType.FIDDLESTICKS ? DialogSize.XLARGE : undefined,
        players: [...this.game.game_base.players.values()].sort((a, b) => a.player_id - b.player_id),
        lobby_players: this.lobby_room.players,
        room_id: this.lobby_room.room_id,
      });
      this.appendChild(players_dialog);
    });
    this.button_settings.addEventListener('click', () => {
      if (!this.game) {
        return;
      }
      const settings_dialog = document.createElement('dwg-settings-dialog');
      settings_dialog.setData({ game_type: this.game.game_base.game_type });
      this.appendChild(settings_dialog);
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
      confirm_dialog.setData({
        question: 'Are you sure you want to exit?',
        size: this.game?.game_base.game_type === GameType.FIDDLESTICKS ? DialogSize.SMALL : undefined,
      });
      confirm_dialog.addEventListener('confirmed', () => {
        this.dispatchEvent(new Event('exit_game'));
      });
      this.appendChild(confirm_dialog);
    });
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    document.removeEventListener('keyup', this.handleKeyup);
  }

  toggleChatbox(): void {
    if (!this.isOpen()) {
      return;
    }
    this.chatbox_lock(async () => {
      this.chatbox_container.classList.toggle('showing');
      this.chatbox.classList.toggle('show');
      if (this.chatbox.classList.contains('show')) {
        this.chatbox.focus();
      }
    });
  }

  closeChatbox(): void {
    this.chatbox_lock(async () => {
      this.chatbox_container.classList.remove('showing');
      this.chatbox.classList.remove('show');
    });
  }

  setPadding(padding: string): void {
    this.game_container.style.setProperty('--padding', padding);
  }

  private setSetting(s: string): void {
    switch (s) {
      case 'help':
        this.chatbox.addChat(
          {
            message: 'There are no commands; this feature is not yet implemented',
            color: 'gray',
          },
          true
        );
        return;
      default:
        break;
    }
    const split = s.split('=', 2).map((s) => s.trim());
    if (split.length !== 2) {
      return;
    }
    switch (split[0]) {
      default:
        this.chatbox.addChat(
          {
            message: `Unrecognized setting key "${split[0]}", try "help" to see a list of commands`,
            color: 'gray',
          },
          true
        );
        break;
    }
  }

  refreshRoom(room: LobbyRoom): void {
    if (
      !room ||
      !room.game_id ||
      !this.connection_metadata?.room_id ||
      room.room_id !== this.connection_metadata?.room_id
    ) {
      console.error('Invalid game state');
      this.exitGame();
      return;
    }
    this.lobby_room = room;
    this.room_name.innerText = room.room_name;
    this.game_id = room.game_id;
    this.is_player = !!this.connection_metadata.client_id && room.players.has(this.connection_metadata.client_id);
  }

  async launchGame(
    room: LobbyRoom | undefined,
    socket: WebSocket | undefined,
    connection_metadata: ConnectionMetadata | undefined,
    rejoining = false
  ): Promise<boolean> {
    if (
      (this.launched && !rejoining) ||
      !room ||
      !room.game_id ||
      !socket ||
      socket.readyState !== WebSocket.OPEN ||
      !connection_metadata ||
      !connection_metadata.client_id ||
      !connection_metadata.room_id ||
      room.room_id !== connection_metadata.room_id
    ) {
      console.log('Invalid state to launch game: ', {
        launched: this.launched,
        rejoining,
        room,
        socket,
        connection_metadata,
      });
      return false;
    }
    this.socket = socket;
    this.connection_metadata = connection_metadata;
    this.refreshRoom(room);
    this.client_name_string.innerText = this.connection_metadata.nickname;
    this.client_ping.innerText = ` (${this.connection_metadata.ping})`;
    try {
      this.listenToSocket(socket);
      return await this.refreshGame();
    } catch (e) {
      console.log(e);
    }
    return false;
  }

  private listenToSocket(socket: WebSocket): void {
    this.socket = socket;
    this.abort_controller?.abort();
    this.abort_controller = new AbortController();
    socket.addEventListener(
      'message',
      (m) => {
        try {
          handleMessage(this, JSON.parse(m.data) as ServerMessage);
        } catch (e) {
          console.log('Error parsing message: ', m, e);
        }
      },
      { signal: this.abort_controller.signal }
    );
  }

  swapSocket(socket: WebSocket): void {
    if (this.socket) {
      this.listenToSocket(socket);
    }
  }

  /** Brings a mounted game up to date after a reconnect under the same client id */
  catchUp(): void {
    if (!this.launched || !this.game) {
      return;
    }
    if (this.game.game_base.game_started && !this.game.game_base.persistant_history) {
      void this.resyncState();
    }
    this.pingServer();
  }

  socketActive(): boolean {
    return !!this.socket && this.socket.readyState === WebSocket.OPEN;
  }

  socketSend(message: string): void {
    if (!this.socketActive()) {
      return;
    }
    this.socket?.send(message);
  }

  private clientId(): number {
    return this.connection_metadata?.client_id ? this.connection_metadata.client_id : -1;
  }

  private handleStartupError(game_el: GameComponent & HTMLElement, error: unknown): void {
    if (!this.isConnected || !this.game || !game_el.isConnected || this.game_el !== game_el) {
      return;
    }
    if (error instanceof DOMException && error.name === 'AbortError') {
      return;
    }
    console.error('Game initialization failed:', error);
    this.launched = false;
    clearInterval(this.ping_interval);
    game_el.remove();
    messageDialog.call(this, { message: 'Unable to initialize the game. Refresh the page to try again.' });
  }

  refreshGame(): Promise<boolean> {
    return (this.refreshing ??= this.refreshGameSnapshot()
      .catch((error: unknown): boolean => {
        console.error('Error refreshing game:', error);
        return false;
      })
      .finally((): void => {
        this.refreshing = undefined;
      }));
  }

  async recoverGame(): Promise<boolean> {
    if (this.refreshing) {
      return this.refreshing;
    }
    const game_id = this.game_id;
    const socket = this.socket;
    this.game_el?.remove();
    const refreshed = await this.refreshGame();
    if (!refreshed && this.isConnected && this.game_id === game_id && this.socket === socket) {
      messageDialog.call(this, { message: 'Unable to refresh game state. Refresh the page to reconnect.' });
    }
    return refreshed;
  }

  /** Repairs a missed setting update by applying the current state as a plain replacement, keeping the board mounted */
  resyncState(): Promise<void> {
    return (this.resyncing ??= this.applyFreshState().finally((): void => {
      this.resyncing = undefined;
    }));
  }

  private async applyFreshState(): Promise<void> {
    const game_el = this.game_el;
    const applied_id = this.game?.game_base.last_applied_update_id;
    const response = await apiPost<GameFromServer>(`lobby/games/get/${this.game_id}`, {
      client_id: this.clientId(),
      token: this.connection_metadata?.token,
      viewer: this.is_player ? 'false' : 'true',
    });
    const game_base = this.game?.game_base;
    const fresh = response.success ? response.result : undefined;
    // anything newer than the request means the heartbeat will compare again
    if (!fresh || !game_base || this.game_el !== game_el || this.refreshing) {
      return;
    }
    if (game_base.last_applied_update_id !== applied_id || fresh.game_base.latest_update_id !== applied_id) {
      return;
    }
    game_base.settings_rev = fresh.game_base.settings_rev;
    await game_el.gameUpdate({ update_id: 0, kind: 'resync', content: { game: fresh } });
  }

  private async refreshGameSnapshot(): Promise<boolean> {
    const game_id = this.game_id;
    const socket = this.socket;
    const response = await apiPost<GameFromServer>(`lobby/games/get/${this.game_id}`, {
      client_id: this.clientId(),
      token: this.connection_metadata?.token,
      viewer: this.is_player ? 'false' : 'true',
    });
    if (!this.isConnected || this.game_id !== game_id || this.socket !== socket) {
      return false;
    }
    if (!response.success) {
      console.error(response.error_message);
      return false;
    }
    const new_game = serverResponseToGame(response.result, this.clientId());
    this.game = new_game;
    this.running_updates = false;
    this.update_queue = [];
    let game_initialized = false;
    const set_game = async (component: GameHtmlTag): Promise<void> => {
      if (!this.bundles_attached.has(component)) {
        const script = document.createElement('script');
        script.setAttribute(
          'src',
          `${window.location.origin}/dist/${component.replace('dwg-', '').replace(/-/g, '_')}.bundle.js?v=${getUrlParam('v')}`
        );
        script.async = false;
        await new Promise<void>((resolve, reject) => {
          script.addEventListener('load', () => resolve());
          script.addEventListener('error', () => {
            script.remove();
            reject(new Error(`Failed to load game bundle ${component}`));
          });
          document.body.appendChild(script);
        });
        this.bundles_attached.add(component);
      }
      await customElements.whenDefined(component);
      const game_el = document.createElement(component);
      this.game_container.replaceChildren(game_el);
      await until(() => game_el.fully_parsed);
      this.game_el = game_el; // assign after attaching to dom to keep TS happy
      this.player_id = -1;
      this.is_player = false;
      for (const [player_id, player] of new_game.players.entries()) {
        if (player.player.client_id === this.clientId()) {
          this.player_id = player_id;
          this.is_player = true;
          break;
        }
      }
      game_el.addEventListener('game_initialization_failed', (e: CustomEvent<unknown>): void =>
        this.handleStartupError(game_el, e.detail)
      );
      try {
        await this.game_el.initialize(this, new_game);
        if (!this.isConnected || this.game !== new_game || !game_el.isConnected || this.game_el !== game_el) {
          return;
        }
        game_initialized = true;
      } catch (error) {
        this.handleStartupError(game_el, error);
      }
      game_el.addEventListener('game_update', (e) => {
        if (new_game.game_base.game_ended) {
          console.log('Game already over');
          return;
        }
        console.log('Sending game update', e.detail);
        this.socketSend(e.detail);
      });
    };
    switch (new_game.game_base.game_type) {
      case GameType.FIDDLESTICKS:
        await set_game('dwg-fiddlesticks');
        break;
      case GameType.EUCHRE:
        await set_game('dwg-euchre');
        break;
      case GameType.EGYPTIAN_RAT_CRAP:
        await set_game('dwg-egyptian-rat-crap');
        break;
      case GameType.RISQ:
        await set_game('dwg-risq');
        break;
      case GameType.TEST_GAME:
        await set_game('dwg-test-game');
        break;
      case GameType.UNSPECIFIED:
      default:
        throw new Error(`Unknown game type: ${new_game.game_base.game_type}`);
    }
    if (this.game !== new_game || this.socket !== socket) {
      return false;
    }
    this.game_name.innerText = capitalize(GameType[new_game.game_base.game_type].toLowerCase().replace('_', ' '));
    if (new_game.game_base.game_started || this.pending_start) {
      this.pending_start = false;
      this.startGame();
    } else {
      this.waiting_room.classList.remove('hide');
      this.game_container.classList.remove('show');
      this.players_waiting_els.clear();
      this.players_waiting.replaceChildren();
      for (const [id, player] of new_game.game_base.players) {
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
    if (
      game_initialized &&
      this.isConnected &&
      this.game === new_game &&
      this.game_el === this.game_container.firstElementChild
    ) {
      this.socketSend(createMessage(`client-${this.clientId()}`, 'game-connected', '', this.game_id.toString()));
      clearInterval(this.ping_interval);
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

  playerDisconnected(player_id: number): void {
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

  playerConnected(player_id: number): void {
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

  startGame(): void {
    if (!this.game) {
      this.pending_start = true;
      return;
    }
    this.pending_start = false;
    this.game.game_base.game_started = true;
    this.waiting_room.classList.add('hide');
    this.game_container.classList.add('show');
  }

  exitGame(): void {
    this.launched = false;
    this.classList.remove('show');
    this.closeChatbox();
    this.socketSend(createMessage(`client-${this.clientId()}`, 'game-exit'));
    this.socket = undefined;
    this.connection_metadata = undefined;
    this.lobby_room = undefined;
    this.game_id = 0;
    this.game = undefined;
    this.pending_start = false;
    this.message_error_count = 0;
    this.running_updates = false;
    this.update_queue = [];
    clearInterval(this.ping_interval);
  }

  pingServer(): void {
    if (!this.socketActive()) {
      this.dispatchEvent(
        new CustomEvent<string>('connection_lost', {
          detail: 'Lost connection in game',
        })
      );
      return;
    }
    if (this.game?.game_base?.game_ended) {
      // nothing to do
    } else if (this.game?.game_base?.game_started) {
      const kind = this.game.game_base.persistant_history ? 'game-resend-last-update' : 'game-lookup-last-update';
      this.socketSend(createMessage(`client-${this.clientId()}`, kind));
    } else {
      this.socketSend(createMessage(`client-${this.clientId()}`, 'game-resend-waiting-room'));
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
