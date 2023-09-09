import {DwgElement} from '../../dwg_element';
import {apiPost} from '../../../scripts/api';
import {LobbyRoom, LobbyRoomFromServer, LobbyUser, serverResponseToRoom} from '../data_models';

import html from './lobby_rooms.html';
import './lobby_rooms.scss';

export class DwgLobbyRooms extends DwgElement {
  rooms = new Map<number, LobbyRoom>();

  constructor() {
    super();
    this.htmlString = html;
  }

  protected override parsedCallback(): void {}

  refresh_rooms_running = false;
  async refreshRooms() {
    if (this.refresh_rooms_running) {
      return;
    }
    this.refresh_rooms_running = true;
    this.innerHTML = ' ... loading';
    this.rooms.clear();
    const response = await apiPost<LobbyRoomFromServer[]>('lobby/rooms/get', '');
    if (response.success) {
      const els: HTMLDivElement[] = [];
      for (const server_room of response.result) {
        const room = serverResponseToRoom(server_room);
        els.push(this.getRoomElement(room));
        this.rooms.set(room.room_id, room);
      }
      this.innerHTML = '';
      for (const el of els) {
        this.appendChild(el);
      }
    } else {
      this.innerHTML = `Error loading rooms: ${response.error_message}`;
    }
    this.refresh_rooms_running = false;
  }

  private getRoomElement(room: LobbyRoom): HTMLDivElement {
    const el = document.createElement('div');
    el.classList.add('lobby-room');
    el.id = `room-${room.room_id}`;
    el.innerText = `${room.host.nickname}'s room`;
    el.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent('join_room', {'detail': room.room_id}))
    });
    return el;
  }

  addRoom(room: LobbyRoom) {
    if (this.rooms.has(room.room_id)) {
      const room_el = this.querySelector<HTMLDivElement>(`#room-${room.room_id}`);
      if (room_el) {
        room_el.replaceWith(this.getRoomElement(room));
      }
    } else {
      this.appendChild(this.getRoomElement(room));
    }
    this.rooms.set(room.room_id, room);
  }

  removeRoom(room_id: number) {
    this.rooms.delete(room_id);
    const room_el = this.querySelector<HTMLDivElement>(`#room-${room_id}`);
    if (room_el) {
      room_el.remove();
    }
  }

  getRoom(room_id: number) {
    return this.rooms.get(room_id);
  }

  clientJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (curr_room) {
      if (curr_room.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.users.delete(joinee.client_id);
      }
    }
    const room = this.getRoom(room_id);
    if (room) {
      room.users.set(joinee.client_id, joinee);
    }
  }

  clientLeavesRoom(room_id: number, client_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.host.client_id === client_id) {
      this.removeRoom(room_id);
    } else {
      room.users.delete(client_id);
    }
  }
}

customElements.define('dwg-lobby-rooms', DwgLobbyRooms);
