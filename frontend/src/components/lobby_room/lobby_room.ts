import {DwgElement} from '../dwg_element';

import html from './lobby_room.html';
import './lobby_room.scss';

/** Object describing lobby room data */
export declare interface LobbyRoom {
  room_name: string;
}

/** Returns lobby room html given a lobby room object */
export function createLobbyRoom(data: LobbyRoom): string {
  return `<dwg-lobby-room>${data.room_name}</dwg-lobby-room>`;
}

export class DwgLobbyRoom extends DwgElement {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example', 'example');
  }

  protected override parsedCallback(): void {
    console.log('DwgLobbyRoom parsed!');
  }
}

customElements.define('dwg-lobby-room', DwgLobbyRoom);
