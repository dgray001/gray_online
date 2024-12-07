import {DwgElement} from '../../dwg_element';
import {ChatMessage, DwgChatbox, SERVER_CHAT_NAME} from '../../chatbox/chatbox';
import {GameSettings, GameType, LobbyRoom, LobbyUser, createMessage} from '../data_models';
import {DwgLobbyGameSettings} from '../lobby_game_settings/lobby_game_settings';
import {capitalize, clickButton, setIntervalX} from '../../../scripts/util';
import {DwgRoomUser} from './room_user/room_user';
import {getReadableGameSettings} from '../lobby_game_settings/game_specific_data';

import html from './lobby_room.html';
import './lobby_room.scss';
import '../lobby_game_settings/lobby_game_settings';
import './room_user/room_user';

enum GameStatusEnum {
  NOT_STARTED = 'Game not started',
  LAUNCHING = 'Game launching ...',
  IN_PROGRESS = 'Game in progress',
  LAUNCH_CANCELED = 'Game launch canceled',
  LAUNCH_FAILED = 'Game launch failed',
  GAME_OVER = 'Game over',
}

export class DwgLobbyRoom extends DwgElement {
  private room_name: HTMLDivElement;
  private chatbox: DwgChatbox;
  private leave_room: HTMLButtonElement;
  private rename_input: HTMLInputElement;
  private rename_room: HTMLButtonElement;
  private cancel_rename: HTMLButtonElement;
  private host_container: HTMLDivElement;
  private players_container: HTMLDivElement;
  private viewers_container: HTMLDivElement;
  private settings_title: HTMLDivElement;
  private num_players_current: HTMLSpanElement;
  private num_players_max: HTMLSpanElement;
  private settings_settings: HTMLDivElement;
  private settings_description: HTMLDivElement;
  private settings_button_container: HTMLDivElement;
  private settings_settings_button: HTMLButtonElement;
  private settings_launch_interval_id: NodeJS.Timer;
  private settings_launch_button: HTMLButtonElement;
  private settings_game_status: HTMLDivElement;
  private lobby_game_settings: DwgLobbyGameSettings;
  private game_button_container: HTMLDivElement;
  private game_resign_button: HTMLButtonElement;
  private game_rejoin_button: HTMLButtonElement;

  private is_host = false;
  private client_id = -1;
  private settings_launching = false;
  private renaming_room = false;
  private room: LobbyRoom;
  private host_el: DwgRoomUser;
  private player_els = new Map<number, DwgRoomUser>();
  private viewer_els = new Map<number,  DwgRoomUser>();

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

