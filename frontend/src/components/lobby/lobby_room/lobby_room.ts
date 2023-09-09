import {DwgElement} from '../../dwg_element';
import {DwgChatbox} from '../../chatbox/chatbox';
import {LobbyRoom, LobbyUser} from '../data_models';

import html from './lobby_room.html';
import './lobby_room.scss';

export class DwgLobbyRoom extends DwgElement {
  room_name: HTMLDivElement;
  user_list: HTMLDivElement;
  chatbox: DwgChatbox;
  leave_room: HTMLButtonElement;
  rename_input: HTMLInputElement;
  rename_room: HTMLButtonElement;
  cancel_rename: HTMLButtonElement;

  is_host = false;
  room: LobbyRoom;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
    this.configureElement('user_list');
    this.configureElement('chatbox');
    this.configureElement('leave_room');
    this.configureElement('rename_input');
    this.configureElement('rename_room');
    this.configureElement('cancel_rename');
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
    this.room_name.classList.add('show');
  }

  setRoom(room: LobbyRoom, is_host: boolean) {
    this.is_host = is_host;
    this.room = room;
    this.room_name.innerText = room.room_name;
    this.classList.add('show');
    if (is_host) {
      this.rename_room.classList.add('show');
    }
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
      } else if (this.room.users.has(renamer_id)) {
        this.chatbox.addChat({
          message: `${this.room.users.get(renamer_id).nickname} renamed the room to: ${new_name}`,
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

  hasClient(client_id: number): boolean {
    if (!this.classList.contains('show') || !this.room) {
      return false;
    }
    return this.room.users.has(client_id);
  }

  getClient(client_id: number): LobbyUser {
    if (!this.classList.contains('show') || !this.room) {
      return undefined;
    }
    return this.room.users.get(client_id);
  }

  joinRoom(joinee: LobbyUser) {
    if (!this.room) {
      return;
    }
    this.room.users.set(joinee.client_id, joinee);
  }

  leaveRoom(client_id: number) {
    if (this.room.host.client_id === client_id) {
      this.clearRoom();
    } else {
      this.room.users.delete(client_id);
    }
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
