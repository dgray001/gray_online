import {DwgElement} from '../../dwg_element';
import {LobbyRoom} from '../lobby_rooms/lobby_rooms';
import {DwgChatbox} from '../../chatbox/chatbox';

import html from './lobby_room.html';
import './lobby_room.scss';

export class DwgLobbyRoom extends DwgElement {
  room_name: HTMLDivElement;
  user_list: HTMLDivElement;
  chatbox: DwgChatbox;

  room: LobbyRoom;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
    this.configureElement('user_list');
    this.configureElement('chatbox');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with room');
  }

  setRoom(room: LobbyRoom) {
    this.room = room;
    this.room_name.innerText = `${room.host.nickname}'s room`;
    this.classList.add('show');
  }

  clearRoom() {
    this.room = undefined;
    this.classList.remove('show');
  }
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);
