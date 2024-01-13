import {DwgElement} from '../../../dwg_element';
import { LobbyUser } from '../../data_models';

import html from './lobby_user.html';

import './lobby_user.scss';

export class DwgLobbyUser extends DwgElement {
  private ping_image: HTMLImageElement;
  private ping_text: HTMLDivElement;
  private user_name: HTMLDivElement;
  private buttons: HTMLDivElement;

  private user: LobbyUser;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('ping_image');
    this.configureElement('ping_text');
    this.configureElement('user_name');
    this.configureElement('buttons');
  }

  protected override parsedCallback(): void {
    if (!this.user) {
      console.error('Must set user before attaching to dom');
      return;
    }
    this.setData();
  }

  updateUser(user: LobbyUser) {
    this.user = user;
    if (this.fully_parsed) {
      this.setData();
    }
  }

  private setData() {
    this.user_name.innerText = this.user.nickname;
    this.updatePing(this.user.ping);
  }

  updatePing(ping: number) {
    this.user.ping = ping;
    this.ping_text.innerText = `${Math.round(ping)} ms`;
    if (ping < 100) {
      this.ping_image.src = '/images/icons/ping_good24.png';
    } else if (ping < 300) {
      this.ping_image.src = '/images/icons/ping_ok24.png';
    } else if (ping < 1000) {
      this.ping_image.src = '/images/icons/ping_bad24.png';
    } else {
      this.ping_image.src = '/images/icons/ping_empty24.png';
    }
  }
}

customElements.define('dwg-lobby-user', DwgLobbyUser);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-lobby-user': DwgLobbyUser;
  }
}
