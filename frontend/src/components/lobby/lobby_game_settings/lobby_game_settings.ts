import {clickButton, enumKeys} from '../../../scripts/util';
import {DwgElement} from '../../dwg_element';
import {GameSettings, GameType} from '../data_models';

import html from './lobby_game_settings.html';
import './lobby_game_settings.scss';

export class DwgLobbyGameSettings extends DwgElement {
  game_settings_wrapper: HTMLDivElement;
  game_chooser: HTMLSelectElement;
  max_players_input: HTMLInputElement;
  max_viewers_input: HTMLInputElement;
  save_button: HTMLButtonElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('game_settings_wrapper');
    this.configureElement('game_chooser');
    this.configureElement('max_players_input');
    this.configureElement('max_viewers_input');
    this.configureElement('save_button');
  }

  protected override parsedCallback(): void {
    for (const gameKey of enumKeys(GameType)) {
      if (!GameType[gameKey]) {
        continue;
      }
      const option = document.createElement('option');
      option.value = GameType[gameKey].toString();
      option.innerText = gameKey;
      this.game_chooser.appendChild(option);
    }
    clickButton(this.save_button, () => {
      this.dispatchEvent(new CustomEvent('saved', {'detail': this.getSettings()}));
    });
  }

  setSettings(settings: GameSettings) {
    this.game_chooser.value = settings.game_type ? GameType[settings.game_type] : "0";
    this.max_players_input.valueAsNumber = settings.max_players;
    this.max_viewers_input.valueAsNumber = settings.max_viewers;
  }

  private getSettings(): GameSettings {
    return {
      game_type: parseInt(this.game_chooser.value) ?? 0,
      max_players: parseInt(this.max_players_input.value) ?? 0,
      max_viewers: parseInt(this.max_viewers_input.value) ?? 0,
    };
  }
}

customElements.define('dwg-lobby-game-settings', DwgLobbyGameSettings);
