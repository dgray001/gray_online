import {DwgElement} from '../../dwg_element';
import {apiPost} from '../../../scripts/api';
import {LobbyUser, LobbyUserFromServer, serverResponseToUser} from '../lobby_users/lobby_users';

import html from './lobby_rooms.html';
import './lobby_rooms.scss';

/** Object describing lobby room data */
export declare interface LobbyRoom {
  room_id: number;
  host: LobbyUser;
  users: LobbyUser[];
}

/** Data describing a lobby room returned from the server */
export declare interface LobbyRoomFromServer {
  room_id: string;
  host: LobbyUserFromServer;
  users: LobbyUserFromServer[];
}

/** Convert a server response to a TS lobby room object */
export function serverResponseToRoom(server_response: LobbyRoomFromServer): LobbyRoom {
  return {
    room_id: parseInt(server_response.room_id),
    host: serverResponseToUser(server_response.host),
    users: server_response.users.map(user => serverResponseToUser(user)),
  }
}

export class DwgLobbyRooms extends DwgElement {
  rooms = new Map<number, LobbyRoom>();

  constructor() {
    super();
    this.htmlString = html;
  }

  protected override parsedCallback(): void {}

  async refreshRooms() {
    this.innerHTML = ' ... loading';
    const response = await apiPost<LobbyRoomFromServer[]>('lobby/rooms/get', '');
    if (response.success) {
      let html = '';
      for (const room of response.result) {
        html += this.addRoomString(serverResponseToRoom(room));
      }
      this.innerHTML = html;
    } else {
      this.innerHTML = `Error loading rooms: ${response.error_message}`;
    }
  }

  private addRoomString(room: LobbyRoom): string {
    return `<div class="lobby-room" id="room-${room.room_id}">${room.host.nickname}'s room</div>`;
  }

  addRoom(room: LobbyRoom) {
    if (this.rooms.has(room.room_id)) {
      const room_el = this.querySelector<HTMLDivElement>(`#room-${room.room_id}`);
      if (room_el) {
        room_el.replaceWith(this.addRoomString(room));
      }
    } else {
      this.innerHTML += this.addRoomString(room);
    }
    this.rooms.set(room.room_id, room);
  }

  removeRoom(room_id: number) {
    this.rooms.delete(room_id);
    const room_el = this.querySelector<HTMLDivElement>(`#room-${room_id}`);
    if (!room_el) {
      room_el.remove();
    }
  }

  getRoom(room_id: number) {
    return this.rooms.get(room_id);
  }
}

customElements.define('dwg-lobby-rooms', DwgLobbyRooms);
