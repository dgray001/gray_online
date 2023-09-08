import {DwgElement} from '../../dwg_element';
import {apiPost} from '../../../scripts/api';
import {clickButton} from '../../../scripts/util';

import html from './lobby_users.html';
import './lobby_users.scss';

/** Data describing a user in the lobby */
export declare interface LobbyUser {
  client_id: number;
  nickname: string;
}

export class DwgLobbyUsers extends DwgElement {
  refresh_button: HTMLButtonElement;
  user_container: HTMLDivElement;

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
    const response = await apiPost<LobbyUser[]>('lobby/users/get', '');
    if (response.success) {
      let html = '';
      for (const user of response.result) {
        html += this.addUserString(user);
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
    this.user_container.innerHTML += this.addUserString(user);
  }

  removeUser(user_id: number) {
    const user_el = this.querySelector<HTMLDivElement>(`#user-${user_id}`);
    console.log(user_el, user_id);
    if (!user_el) {
      return;
    }
    user_el.classList.add('left');
  }
}

customElements.define('dwg-lobby-users', DwgLobbyUsers);
