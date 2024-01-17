import {capitalize} from '../../../scripts/util';
import {DwgDialogBox} from '../../dialog_box/dialog_box';
import {GameType, LobbyRoom} from '../../lobby/data_models';
import {Game} from '../data_models';

import html from './game_info_dialog.html';

import './game_info_dialog.scss';

/** Input data for a game info dialog box */
interface GameInfoData {
  room: LobbyRoom;
  game: Game;
}

export class DwgGameInfoDialog extends DwgDialogBox<GameInfoData> {
  private close_button: HTMLButtonElement;
  private room_title: HTMLDivElement;
  private game_title: HTMLDivElement;
  private num_players_current: HTMLSpanElement;
  private num_players_max: HTMLSpanElement;
  private game_specific_settings: HTMLDivElement;
  private room_description: HTMLDivElement;

  private data: GameInfoData;

  constructor() {
    super();
    this.configureElement('close_button');
    this.configureElement('room_title');
    this.configureElement('game_title');
    this.configureElement('num_players_current');
    this.configureElement('num_players_max');
    this.configureElement('game_specific_settings');
    this.configureElement('room_description');
  }

  override getHTML(): string {
    return html;
  }

  getData(): GameInfoData {
    return this.data;
  }

  setData(data: GameInfoData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
    this.room_title.innerText = this.data.room.room_name;
    this.game_title.innerText = GameType[this.data.room.game_settings.game_type ?? -1] ?? '';
    this.num_players_current.innerText = this.data.game.players.length.toString();
    this.num_players_max.innerText = this.data.room.game_settings.max_players.toString();
    if (!this.data.room.game_settings.game_specific_settings) {
      this.game_specific_settings.replaceChildren();
    } else {
      const settings: HTMLDivElement[] = []
      for (const [setting_name, setting] of Object.entries(this.data.room.game_settings.game_specific_settings)) {
        const setting_el = document.createElement('div');
        setting_el.classList.add('setting');
        setting_el.classList.add('settings-small');
        setting_el.id = `setting-${setting_name}`;
        setting_el.innerText = `${capitalize(setting_name.replace('_', ' '))}: ${setting}`;
        settings.push(setting_el);
      }
      this.game_specific_settings.replaceChildren(...settings);
    }
    this.room_description.innerText = this.data.room.room_description;
  }
}

customElements.define('dwg-game-info-dialog', DwgGameInfoDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-game-info-dialog': DwgGameInfoDialog;
  }
}
