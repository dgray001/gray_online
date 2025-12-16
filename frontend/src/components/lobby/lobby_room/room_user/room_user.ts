import { DwgElement } from '../../../dwg_element';
import type { LobbyUser } from '../../data_models';

import html from './room_user.html';

import './room_user.scss';

export class DwgRoomUser extends DwgElement {
  private ping_image: HTMLImageElement;
  private ping_text: HTMLDivElement;
  private user_name: HTMLDivElement;
  private buttons: HTMLDivElement;

  private user: LobbyUser;
  private is_host = false;
  private is_player = false;
  private is_self = false;
  private room_launched = false;

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
      console.error('Must set config before attaching to dom');
      return;
    }
    this.user_name.innerText = this.user.nickname;
    this.updatePing(this.user.ping);
    if (!this.room_launched) {
      if (this.is_host && !this.is_self) {
        if (this.is_player) {
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'kick', 'kick'));
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'viewer', 'eye'));
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'promote', 'promote'));
        } else {
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'kick', 'kick'));
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'player', 'person'));
        }
      } else if (!this.is_host && this.is_self) {
        if (this.is_player) {
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'viewer', 'eye'));
        } else {
          this.buttons.appendChild(this.getUserButton(this.user.client_id, 'player', 'person'));
        }
      }
    }
  }

  setConfig(user: LobbyUser, is_host: boolean, is_player: boolean, is_self: boolean, room_launched: boolean) {
    this.user = user;
    this.is_host = is_host;
    this.is_player = is_player;
    this.is_self = is_self;
    this.room_launched = room_launched;
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

  private getUserButton(client_id: number, name: string, src: string): HTMLButtonElement {
    const button = document.createElement('button');
    button.id = `${name}-button-${client_id}`;
    button.classList.add(`${name}-button`);
    const icon = document.createElement('img');
    icon.src = `/images/icons/${src}18.png`;
    icon.alt = name;
    icon.draggable = false;
    button.appendChild(icon);
    button.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent(`${name}_player`, { bubbles: true, detail: client_id }));
    });
    return button;
  }
}

customElements.define('dwg-room-user', DwgRoomUser);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-room-user': DwgRoomUser;
  }
}
