import {DwgElement} from '../../dwg_element';
import {ChatMessage, DwgChatbox, SERVER_CHAT_NAME} from '../../chatbox/chatbox';
import {GameSettings, GameType, LobbyRoom, LobbyUser, createMessage} from '../data_models';
import {DwgLobbyGameSettings} from '../lobby_game_settings/lobby_game_settings';
import {capitalize, clickButton, setIntervalX} from '../../../scripts/util';

import html from './lobby_room.html';
import './lobby_room.scss';
import '../lobby_game_settings/lobby_game_settings';

enum GameStatusEnum {
  NOT_STARTED = 'Game not started',
  LAUNCHING = 'Game launching ...',
  IN_PROGRESS = 'Game in progress',
  LAUNCH_CANCELED = 'Game launch canceled',
  LAUNCH_FAILED = 'Game launch failed',
  GAME_OVER = 'Game over',
}

export class DwgLobbyRoom extends DwgElement {
  room_name: HTMLDivElement;
  chatbox: DwgChatbox;
  leave_room: HTMLButtonElement;
  rename_input: HTMLInputElement;
  rename_room: HTMLButtonElement;
  cancel_rename: HTMLButtonElement;
  host_container: HTMLDivElement;
  players_container: HTMLDivElement;
  viewers_container: HTMLDivElement;
  kick_img: HTMLImageElement;
  promote_img: HTMLImageElement;
  viewer_img: HTMLImageElement;
  player_img: HTMLImageElement;
  settings_title: HTMLDivElement;
  num_players_current: HTMLSpanElement;
  num_players_max: HTMLSpanElement;
  settings_settings: HTMLDivElement;
  settings_description: HTMLDivElement;
  settings_button_container: HTMLDivElement;
  settings_settings_button: HTMLButtonElement;
  settings_launching = false;
  settings_launch_interval_id: NodeJS.Timer;
  settings_launch_button: HTMLButtonElement;
  settings_game_status: HTMLDivElement;
  lobby_game_settings: DwgLobbyGameSettings;
  game_button_container: HTMLDivElement;
  game_resign_button: HTMLButtonElement;
  game_rejoin_button: HTMLButtonElement;

