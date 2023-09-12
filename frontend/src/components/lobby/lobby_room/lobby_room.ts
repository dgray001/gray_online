import {DwgElement} from '../../dwg_element';
import {DwgChatbox} from '../../chatbox/chatbox';
import {LobbyRoom, LobbyUser} from '../data_models';
import {DwgLobbyGameSettings} from '../lobby_game_settings/lobby_game_settings';

import html from './lobby_room.html';
import './lobby_room.scss';
import '../lobby_game_settings/lobby_game_settings';

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
  num_players_current: HTMLSpanElement;
  num_players_max: HTMLSpanElement;
  settings_button_container: HTMLDivElement;
  settings_settings_button: HTMLButtonElement;
  settings_launch_button: HTMLButtonElement;
  settings_game_status: HTMLDivElement;
  lobby_game_settings: DwgLobbyGameSettings;

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
    this.configureElement('num_players_current');
    this.configureElement('num_players_max');
    this.configureElement('settings_button_container');
    this.configureElement('settings_settings_button');
    this.configureElement('settings_launch_button');
    this.configureElement('settings_game_status');
    this.configureElement('lobby_game_settings');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with room');
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent) => {
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
      this.lobby_game_settings.classList.add('show');
    });
    this.settings_launch_button.addEventListener('click', () => {
      // TODO: launch game (or cancel if launching)
    });
    this.room_name.classList.add('show');
  }

  setRoom(room: LobbyRoom, is_host: boolean) {
    this.is_host = is_host;
    this.room = room;
    this.room_name.innerText = room.room_name;
    this.num_players_current.innerText = room.players.size.toString();
    this.num_players_max.innerText = room.max_players.toString();
    this.settings_game_status.innerText = 'Game not started';
    this.classList.add('show');
    if (is_host) {
      this.rename_room.classList.add('show');
      this.settings_button_container.classList.remove('hide');
    }
    this.host_container.replaceChildren(this.getUserElement(room.host, false));
    const user_els = [];
    for (const user of room.players.values()) {
      if (user.client_id === room.host.client_id) {
        continue;
      }
      user_els.push(this.getUserElement(user, is_host));
    }
    this.players_container.replaceChildren(...user_els);
  }

  private getUserElement(user: LobbyUser, is_host: boolean): HTMLDivElement {
    const el = document.createElement('div');
    el.classList.add('room-user');
    el.id = `user-${user.client_id}`;
    const el_text = document.createElement('span');
    el_text.innerText = this.getUserElementText(user);
    el.appendChild(el_text);
    if (is_host) {
      const kick_button = document.createElement('button');
      kick_button.id = `kick-button-${user.client_id}`;
      kick_button.classList.add('kick-button');
      const kick_icon = document.createElement('img');
      kick_icon.src = this.kick_img.src;
      kick_icon.alt = 'kick';
      kick_icon.draggable = false;
      kick_button.appendChild(kick_icon);
      el.appendChild(kick_button);
      kick_button.addEventListener('click', () => {
        this.dispatchEvent(new CustomEvent('kick_player', {'detail': user.client_id}));
      });
      const promote_button = document.createElement('button');
      promote_button.id = `promote-button-${user.client_id}`;
      promote_button.classList.add('promote-button');
      const promote_icon = document.createElement('img');
      promote_icon.src = this.promote_img.src;
      promote_icon.alt = 'promote';
      promote_icon.draggable = false;
      promote_button.appendChild(promote_icon);
      el.appendChild(promote_button);
      promote_button.addEventListener('click', () => {
        this.dispatchEvent(new CustomEvent('promote_player', {'detail': user.client_id}));
      });
    }
    return el;
  }

  private getUserElementText(user: LobbyUser): string {
    return `${user.nickname} (${user.ping})`;
  }

  clearRoom() {
    this.room = undefined;
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
      user_el.replaceWith(this.getUserElement(joinee, this.is_host));
    } else if (join_as_player) {
      this.players_container.appendChild(this.getUserElement(joinee, this.is_host));
    } else {
      this.viewers_container.appendChild(this.getUserElement(joinee, this.is_host));
    }
    if (join_as_player) {
      this.room.players.set(joinee.client_id, joinee);
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
      const user = this.room.players.get(client_id);
      this.room.players.delete(client_id);
      const user_el = this.querySelector<HTMLDivElement>(`#user-${client_id}`);
      if (user_el) {
        user_el.remove();
      }
      this.chatbox.addChat({
        message: `${user.nickname} (${user.client_id}) ${left_text} the room`,
        color: 'gray',
      });
    }
  }

  promoteUser(client_id: number, is_host: boolean) {
    if (!this.room) {
      return;
    }
    if (this.room.players.has(client_id)) {
      this.room.host = this.room.players.get(client_id);
    }
    this.setRoom(this.room, is_host);
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
