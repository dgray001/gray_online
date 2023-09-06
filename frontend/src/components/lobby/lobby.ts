import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';

import html from './lobby.html';
import './lobby.scss';

interface Lobby {
  rooms: LobbyRoom[];
}

interface LobbyRoom {}

export class DwgLobby extends DwgElement {
  room_container: HTMLDivElement;
  create_room_button: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_container');
    this.configureElement('create_room_button');
  }

  protected override parsedCallback(): void {
    this.refreshLobby();
  }

  private async refreshLobby() {
    this.room_container.innerHTML = ' ... loading';
    const response = await apiPost<Lobby>('lobby/rooms/get', '');
  }
}

customElements.define('dwg-lobby', DwgLobby);