  is_host = false;
  room: LobbyRoom;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
    this.configureElement('chatbox');
    this.configureElement('leave_room');
    this.configureElement('rename_input');
    this.configureElement('rename_room');
    this.configureElement('cancel_rename');
    this.configureElement('host_container');
    this.configureElement('players_container');
    this.configureElement('viewers_container');
    this.configureElement('kick_img');
    this.configureElement('promote_img');
    this.configureElement('viewer_img');
    this.configureElement('player_img');
    this.configureElement('settings_title');
    this.configureElement('num_players_current');
    this.configureElement('num_players_max');
    this.configureElement('settings_settings');
    this.configureElement('settings_description');
    this.configureElement('settings_button_container');
    this.configureElement('settings_settings_button');
    this.configureElement('settings_launch_button');
    this.configureElement('settings_game_status');
    this.configureElement('lobby_game_settings');
    this.configureElement('game_button_container');
    this.configureElement('game_resign_button');
    this.configureElement('game_rejoin_button');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with room');
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      this.dispatchEvent(new CustomEvent('chat_sent', {'detail': e.detail}));
    });
    this.leave_room.addEventListener('click', () => {
      this.dispatchEvent(new Event('leave_room'));
    });
    this.rename_room.addEventListener('click', () => {
      this.openRename();
    });
    this.cancel_rename.addEventListener('click', () => {
      this.cancelRename();
    });
    this.rename_input.addEventListener('keyup', (e) => {
      if (e.key === 'Enter') {
        this.submitRename();
      }
    });
    this.settings_settings_button.addEventListener('click', () => {
      this.lobby_game_settings.setSettings(this.room.game_settings);
      this.lobby_game_settings.classList.add('show');
    });
    this.lobby_game_settings.addEventListener('saved', () => {
      const message = createMessage(
        `client-${this.room.host.client_id}`,
        'room-settings-update',
        JSON.stringify(this.lobby_game_settings.getSettings()),
        this.room.room_id.toString(),
      );
      this.dispatchEvent(new CustomEvent('save_settings', {'detail': message}));
      this.lobby_game_settings.classList.remove('show');
    });
    this.settings_launch_button.addEventListener('click', () => {
      if (this.settings_launching) {
        this.setLaunching(false);
        this.settings_game_status.innerText = GameStatusEnum.LAUNCH_CANCELED;
        if (this.settings_launch_interval_id) {
          clearInterval(this.settings_launch_interval_id);
        }
        this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
          message: 'Game launch canceled',
          sender: SERVER_CHAT_NAME,
        }}));
      } else {
        this.setLaunching(true);
        const countdown = 5;
        this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
          message: `Game is launching in ${countdown} ...`,
          sender: SERVER_CHAT_NAME,
        }}));
        this.settings_launch_interval_id = setIntervalX((counter) => {
          this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
            message: `${countdown - counter} ...`,
            sender: SERVER_CHAT_NAME,
          }}));
        }, 1000, countdown - 1, () => {
          this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
            message: 'Game is launching ...',
            sender: SERVER_CHAT_NAME,
          }}));
          this.dispatchEvent(new Event('launch_game'));
          this.settings_launch_button.disabled = true;
        });
      }
    });
    clickButton(this.game_rejoin_button, () => {
      this.dispatchEvent(new Event('rejoin_game'));
    });
    this.room_name.classList.add('show');
  }

  setLaunching(launching: boolean) {
    this.settings_launching = launching;
    if (launching) {
      this.settings_launch_button.innerText = 'Cancel';
      this.settings_launch_button.classList.add('launching');
      this.settings_game_status.innerText = GameStatusEnum.LAUNCHING;
      this.settings_settings_button.disabled = true;
    } else {
      this.settings_launch_button.innerText = 'Launch';
      this.settings_launch_button.classList.remove('launching');
      this.settings_launch_button.disabled = false;
      this.settings_settings_button.disabled = false;
    }
  }

  setRoom(room: LobbyRoom, is_host: boolean) {
    const game_launched = !!room.game_id;
    this.is_host = is_host;
    this.room = room;
    this.room_name.innerText = room.room_name;
    this.lobby_game_settings.setSettings(room.game_settings);
    this.updateSettingsDependencies();
    this.num_players_current.innerText = room.players.size.toString();
    this.settings_game_status.innerText = game_launched ? GameStatusEnum.IN_PROGRESS : GameStatusEnum.NOT_STARTED;
    this.classList.add('show');
    if (game_launched) {
      this.game_button_container.classList.remove('hide');
    } else {
      this.game_button_container.classList.add('hide');
    }
    if (is_host) {
      this.rename_room.classList.add('show');
      if (game_launched) {
        this.settings_button_container.classList.add('hide');
        this.settings_launch_button.disabled = true;
        this.settings_settings_button.disabled = true;
      } else {
        this.settings_button_container.classList.remove('hide');
        this.settings_launch_button.disabled = false;
        this.settings_settings_button.disabled = false;
      }
    } else {
      this.rename_room.classList.remove('show');
      this.settings_button_container.classList.add('hide');
    }
    this.host_container.replaceChildren(this.getUserElement(room.host, false, true));
    const player_els = [];
    for (const player of room.players.values()) {
      if (player.client_id === room.host.client_id) {
        continue;
      }
      player_els.push(this.getUserElement(player, is_host, true));
    }
    this.players_container.replaceChildren(...player_els);
    const viewer_els = [];
    for (const viewer of room.viewers.values()) {
      viewer_els.push(this.getUserElement(viewer, is_host, false));
    }
    this.viewers_container.replaceChildren(...viewer_els);
  }

  updateSettings(new_settings: GameSettings) {
    if (!this.room) {
      return;
    }
    this.room.game_settings = new_settings;
    this.updateSettingsDependencies();
  }

  updateSettingsDependencies() {
    this.settings_title.innerText = GameType[this.room.game_settings.game_type || -1] ?? '';
    this.num_players_max.innerText = this.room.game_settings.max_players.toString();
    if (!this.room.game_settings.game_specific_settings) {
      this.settings_settings.replaceChildren();
    } else {
      const settings: HTMLDivElement[] = []
      for (const [setting_name, setting] of Object.entries(this.room.game_settings.game_specific_settings)) {
        const setting_el = document.createElement('div');
        setting_el.classList.add('setting');
        setting_el.classList.add('settings-small');
        setting_el.id = `setting-${setting_name}`;
        setting_el.innerText = `${capitalize(setting_name.replace('_', ' '))}: ${setting}`;
        settings.push(setting_el);
      }
      this.settings_settings.replaceChildren(...settings);
    }
  }

  private getUserElement(user: LobbyUser, is_host: boolean, is_player: boolean): HTMLDivElement {
    const el = document.createElement('div');
    el.classList.add('room-user');
    el.id = `user-${user.client_id}`;
    const el_text = document.createElement('span');
    el_text.innerText = this.getUserElementText(user);
    el.appendChild(el_text);
    if (is_host && !this.room.game_id) {
      if (is_player) {
        el.appendChild(this.getUserButton(user.client_id, 'kick', this.kick_img.src));
        el.appendChild(this.getUserButton(user.client_id, 'viewer', this.viewer_img.src));
        el.appendChild(this.getUserButton(user.client_id, 'promote', this.promote_img.src));
      } else {
        el.appendChild(this.getUserButton(user.client_id, 'kick', this.kick_img.src));
        el.appendChild(this.getUserButton(user.client_id, 'player', this.player_img.src));
      }
    }
    return el;
  }

  private getUserButton(client_id: number, name: string, src: string): HTMLButtonElement {
    const button = document.createElement('button');
    button.id = `${name}-button-${client_id}`;
    button.classList.add(`${name}-button`);
    const icon = document.createElement('img');
    icon.src = src;
    icon.alt = name;
    icon.draggable = false;
    button.appendChild(icon);
    button.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent(`${name}_player`, {'detail': client_id}));
    });
    return button;
  }

  private getUserElementText(user: LobbyUser): string {
    return `${user.nickname} (${user.ping})`;
  }

  clearRoom() {
    this.room = undefined;
    this.lobby_game_settings.clearSettings();
    this.chatbox.clear();
    this.classList.remove('show');
  }

  renameRoom(new_name: string, renamer_id: number) {
    if (this.room) {
      this.room.room_name = new_name;
      this.room_name.innerText = new_name;
      if (renamer_id === this.room.host.client_id) {
        this.chatbox.addChat({
          message: `The host has renamed the room to: ${new_name}`,
          color: 'gray',
        });
      } else if (this.room.players.has(renamer_id)) {
        this.chatbox.addChat({
          message: `${this.room.players.get(renamer_id).nickname} renamed the room to: ${new_name}`,
          color: 'gray',
        });
      } else {
        this.chatbox.addChat({
          message: `The room has been renamed to: ${new_name}`,
          color: 'gray',
        });
      }
    }
  }

  hasPlayer(client_id: number): boolean {
    if (!this.classList.contains('show') || !this.room) {
      return false;
    }
    return this.room.players.has(client_id);
  }

  getClient(client_id: number): LobbyUser {
    if (!this.classList.contains('show') || !this.room) {
      return undefined;
    }
    if (this.room.players.has(client_id)) {
      return this.room.players.get(client_id);
    }
    return this.room.viewers.get(client_id);
  }

  getPlayer(client_id: number): LobbyUser {
    if (!this.classList.contains('show') || !this.room) {
      return undefined;
    }
    return this.room.players.get(client_id);
  }

  getHost(): LobbyUser {
    if (!this.classList.contains('show') || !this.room) {
      return undefined;
    }
    return this.room.host;
  }

  updatePing(client_id: number, ping: number) {
    if (!this.room) {
      return;
    }
    if (this.room.players.has(client_id)) {
      this.room.players.get(client_id).ping = ping;
      const user_el = this.querySelector<HTMLDivElement>(`#user-${client_id} span`);
      if (user_el) {
        user_el.innerText = this.getUserElementText(this.room.players.get(client_id));
      }
    }
  }

  joinRoom(joinee: LobbyUser, join_as_player: boolean) {
    if (!this.room) {
      return;
    }
    const user_el = this.querySelector<HTMLDivElement>(`#user-${joinee.client_id}`);
    if (user_el) {
      user_el.replaceWith(this.getUserElement(joinee, this.is_host, true));
    } else if (join_as_player) {
      this.players_container.appendChild(this.getUserElement(joinee, this.is_host, true));
    } else {
      this.viewers_container.appendChild(this.getUserElement(joinee, this.is_host, true));
    }
    if (join_as_player) {
      this.room.players.set(joinee.client_id, joinee);
      this.num_players_current.innerText = this.room.players.size.toString();
    } else {
      this.room.viewers.set(joinee.client_id, joinee);
    }
    this.chatbox.addChat({
      message: `${joinee.nickname} (${joinee.client_id}) joined the room`,
      color: 'gray',
    });
  }

  leaveRoom(client_id: number, left_text: string) {
    if (!this.room) {
      return;
    }
    if (this.room.host.client_id === client_id) {
      this.clearRoom();
    } else {
      const user = this.room.players.get(client_id) ?? this.room.viewers.get(client_id);
      this.room.players.delete(client_id);
      this.room.viewers.delete(client_id);
      this.num_players_current.innerText = this.room.players.size.toString();
      const user_el = this.querySelector<HTMLDivElement>(`#user-${client_id}`);
      if (user_el) {
        user_el.remove();
      }
      if (user) {
        this.chatbox.addChat({
          message: `${user.nickname} (client id ${user.client_id}) ${left_text} the room`,
          color: 'gray',
        });
      }
    }
  }

  promoteUser(client_id: number, is_host: boolean) {
    if (!this.room) {
      return;
    }
    const user = this.room.players.get(client_id);
    if (!user) {
      return;
    }
    this.room.host = user;
    this.setRoom(this.room, is_host);
    this.chatbox.addChat({
      message: `${user.nickname} (client id ${user.client_id}) was promoted as host of room`,
      color: 'gray',
    });
  }

  playerToViewer(client_id: number) {
    if (!this.room) {
      return;
    }
    if (this.room.players.has(client_id)) {
      this.room.viewers.set(client_id, this.room.players.get(client_id));
      this.room.players.delete(client_id);
      this.setRoom(this.room, this.is_host);
    }
  }

  viewerToPlayer(client_id: number) {
    if (!this.room) {
      return;
    }
    if (this.room.viewers.has(client_id)) {
      this.room.players.set(client_id, this.room.viewers.get(client_id));
      this.room.viewers.delete(client_id);
      this.setRoom(this.room, this.is_host);
    }
  }

  launchRoom(game_id: number) {
    if (!this.room) {
      return;
    }
    this.room.game_id = game_id;
    this.settings_game_status.innerText = GameStatusEnum.IN_PROGRESS;
    this.game_button_container.classList.remove('hide');
    this.settings_button_container.classList.add('hide');
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
      message: 'Game launched',
      sender: SERVER_CHAT_NAME,
    }}));
  }

  launchFailed() {
    if (!this.room) {
      return;
    }
    this.room.game_id = undefined;
    this.setLaunching(false);
    this.settings_game_status.innerText = GameStatusEnum.LAUNCH_FAILED;
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
      message: 'Game launch failed',
      sender: SERVER_CHAT_NAME,
    }}));
  }

  gameOver() {
    if (!this.room) {
      return;
    }
    this.room.game_id = undefined;
    this.settings_game_status.innerText = GameStatusEnum.GAME_OVER;
    this.game_button_container.classList.add('hide');
    if (this.is_host) {
      this.settings_button_container.classList.remove('hide');
      this.settings_launch_button.innerText = 'Launch';
      this.settings_launch_button.disabled = false;
      this.settings_settings_button.disabled = false;
    }
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {'detail': {
      message: 'Game over',
      sender: SERVER_CHAT_NAME,
    }}));
  }

  private openRename() {
    this.rename_room.classList.remove('show');
    this.room_name.classList.remove('show');
    this.rename_input.value = this.room.room_name;
    this.rename_input.classList.add('show');
    this.cancel_rename.classList.add('show');
  }

  private cancelRename() {
    this.rename_room.classList.add('show');
    this.room_name.classList.add('show');
    this.rename_input.classList.remove('show');
    this.cancel_rename.classList.remove('show');
  }

  private submitRename() {
    this.dispatchEvent(new CustomEvent('rename_room', {'detail': this.rename_input.value}));
    this.cancelRename();
  }
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);
