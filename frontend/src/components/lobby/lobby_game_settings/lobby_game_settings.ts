import {enumKeys} from '../../../scripts/util';
import {DwgElement} from '../../dwg_element';
import {GameType} from '../data_models';

import html from './lobby_game_settings.html';
import './lobby_game_settings.scss';

export class DwgLobbyGameSettings extends DwgElement {
  game_settings_wrapper: HTMLDivElement;
  game_chooser: HTMLSelectElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('game_settings_wrapper');
    this.configureElement('game_chooser');
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
  }
}

customElements.define('dwg-lobby-game-settings', DwgLobbyGameSettings);
