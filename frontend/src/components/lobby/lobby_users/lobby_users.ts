import {DwgElement} from '../../dwg_element';
import {apiGet} from '../../../scripts/api';
import {clickButton} from '../../../scripts/util';
import {LobbyUser, LobbyUserFromServer, serverResponseToUser} from '../data_models';

import html from './lobby_users.html';
import './lobby_users.scss';

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
    setInterval(() => {
      this.updatePings();
    }, 2500);
  }

  refresh_users_running = false;
  async refreshUsers() {
    if (this.refresh_users_running) {
      return;
    }
    this.refresh_users_running = true;
    this.user_container.innerHTML = ' ... loading';
    this.users.clear();
    const response = await apiGet<LobbyUserFromServer[]>('lobby/users/get');
    if (response.success) {
      let html = '';
      for (const server_user of response.result) {
        const user = serverResponseToUser(server_user);
        html += this.addUserString(user);
        this.users.set(user.client_id, user);
      }
      this.user_container.innerHTML = html;
    } else {
      this.user_container.innerHTML = `Error loading users: ${response.error_message}`;
    }
    this.refresh_users_running = false;
  }

  private addUserString(user: LobbyUser): string {
    return `<div class="lobby-user" id="user-${user.client_id}">${this.getUserInnerText(user)}</div>`;
  }

  private getUserInnerText(user: LobbyUser): string {
    return `${user.nickname} (${Math.round(user.ping)})`;
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

  updatePing(client_id: number, ping: number) {
    const user = this.users.get(client_id);
    if (!!user) {
      user.ping = ping;
    }
  }

  private updatePings() {
    for (const client_id of this.users.keys()) {
      const user_el = this.querySelector<HTMLDivElement>(`#user-${client_id}`);
      if (!!user_el) {
        user_el.innerText = this.getUserInnerText(this.users.get(client_id));
      }
    }
  }

  removeUser(client_id: number) {
    this.users.delete(client_id);
    const user_el = this.querySelector<HTMLDivElement>(`#user-${client_id}`);
    if (!!user_el) {
      user_el.classList.add('left');
    }
  }

  getUser(user_id: number): LobbyUser {
    return this.users.get(user_id);
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
