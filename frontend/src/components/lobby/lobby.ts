import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {LobbyRoom, createLobbyRoom} from '../lobby_room/lobby_room';

import html from './lobby.html';
import './lobby.scss';

interface Lobby {
  rooms: LobbyRoom[];
}

export class DwgLobby extends DwgElement {
  refresh_lobby_button: HTMLButtonElement;
  room_container: HTMLDivElement;
  create_room_button: HTMLButtonElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('refresh_lobby_button');
    this.configureElement('room_container');
    this.configureElement('create_room_button');
  }

  protected override parsedCallback(): void {
    this.refresh_lobby_button.addEventListener('click', () => {
      this.refreshLobby();
    });
    this.refreshLobby();
  }

  private async refreshLobby() {
    this.room_container.innerHTML = ' ... loading';
    const response = await apiPost<LobbyRoom[]>('lobby/rooms/get', '');
    if (response.success) {
      let html = '';
      for (const room of response.result) {
        html += createLobbyRoom(room);
      }
      this.room_container.innerHTML = html;
    } else {
      this.room_container.innerHTML = `Error loading rooms: ${response.error_message}`;
    }
  }
}

customElements.define('dwg-lobby', DwgLobby);
