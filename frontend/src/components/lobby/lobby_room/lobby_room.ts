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

  room: LobbyRoom;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
    this.configureElement('user_list');
    this.configureElement('chatbox');
    this.configureElement('leave_room');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with room');
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent) => {
      this.dispatchEvent(new CustomEvent('chat_sent', {'detail': e.detail}));
    });
    this.leave_room.addEventListener('click', () => {
      this.dispatchEvent(new Event('leave_room'));
    });
  }

  setRoom(room: LobbyRoom) {
    this.room = room;
    this.room_name.innerText = `${room.host.nickname}'s room`;
    this.classList.add('show');
  }

  clearRoom() {
    this.room = undefined;
    this.chatbox.clear();
    this.classList.remove('show');
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
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);