  getChatbox(): DwgChatbox {
    return this.chatbox;
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with room');
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      this.dispatchEvent(new CustomEvent('chat_sent', {detail: e.detail}));
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
      this.lobby_game_settings.setSettings(this.room.game_settings, this.room.room_description);
      this.lobby_game_settings.classList.add('show');
    });
    this.lobby_game_settings.addEventListener('saved', () => {
      this.dispatchEvent(new CustomEvent('save_settings', {detail: createMessage(
        `client-${this.room.host.client_id}`,
        'room-settings-update',
        JSON.stringify(this.lobby_game_settings.getSettings()),
        this.room.room_id.toString(),
      )}));
      this.dispatchEvent(new CustomEvent('save_settings', {detail: createMessage(
        `client-${this.room.host.client_id}`,
        'room-update-description',
        this.lobby_game_settings.getDescription(),
        this.room.room_id.toString(),
      )}))
      this.lobby_game_settings.classList.remove('show');
    });
    this.settings_launch_button.addEventListener('click', () => {
      if (this.settings_launching) {
        this.setLaunching(false);
        this.settings_game_status.innerText = GameStatusEnum.LAUNCH_CANCELED;
        if (this.settings_launch_interval_id) {
          clearInterval(this.settings_launch_interval_id);
        }
        this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
          message: 'Game launch canceled',
          sender: SERVER_CHAT_NAME,
        }}));
      } else {
        // TODO: check if launchable
        this.setLaunching(true);
        const countdown = 5;
        this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
          message: `Game is launching in ${countdown} ...`,
          sender: SERVER_CHAT_NAME,
        }}));
        this.settings_launch_interval_id = setIntervalX((counter) => {
          this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
            message: `${countdown - counter} ...`,
            sender: SERVER_CHAT_NAME,
          }}));
        }, 1000, countdown - 1, () => {
          this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
            message: 'Game is launching ...',
            sender: SERVER_CHAT_NAME,
          }}));
          this.dispatchEvent(new Event('launch_game'));
          this.settings_launch_button.disabled = true;
        });
      }
    });
    this.game_resign_button.disabled = true;
    /*
    clickButton(this.game_resign_button, () => {
      this.dispatchEvent(new Event('resign_game'));
    });
    */
    clickButton(this.game_rejoin_button, () => {
      this.dispatchEvent(new Event('rejoin_game'));
    });
    this.room_name.classList.add('show');
    setInterval(() => {
      this.updatePings();
    }, 1500);
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

  refreshRoom(room: LobbyRoom, is_host: boolean) {
    this.setRoom(room, is_host, this.client_id, true);
  }

  setRoom(room: LobbyRoom, is_host: boolean, client_id: number, refresh_room = false) {
    const game_launched = !!room.game_id;
    this.is_host = is_host;
    this.client_id = client_id;
    this.room = room;
    this.room_name.innerText = room.room_name;
    if (!this.lobby_game_settings.classList.contains('show')) {
      this.lobby_game_settings.setSettings(room.game_settings, room.room_description);
      this.updateSettingsDependencies();
    }
    this.num_players_current.innerText = room.players.size.toString();
    if (!refresh_room || game_launched) {
      this.settings_game_status.innerText = game_launched ? GameStatusEnum.IN_PROGRESS : GameStatusEnum.NOT_STARTED;
    }
    this.classList.add('show');
    if (game_launched) {
      this.game_button_container.classList.remove('hide');
    } else {
      this.game_button_container.classList.add('hide');
    }
    this.renaming_room = is_host && refresh_room && this.renaming_room;
    if (is_host) {
      this.rename_room.classList.toggle('show', !this.renaming_room);
      if (game_launched) {
        this.settings_button_container.classList.add('hide');
      } else {
        this.settings_button_container.classList.remove('hide');
      }
    } else {
      this.rename_room.classList.remove('show');
      this.settings_button_container.classList.add('hide');
    }
    this.setLaunching(this.settings_launching && refresh_room);
    this.host_el = this.getUserElement(room.host, is_host, true);
    this.host_container.replaceChildren(this.host_el);
    this.player_els.clear();
    for (const player of room.players.values()) {
      if (player.client_id === room.host.client_id) {
        continue;
      }
      this.player_els.set(player.client_id, this.getUserElement(player, is_host, true));
    }
    this.players_container.replaceChildren(...this.player_els.values());
    this.viewer_els.clear();
    for (const viewer of room.viewers.values()) {
      this.viewer_els.set(viewer.client_id, this.getUserElement(viewer, is_host, false));
    }
    this.viewers_container.replaceChildren(...this.viewer_els.values());
  }

  updateSettings(new_settings: GameSettings) {
    if (!this.room) {
      return;
    }
    this.room.game_settings = new_settings;
    this.updateSettingsDependencies();
  }

  updateSettingsDependencies() {
    this.settings_title.innerText = GameType[this.room.game_settings.game_type ?? -1] ?? '';
    this.num_players_max.innerText = this.room.game_settings.max_players.toString();
    if (!this.room.game_settings.game_specific_settings) {
      this.settings_settings.replaceChildren();
    } else {
      const settings_els: HTMLDivElement[] = []
      const settings = getReadableGameSettings(this.room.game_settings.game_specific_settings, this.room.game_settings.game_type);
      for (const [setting_name, setting] of settings) {
        const setting_el = document.createElement('div');
        setting_el.classList.add('setting');
        setting_el.classList.add('settings-small');
        setting_el.id = `setting-${setting_name}`;
        setting_el.innerText = `${setting_name}: ${setting}`;
        settings_els.push(setting_el);
      }
      this.settings_settings.replaceChildren(...settings_els);
    }
    this.settings_description.innerText = this.room.room_description;
  }

  private getUserElement(user: LobbyUser, is_host: boolean, is_player: boolean): DwgRoomUser {
    const el = document.createElement('dwg-room-user');
    el.classList.add('room-user');
    el.id = `user-${user.client_id}`;
    el.setConfig(user, is_host, is_player, this.client_id === user.client_id, !!this.room.game_id);
    return el;
  }

  clearRoom() {
    this.room = undefined;
    this.lobby_game_settings.clearSettings();
    this.chatbox.clear();
    this.classList.remove('show');
  }

  inRoom(): boolean {
    return !!this.room;
  }

  getRoom(): LobbyRoom {
    return this.room;
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

  updateRoomDescription(new_description: string) {
    if (!!this.room) {
      this.room.room_description = new_description;
      this.settings_description.innerText = new_description;
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
    }
  }

  private updatePings() {
    if (!this.room) {
      return;
    }
    if (!!this.host_el) {
      this.host_el.updatePing(this.room.host.ping);
    }
    for (const [client_id, player_el] of this.player_els.entries()) {
      const player = this.room.players.get(client_id);
      if (!!player) {
        player_el.updatePing(player.ping);
      } else {
        player_el.remove();
      }
    }
    for (const [client_id, viewer_el] of this.viewer_els.entries()) {
      const viewer = this.room.viewers.get(client_id);
      if (!!viewer) {
        viewer_el.updatePing(viewer.ping);
      } else {
        viewer_el.remove();
      }
    }
  }

  joinRoom(joinee: LobbyUser, join_as_player: boolean) {
    if (!this.room) {
      return;
    }
    const user_el = this.querySelector<HTMLDivElement>(`#user-${joinee.client_id}`);
    if (!!user_el) {
      user_el.replaceWith(this.getUserElement(joinee, this.is_host, join_as_player));
    } else if (join_as_player) {
      const player_el = this.getUserElement(joinee, this.is_host, true);
      this.player_els.set(joinee.client_id, player_el);
      this.players_container.appendChild(player_el);
    } else {
      const viewer_el = this.getUserElement(joinee, this.is_host, false);
      this.viewer_els.set(joinee.client_id, viewer_el);
      this.viewers_container.appendChild(viewer_el);
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
    this.setRoom(this.room, is_host, this.client_id);
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
      this.setRoom(this.room, this.is_host, this.client_id);
    }
  }

  viewerToPlayer(client_id: number) {
    if (!this.room) {
      return;
    }
    if (this.room.viewers.has(client_id)) {
      this.room.players.set(client_id, this.room.viewers.get(client_id));
      this.room.viewers.delete(client_id);
      this.setRoom(this.room, this.is_host, this.client_id);
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
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
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
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
      message: 'Game launch failed',
      sender: SERVER_CHAT_NAME,
    }}));
  }

  gameOver(message: string) {
    if (!this.room) {
      return;
    }
    this.room.game_id = undefined;
    this.settings_game_status.innerText = GameStatusEnum.GAME_OVER;
    this.game_button_container.classList.add('hide');
    if (this.is_host) {
      this.settings_launching = false;
      this.settings_launch_button.classList.remove('launching');
      this.settings_launch_button.innerText = 'Launch';
      this.settings_launch_button.disabled = false;
      this.settings_settings_button.disabled = false;
      this.settings_button_container.classList.remove('hide');
    }
    this.dispatchEvent(new CustomEvent<ChatMessage>('chat_sent', {detail: {
      message: `Game over: ${message}`,
      sender: SERVER_CHAT_NAME,
    }}));
  }

  private openRename() {
    this.renaming_room = true;
    this.rename_room.classList.remove('show');
    this.room_name.classList.remove('show');
    this.rename_input.value = this.room.room_name;
    this.rename_input.classList.add('show');
    this.cancel_rename.classList.add('show');
  }

  private cancelRename() {
    this.renaming_room = false;
    this.rename_room.classList.add('show');
    this.room_name.classList.add('show');
    this.rename_input.classList.remove('show');
    this.cancel_rename.classList.remove('show');
  }

  private submitRename() {
    this.dispatchEvent(new CustomEvent('rename_room', {detail: this.rename_input.value}));
    this.cancelRename();
  }
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-lobby-room': DwgLobbyRoom;
  }
}
