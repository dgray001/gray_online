import {DwgElement} from '../../../dwg_element';
import {GameType, LobbyRoom} from '../../data_models';
import {capitalize} from '../../../../scripts/util';

import html from './room_selector.html';

import './room_selector.scss';

export class DwgRoomSelector extends DwgElement {
  room_name: HTMLDivElement;
  room_curr_players: HTMLSpanElement;
  room_max_players: HTMLSpanElement;
  room_game: HTMLDivElement;
  button_join_player: HTMLButtonElement;
  button_join_viewer: HTMLButtonElement;

  room: LobbyRoom;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('room_name');
    this.configureElement('room_curr_players');
    this.configureElement('room_max_players');
    this.configureElement('room_game');
    this.configureElement('button_join_player');
    this.configureElement('button_join_viewer');
  }

  protected override parsedCallback(): void {
    if (!this.room) {
      console.error('Must set room before attaching to dom');
      return;
    }
    this.setRoomData();
    this.addEventListener('dblclick', () => {
      this.dispatchEvent(new Event('join_room'));
    });
    this.button_join_player.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent<boolean>('join_room', {detail: true}));
    });
    this.button_join_viewer.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent<boolean>('join_room', {detail: false}));
    });
  }

  updateRoom(room: LobbyRoom) {
    this.room = room;
    if (this.fully_parsed) {
      this.setRoomData();
    }
  }

  private setRoomData() {
    if (!this.room) {
      return;
    }
    this.room_curr_players.innerText = this.room.players?.size.toString();
    this.room_max_players.innerText = this.room.game_settings?.max_players?.toString();
    this.room_name.innerText = this.room.room_name;
    this.room_game.innerText = capitalize((GameType[this.room.game_settings.game_type || -1] ?? '').toLowerCase().replace('_', ' '));
  }
}

customElements.define('dwg-room-selector', DwgRoomSelector);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-room-selector': DwgRoomSelector;
  }
}
