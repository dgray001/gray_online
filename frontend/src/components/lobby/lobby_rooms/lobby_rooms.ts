import {DwgElement} from '../../dwg_element';
import {apiPost} from '../../../scripts/api';
import {GameSettings, LobbyRoom, LobbyRoomFromServer, LobbyUser, serverResponseToRoom} from '../data_models';

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
    el.innerText = room.room_name;
    el.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent('join_room', {'detail': room.room_id}));
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

  userDisconnected(client_id: number) {
    const removeIds = [];
    for (const room of this.rooms.values()) {
      if (client_id === room.host.client_id) {
        removeIds.push(room.room_id);
      }
    }
    removeIds.forEach(id => this.removeRoom(id));
  }

  getRoom(room_id: number) {
    return this.rooms.get(room_id);
  }

  renameRoom(room_id: number, new_name: string) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.room_name = new_name;
    const room_el = this.querySelector<HTMLDivElement>(`#room-${room.room_id}`);
    if (room_el) {
      room_el.replaceWith(this.getRoomElement(room));
    }
  }

  promoteUser(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.players.has(user_id)) {
      room.host = room.players.get(user_id);
    }
  }

  playerToViewer(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.players.has(user_id)) {
      room.viewers.set(user_id, room.players.get(user_id));
      room.players.delete(user_id);
    }
  }

  viewerToPlayer(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.viewers.has(user_id)) {
      room.players.set(user_id, room.viewers.get(user_id));
      room.viewers.delete(user_id);
    }
  }

  playerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (curr_room) {
      if (curr_room.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.players.delete(joinee.client_id);
      }
    }
    const room = this.getRoom(room_id);
    if (room) {
      room.players.set(joinee.client_id, joinee);
    }
  }

  viewerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (curr_room) {
      if (curr_room.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.players.delete(joinee.client_id);
      }
    }
    const room = this.getRoom(room_id);
    if (room) {
      room.viewers.set(joinee.client_id, joinee);
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
      room.players.delete(client_id);
      room.viewers.delete(client_id);
    }
  }

  updateRoomSettings(room_id: number, new_settings: GameSettings) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.game_settings = new_settings;
  }
}

customElements.define('dwg-lobby-rooms', DwgLobbyRooms);
