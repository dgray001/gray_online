import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {LobbyRoom} from '../lobby_room/lobby_room';
import {createLobbyRoomSelector} from '../lobby_room_selector/lobby_room_selector';

import html from './lobby.html';
import './lobby.scss';
import { clickButton, untilTimer } from '../../scripts/util';

interface Lobby {
  rooms: LobbyRoom[];
}

export class DwgLobby extends DwgElement {
  refresh_lobby_button: HTMLButtonElement;
  room_container: HTMLDivElement;
  create_room_button: HTMLButtonElement;

  socket: WebSocket;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('refresh_lobby_button');
    this.configureElement('room_container');
    this.configureElement('create_room_button');
  }

  protected override parsedCallback(): void {
    clickButton(this.refresh_lobby_button, async () => {
      await this.refreshLobby();
    });
    clickButton(this.create_room_button, async () => {
      if (!!this.socket) {
        this.socket.close(1000, "opening new connection");
      }
      this.socket = new WebSocket('ws://127.0.0.1:6807/api/lobby/rooms/create');
      this.socket.addEventListener('error', (e) => {
        console.log(e);
      });
      this.socket.addEventListener('message', (e) => {
        console.log(e);
      });
      this.socket.addEventListener('open', (e) => {
        console.log('socket opened', e);
      });
      return 'Room Created';
    }, {loading_text: 'Creating Room ...', re_enable_button: false});
    this.refreshLobby();
  }

  private async refreshLobby() {
    this.room_container.innerHTML = ' ... loading';
    await untilTimer(500);
    const response = await apiPost<LobbyRoom[]>('lobby/rooms/get', '');
    if (response.success) {
      let html = '';
      for (const room of response.result) {
        html += createLobbyRoomSelector(room);
      }
      this.room_container.innerHTML = html;
    } else {
      this.room_container.innerHTML = `Error loading rooms: ${response.error_message}`;
    }
  }
}

customElements.define('dwg-lobby', DwgLobby);
