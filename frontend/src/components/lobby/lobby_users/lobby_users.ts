import {DwgElement} from '../../dwg_element';
import {apiGet} from '../../../scripts/api';
import {clickButton, untilTimer} from '../../../scripts/util';
import {LobbyUser, LobbyUserFromServer, serverResponseToUser} from '../data_models';
import {DwgLobbyUser} from './lobby_user/lobby_user';

import html from './lobby_users.html';
import './lobby_users.scss';
import './lobby_user/lobby_user';

interface UserData {
  data: LobbyUser;
  el: DwgLobbyUser;
  refreshed: boolean;
}

export class DwgLobbyUsers extends DwgElement {
  refresh_button: HTMLButtonElement;
  loading_message: HTMLDivElement;
  user_container: HTMLDivElement;

  users = new Map<number, UserData>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('refresh_button');
    this.configureElement('loading_message');
    this.configureElement('user_container');
  }

  protected override parsedCallback(): void {
    clickButton(this.refresh_button, () => {
      this.refreshUsers(true);
    }, {});
    this.refreshUsers();
    setInterval(() => {
      this.updatePings();
    }, 2500);
  }

  private first_load = true;
  async refreshUsers(force_load_message = false) {
    if (this.classList.contains('loading')) {
      return;
    }
    if (this.first_load || force_load_message) {
      this.classList.add('loading');
      this.loading_message.innerHTML = ' ... loading';
      this.first_load = false;
    }
    const response = await apiGet<LobbyUserFromServer[]>('lobby/users/get');
    if (response.success) {
      for (const data of this.users.values()) {
        data.refreshed = false;
      }
      for (const server_user of response.result) {
        const user = serverResponseToUser(server_user);
        this.addUser(user);
      }
      const keys = [...this.users.keys()];
      for (const k of keys) {
        const data = this.users.get(k);
        if (!data) {
          continue;
        }
        if (!data.refreshed) {
          data.el.remove();
          this.users
        }
      }
      this.classList.remove('loading');
    } else {
      this.loading_message.innerHTML = `Error loading users: ${response.error_message}`;
    }
  }

  addUser(user: LobbyUser) {
    if (this.users.has(user.client_id)) {
      this.users.get(user.client_id).data = user;
      this.users.get(user.client_id).refreshed = true;
      this.users.get(user.client_id).el.updateUser(user);
    } else {
      const el = document.createElement('dwg-lobby-user');
      el.classList.add('lobby-user');
      el.updateUser(user);
      this.user_container.appendChild(el);
      this.users.set(user.client_id, {data: user, el, refreshed: true});
    }
  }

  updatePing(client_id: number, ping: number) {
    const user = this.users.get(client_id);
    if (!!user) {
      user.data.ping = ping;
      user.el.updatePing(ping);
    }
  }

  private updatePings() {
    for (const user of this.users.values()) {
      user.el.updatePing(user.data.ping);
    }
  }

  removeUser(client_id: number) {
    this.users.delete(client_id);
    const user_el = this.querySelector<DwgLobbyUser>(`#user-${client_id}`);
    if (!!user_el) {
      user_el.classList.add('left');
    }
  }

  getUser(user_id: number): LobbyUser {
    return this.users.get(user_id).data;
  }

  joinRoom(user_id: number, room_id: number) {
    const user = this.getUser(user_id);
    if (!!user) {
      user.room_id = room_id;
    }
  }

  leaveRoom(user_id: number) {
    const user = this.getUser(user_id);
    if (!!user) {
      user.room_id = undefined;
    }
  }
}

customElements.define('dwg-lobby-users', DwgLobbyUsers);
