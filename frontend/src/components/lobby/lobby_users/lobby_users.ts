import {DwgElement} from '../../dwg_element';
import {apiPost} from '../../../scripts/api';
import {clickButton} from '../../../scripts/util';

import html from './lobby_users.html';
import './lobby_users.scss';

/** Data describing a user in the lobby */
export declare interface LobbyUser {
  client_id: number;
  nickname: string;
  room_id?: number;
}

/** Data describing a user returned from the server */
export declare interface LobbyUserFromServer {
  client_id: string;
  nickname: string;
  room_id?: string;
}

/** Convert a server response to a TS lobby user object */
export function serverResponseToUser(server_response: LobbyUserFromServer): LobbyUser {
  return {
    client_id: parseInt(server_response.client_id),
    nickname: server_response.nickname,
    room_id: parseInt(server_response.room_id) ?? undefined,
  }
}

export class DwgLobbyUsers extends DwgElement {
  refresh_button: HTMLButtonElement;
  user_container: HTMLDivElement;

  users = new Map<number, LobbyUser>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('refresh_button');
    this.configureElement('user_container');
  }

  protected override parsedCallback(): void {
    clickButton(this.refresh_button, () => {
      this.refreshUsers();
    }, {});
    this.refreshUsers();
  }

  add_users_running = false;
  private async refreshUsers() {
    if (this.add_users_running) {
      return;
    }
    this.add_users_running = true;
    this.user_container.innerHTML = ' ... loading';
    const response = await apiPost<LobbyUserFromServer[]>('lobby/users/get', '');
    if (response.success) {
      console.log(response);
      let html = '';
      for (const user of response.result) {
        html += this.addUserString(serverResponseToUser(user));
      }
      this.user_container.innerHTML = html;
    } else {
      this.user_container.innerHTML = `Error loading users: ${response.error_message}`;
    }
    this.add_users_running = false;
  }

  private addUserString(user: LobbyUser): string {
    return `<div class="lobby-user" id="user-${user.client_id}">${user.nickname} (${user.client_id})</div>`;
  }

  addUser(user: LobbyUser) {
    if (this.users.has(user.client_id)) {
      const user_el = this.querySelector<HTMLDivElement>(`#user-${user.client_id}`);
      if (user_el) {
        user_el.replaceWith(this.addUserString(user));
      }
    } else {
      this.user_container.innerHTML += this.addUserString(user);
    }
    this.users.set(user.client_id, user);
  }

  removeUser(user: LobbyUser) {
    this.users.delete(user.client_id);
    const user_el = this.querySelector<HTMLDivElement>(`#user-${user.client_id}`);
    if (!user_el) {
      return;
    }
    user_el.classList.add('left');
  }

  getUser(user_id: number): LobbyUser {
    return this.users.get(user_id);
  }

  leaveRoom(user_id: number) {
    const user = this.getUser(user_id);
    if (user) {
      user.room_id = undefined;
    }
  }
}

customElements.define('dwg-lobby-users', DwgLobbyUsers);
