import {DwgElement} from '../../dwg_element';
import {LobbyRoom} from '../lobby_room/lobby_room';

import html from './lobby_room_selector.html';
import './lobby_room_selector.scss';

/** Returns html string for lobby room selector */
export function createLobbyRoomSelector(room: LobbyRoom): string {
  return `<dwg-lobby-room-selector room-name="${room.room_name}"></dwg-lobby-room-selector>`;
}

export class DwgLobbyRoomSelector extends DwgElement {
  room_name: HTMLSpanElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
  }

  protected override parsedCallback(): void {
    this.room_name.innerHTML = this.attributes.getNamedItem('room-name')?.value ?? '- unknown -';
  }
}

customElements.define('dwg-lobby-room-selector', DwgLobbyRoomSelector);
