import { DwgElement } from '../../../dwg_element';
import type { LobbyRoom } from '../../data_models';
import { GameType } from '../../data_models';
import { capitalize } from '../../../../scripts/util';

import html from './room_selector.html';

import './room_selector.scss';

export class DwgRoomSelector extends DwgElement {
  private room_name!: HTMLDivElement;
  private room_curr_players!: HTMLSpanElement;
  private room_max_players!: HTMLSpanElement;
  private room_game!: HTMLDivElement;
  private button_join_player!: HTMLButtonElement;
  private button_join_viewer!: HTMLButtonElement;

  private room!: LobbyRoom;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'room_name',
      'room_curr_players',
      'room_max_players',
      'room_game',
      'button_join_player',
      'button_join_viewer'
    );
  }

  protected override parsedCallback(): void {
    if (!this.room) {
      console.error('Must set room before attaching to dom');
      return;
    }
    this.setRoomData();
    this.addEventListener('dblclick', () => {
      this.dispatchEvent(new Event('join_lobby_room'));
    });
    this.button_join_player.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent<boolean>('join_lobby_room', { detail: true }));
    });
    this.button_join_viewer.addEventListener('click', () => {
      this.dispatchEvent(new CustomEvent<boolean>('join_lobby_room', { detail: false }));
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
    this.room_game.innerText = capitalize(
      (GameType[this.room.game_settings.game_type || -1] ?? '').toLowerCase().replace('_', ' ')
    );
  }
}

customElements.define('dwg-room-selector', DwgRoomSelector);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-room-selector': DwgRoomSelector;
  }
}
