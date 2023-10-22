import {DwgElement} from '../../dwg_element';
import {apiGet} from '../../../scripts/api';
import {createLock} from '../../../scripts/util';
import {GameSettings, LobbyRoom, LobbyRoomFromServer, LobbyUser, serverResponseToRoom} from '../data_models';

import {DwgRoomSelector} from './room_selector/room_selector';
import html from './lobby_rooms.html';

import './lobby_rooms.scss';
import './room_selector/room_selector';

/** Data describing a join room event */
export declare interface JoinRoomData {
  room_id: number;
  join_as_player: boolean;
}

export class DwgLobbyRooms extends DwgElement {
  lock = createLock();
  rooms = new Map<number, LobbyRoom>();

  constructor() {
    super();
    this.htmlString = html;
  }

  protected override parsedCallback(): void {}

  async refreshRooms(client_id: number): Promise<LobbyRoom> {
    let current_room = undefined;
    await this.lock(async () => {
      this.innerHTML = ' ... loading';
      this.rooms.clear();
      const response = await apiGet<LobbyRoomFromServer[]>('lobby/rooms/get');
      if (response.success) {
        const els: DwgRoomSelector[] = [];
        for (const server_room of response.result) {
          const room = serverResponseToRoom(server_room);
          els.push(this.getRoomElement(room));
          this.rooms.set(room.room_id, room);
          if (room.players.has(client_id) || room.viewers.has(client_id)) {
            current_room = room;
          }
        }
        this.replaceChildren(...els);
      } else {
        this.innerHTML = `Error loading rooms: ${response.error_message}`;
      }
    });
    return current_room;
  }

  private getRoomElement(room: LobbyRoom): DwgRoomSelector {
    const el = document.createElement('dwg-room-selector');
    el.classList.add('lobby-room');
    el.id = `room-${room.room_id}`;
    el.setRoom(room);
    el.addEventListener('join_room', (e: CustomEvent<boolean>) => {
      const join_data = {'detail': {room_id: room.room_id, join_as_player: e.detail ?? true}};
      this.dispatchEvent(new CustomEvent<JoinRoomData>('join_room', join_data));
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

  roomUpdated(room: LobbyRoom) {
    const room_el = this.querySelector<DwgRoomSelector>(`#room-${room.room_id}`);
    if (!!room_el) {
      room_el.setRoom(room);
    }
  }

  renameRoom(room_id: number, new_name: string) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.room_name = new_name;
    this.roomUpdated(room);
  }

  promoteUser(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.players.has(user_id)) {
      room.host = room.players.get(user_id);
      this.roomUpdated(room);
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
      this.roomUpdated(room);
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
      this.roomUpdated(room);
    }
  }

  playerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (room_id === joinee.room_id) {
      if (!!curr_room) {
        curr_room.players.set(joinee.client_id, joinee);
        this.roomUpdated(curr_room);
      }
      return;
    }
    if (!!curr_room) {
      if (curr_room.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.players.delete(joinee.client_id);
      }
      this.roomUpdated(curr_room);
    }
    const room = this.getRoom(room_id);
    if (!!room) {
      room.players.set(joinee.client_id, joinee);
      this.roomUpdated(room);
    }
  }

  viewerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (!!curr_room) {
      if (curr_room.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.players.delete(joinee.client_id);
      }
      this.roomUpdated(curr_room);
    }
    const room = this.getRoom(room_id);
    if (!!room) {
      room.viewers.set(joinee.client_id, joinee);
      this.roomUpdated(room);
    }
  }

  clientLeavesRoom(room_id: number, client_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.host.client_id === client_id && !room.game_id) {
      this.removeRoom(room_id);
    } else {
      room.players.delete(client_id);
      room.viewers.delete(client_id);
    }
    this.roomUpdated(room);
  }

  updateRoomSettings(room_id: number, new_settings: GameSettings) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.game_settings = new_settings;
    this.roomUpdated(room);
  }

  launchRoom(room_id: number, game_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.game_id = game_id;
    this.roomUpdated(room);
  }
}

customElements.define('dwg-lobby-rooms', DwgLobbyRooms);
