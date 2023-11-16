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
import './fiddlesticks/fiddlesticks';
import './euchre/euchre';
import './game_history_dialog/game_history_dialog';
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
  button_game_history: HTMLButtonElement;
  button_room_players: HTMLButtonElement;
  button_exit: HTMLButtonElement;

  launched: boolean;
  socket: WebSocket;
  connection_metadata: ConnectionMetadata;
  game_id = 0;
  game: Game;

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
    this.configureElement('button_game_history');
    this.configureElement('button_room_players');
    this.configureElement('button_exit');
  }

  protected override parsedCallback(): void {
    document.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.toggleChatbox();
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
    this.open_chatbox_button.addEventListener('click', () => {
      this.toggleChatbox();
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
      // TODO: game players dialog box
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

  async launchGame(lobby: LobbyRoom, socket: WebSocket, connection_metadata: ConnectionMetadata, rejoining = false): Promise<boolean> {
    if ((this.launched && !rejoining) || !lobby || !lobby.game_id || !socket || socket.readyState !== WebSocket.OPEN ||
      !connection_metadata || !connection_metadata.client_id || !connection_metadata.room_id || lobby.room_id !== connection_metadata.room_id
    ) {
      console.log('Invalid state to launch game');
      return;
    }
    this.socket = socket;
    this.connection_metadata = connection_metadata;
    this.client_name_string.innerText = this.connection_metadata.nickname;
    this.client_ping.innerText = ` (${this.connection_metadata.ping})`;
    this.room_name.innerText = lobby.room_name;
    this.game_id = lobby.game_id;
    try {
      this.socket.addEventListener('message', (m) => {
        try {
          const message = JSON.parse(m.data) as ServerMessage;
          handleMessage(this, message);
        } catch(e) {
          console.log('Error parsing message: ', m, e)
        }
      });
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
      viewer: "false", // TODO: add viewer support here somehow
    });
    if (!response.success || !response.result) {
      console.log(response.error_message);
      return false;
    }
    this.game = serverResponseToGame(response.result, this.connection_metadata.client_id);
    const setGame = async (component: string) => {
      const game_el = document.createElement(component);
      this.game_container.replaceChildren(game_el);
      await until(() => (game_el as DwgGame).fully_parsed);
      this.game_el = game_el as unknown as GameComponent;
      this.game_el.initialize(this.game, this.connection_metadata.client_id);
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
    this.socket.send(createMessage(
      `client-${this.connection_metadata.client_id}`,
      'game-connected',
      '',
      this.game_id.toString(),
    ));
    this.launched = true;
    return true;
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
