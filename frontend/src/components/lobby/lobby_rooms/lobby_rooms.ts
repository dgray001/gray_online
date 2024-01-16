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

interface RoomData {
  data: LobbyRoom;
  el: DwgRoomSelector;
  refreshed: boolean;
}

export class DwgLobbyRooms extends DwgElement {
  private loading_message: HTMLDivElement;
  private room_container: HTMLDivElement;

  lock = createLock();
  refreshing_rooms = false;
  rooms = new Map<number, RoomData>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('loading_message');
    this.configureElement('room_container');
  }

  protected override parsedCallback(): void {}

  private first_load = true;
  async refreshRooms(client_id: number, show_load_message = false): Promise<LobbyRoom> {
    let current_room = undefined;
    await this.lock(async () => {
      if (this.first_load || show_load_message) {
        this.classList.add('loading');
        this.loading_message.innerHTML = ' ... loading';
        this.first_load = false;
      }
      const response = await apiGet<LobbyRoomFromServer[]>('lobby/rooms/get');
      if (response.success) {
        for (const data of this.rooms.values()) {
          data.refreshed = false;
        }
        for (const server_room of response.result) {
          const room = serverResponseToRoom(server_room);
          this.addRoom(room);
          if (room.players.has(client_id) || room.viewers.has(client_id)) {
            current_room = room;
          }
        }
        const keys = [...this.rooms.keys()];
        for (const k of keys) {
          const data = this.rooms.get(k);
          if (!data) {
            continue;
          }
          if (!data.refreshed) {
            data.el.remove();
          }
        }
        this.classList.remove('loading');
      } else {
        this.loading_message.innerHTML = `Error loading rooms: ${response.error_message}`;
      }
    });
    return current_room;
  }

  addRoom(room: LobbyRoom) {
    if (this.rooms.has(room.room_id)) {
      this.rooms.get(room.room_id).data = room;
      this.rooms.get(room.room_id).refreshed = true;
      this.rooms.get(room.room_id).el.updateRoom(room);
    } else {
      const el = document.createElement('dwg-room-selector');
      el.classList.add('lobby-room');
      el.id = `room-selector-${room.room_id}`;
      el.updateRoom(room);
      el.addEventListener('join_room', (e: CustomEvent<boolean>) => {
        const join_data = {'detail': {room_id: room.room_id, join_as_player: e.detail ?? true}};
        this.dispatchEvent(new CustomEvent<JoinRoomData>('join_room', join_data));
      });
      this.room_container.appendChild(el);
      this.rooms.set(room.room_id, {data: room, el, refreshed: true});
    }
  }

  removeRoom(room_id: number) {
    const room_el = this.getRoom(room_id)?.el;
    if (!!room_el) {
      room_el.remove();
    }
    this.rooms.delete(room_id);
  }

  userDisconnected(client_id: number) {
    const removeIds = [];
    for (const room of this.rooms.values()) {
      if (client_id === room.data.host.client_id) {
        removeIds.push(room.data.room_id);
      }
    }
    removeIds.forEach(id => this.removeRoom(id));
  }

  getRoom(room_id: number): RoomData {
    return this.rooms.get(room_id);
  }

  roomUpdated(room: RoomData) {
    room.el.updateRoom(room.data);
  }

  renameRoom(room_id: number, new_name: string) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.data.room_name = new_name;
    this.roomUpdated(room);
  }

  promoteUser(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.data.players.has(user_id)) {
      room.data.host = room.data.players.get(user_id);
      this.roomUpdated(room);
    }
  }

  playerToViewer(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.data.players.has(user_id)) {
      room.data.viewers.set(user_id, room.data.players.get(user_id));
      room.data.players.delete(user_id);
      this.roomUpdated(room);
    }
  }

  viewerToPlayer(room_id: number, user_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.data.viewers.has(user_id)) {
      room.data.players.set(user_id, room.data.viewers.get(user_id));
      room.data.viewers.delete(user_id);
      this.roomUpdated(room);
    }
  }

  playerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (room_id === joinee.room_id) {
      if (!!curr_room) {
        curr_room.data.players.set(joinee.client_id, joinee);
        this.roomUpdated(curr_room);
      }
      return;
    }
    if (!!curr_room) {
      if (curr_room.data.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.data.players.delete(joinee.client_id);
      }
      this.roomUpdated(curr_room);
    }
    const room = this.getRoom(room_id);
    if (!!room) {
      room.data.players.set(joinee.client_id, joinee);
      this.roomUpdated(room);
    }
  }

  viewerJoinsRoom(room_id: number, joinee: LobbyUser) {
    const curr_room = this.getRoom(joinee.room_id);
    if (room_id === joinee.room_id) {
      if (!!curr_room) {
        curr_room.data.viewers.set(joinee.client_id, joinee);
        this.roomUpdated(curr_room);
      }
      return;
    }
    if (!!curr_room) {
      if (curr_room.data.host.client_id === joinee.client_id) {
        this.removeRoom(joinee.room_id);
      } else {
        curr_room.data.players.delete(joinee.client_id);
      }
      this.roomUpdated(curr_room);
    }
    const room = this.getRoom(room_id);
    if (!!room) {
      room.data.viewers.set(joinee.client_id, joinee);
      this.roomUpdated(room);
    }
  }

  clientLeavesRoom(room_id: number, client_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    if (room.data.host.client_id === client_id && !room.data.game_id) {
      this.removeRoom(room_id);
    } else {
      room.data.players.delete(client_id);
      room.data.viewers.delete(client_id);
    }
    this.roomUpdated(room);
  }

  updateRoomSettings(room_id: number, new_settings: GameSettings) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.data.game_settings = new_settings;
    this.roomUpdated(room);
  }

  launchRoom(room_id: number, game_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.data.game_id = game_id;
    this.roomUpdated(room);
  }

  gameOver(room_id: number) {
    const room = this.getRoom(room_id);
    if (!room) {
      return;
    }
    room.data.game_id = undefined;
    this.roomUpdated(room);
  }
}

customElements.define('dwg-lobby-rooms', DwgLobbyRooms);
