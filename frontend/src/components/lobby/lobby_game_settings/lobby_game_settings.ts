import {clickButton, enumKeys} from '../../../scripts/util';
import {DwgElement} from '../../dwg_element';
import {GameSettings, GameType} from '../data_models';

import html from './lobby_game_settings.html';
import './lobby_game_settings.scss';

/** Data describing fiddlesticks-specific game settings */
declare interface GameSettingsFiddlesticks {
  max_round?: number;
  round_points: number;
  trick_points: number;
}

export class DwgLobbyGameSettings extends DwgElement {
  game_settings_wrapper: HTMLDivElement;
  game_chooser: HTMLSelectElement;
  max_players_input: HTMLInputElement;
  max_viewers_input: HTMLInputElement;
  game_specific_settings: HTMLDivElement;
  save_button: HTMLButtonElement;

  game_specific_settings_els = new Map<string, HTMLInputElement>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('game_settings_wrapper');
    this.configureElement('game_chooser');
    this.configureElement('max_players_input');
    this.configureElement('max_viewers_input');
    this.configureElement('game_specific_settings');
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
      this.dispatchEvent(new Event('saved'));
    });
    this.game_chooser.addEventListener('change', () => {
      const value = parseInt(this.game_chooser.value) ?? 0;
      const settings = this.getGameSpecificHTML(value);
      this.game_specific_settings.replaceChildren(...settings);
    });
  }

  getGameSpecificHTML(game_type: GameType): HTMLElement[] {
    const children: HTMLElement[] = [];
    this.game_specific_settings_els.clear();
    switch(game_type) {
      case GameType.FIDDLESTICKS:
        const max_round = this.createNumberElement('max-round', 'Override Max Round', 1, 12);
        const round_points = this.createNumberElement('round-points', 'Round Points', 0, 20, 10);
        const trick_points = this.createNumberElement('trick-points', 'Trick Points', 0, 10, 1);
        children.push(this.createRowElement([max_round, round_points, trick_points]));
        break;
      default:
        break;
    }
    return children;
  }

  private createNumberElement(
    id: string, label: string, min: number, max: number, default_value?: number
  ): HTMLSpanElement {
    const el = document.createElement('input');
    el.id = id;
    el.type = 'number';
    el.placeholder = label;
    el.min = min.toString();
    el.max = max.toString();
    if (default_value !== undefined) {
      el.valueAsNumber = default_value;
    }

    const label_el = document.createElement('label');
    label_el.setAttribute('for', id);
    label_el.innerText = label;
    this.game_specific_settings_els.set(id, el);

    const wrapper_el = document.createElement('span');
    wrapper_el.classList.add('input-wrapper');
    wrapper_el.appendChild(label_el);
    wrapper_el.appendChild(el);
    return wrapper_el;
  }

  private createRowElement(els: HTMLElement[]): HTMLDivElement {
    const el = document.createElement('div');
    el.classList.add('row-wrapper');
    el.replaceChildren(...els);
    return el;
  }

  setSettings(settings: GameSettings) {
    this.game_chooser.value = settings.game_type.toString();
    this.max_players_input.valueAsNumber = settings.max_players;
    this.max_viewers_input.valueAsNumber = settings.max_viewers;
    try {
      switch(settings.game_type) {
        case GameType.FIDDLESTICKS:
          const specific_settings = settings.game_specific_settings as GameSettingsFiddlesticks;
          if (this.game_specific_settings_els.has('max-round')) {
            this.game_specific_settings_els.get('max-round').valueAsNumber = specific_settings.max_round ?? 0;
          }
          if (this.game_specific_settings_els.has('round-points')) {
            this.game_specific_settings_els.get('round-points').valueAsNumber = specific_settings.round_points ?? 0;
          }
          if (this.game_specific_settings_els.has('trick-points')) {
            this.game_specific_settings_els.get('trick-points').valueAsNumber = specific_settings.trick_points ?? 0;
          }
          break;
        default:
          break;
      }
    } catch(e) {
      console.log(e);
    }
  }

  clearSettings() {
    this.game_chooser.value = '';
    this.max_players_input.value = '';
    this.max_viewers_input.value = '';
    this.game_specific_settings.replaceChildren();
    this.game_specific_settings_els.clear();
  }

  getSettings(): GameSettings {
    const settings = {
      game_type: parseInt(this.game_chooser.value) || 0,
      max_players: this.max_players_input.valueAsNumber || 0,
      max_viewers: this.max_viewers_input.valueAsNumber || 0,
    } as GameSettings;
    switch(settings.game_type) {
      case GameType.FIDDLESTICKS:
        settings.game_specific_settings = {
          max_round: this.game_specific_settings_els.get('max-round')?.valueAsNumber || 0,
          round_points: this.game_specific_settings_els.get('round-points')?.valueAsNumber || 0,
          trick_points: this.game_specific_settings_els.get('trick-points')?.valueAsNumber || 0,
        };
        break;
      default:
        break;
    }
    return settings;
  }
}

customElements.define('dwg-lobby-game-settings', DwgLobbyGameSettings);
