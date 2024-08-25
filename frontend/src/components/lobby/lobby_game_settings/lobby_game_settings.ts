import {clickButton, DEV, enumKeys} from '../../../scripts/util';
import {DwgElement} from '../../dwg_element';
import {GameSettings, GameType} from '../data_models';

import html from './lobby_game_settings.html';
import './lobby_game_settings.scss';
import './ai_selector/ai_selector';
import {GameSettingsFiddlesticks} from './game_specific_data';
import {AiSelectorData, DwgAiSelector} from './ai_selector/ai_selector';

export class DwgLobbyGameSettings extends DwgElement {
  private game_chooser: HTMLSelectElement;
  private max_players_input: HTMLInputElement;
  private max_viewers_input: HTMLInputElement;
  private game_specific_settings: HTMLDivElement;
  private room_description: HTMLTextAreaElement;
  private save_button: HTMLButtonElement;

  private game_specific_settings_els = new Map<string, HTMLElement>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('game_chooser');
    this.configureElement('max_players_input');
    this.configureElement('max_viewers_input');
    this.configureElement('game_specific_settings');
    this.configureElement('room_description');
    this.configureElement('save_button');
  }

  protected override parsedCallback(): void {
    for (const gameKey of enumKeys(GameType)) {
      if (!GameType[gameKey]) {
        continue;
      }
      if (!DEV && gameKey === GameType[GameType.TEST_GAME].toString()) {
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
        const min_round = this.createNumberElement('min-round', 'Min Round', 1, 17);
        const max_round = this.createNumberElement('max-round', 'Max Round', 1, 17);
        children.push(this.createRowElement([min_round, max_round]));
        const round_points = this.createNumberElement('round-points', 'Round Points', 0, 20, 10);
        const trick_points = this.createNumberElement('trick-points', 'Trick Points', 0, 10, 1);
        children.push(this.createRowElement([round_points, trick_points]));
        children.push(this.createLabelElement('AI Players'));
        const ai_selector = document.createElement('dwg-ai-selector');
        ai_selector.setData({game_type, ai_players: []} satisfies AiSelectorData);
        ai_selector.id = 'ai-players';
        this.game_specific_settings_els.set('ai-players', ai_selector);
        children.push(ai_selector);
        break;
      default:
        break;
    }
    return children;
  }

  private createLabelElement(label: string): HTMLSpanElement {
    const label_el = document.createElement('label');
    label_el.innerText = label;
    return label_el;
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
      el.value = default_value.toString();
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

  setSettings(settings: GameSettings, room_description: string) {
    this.game_chooser.value = settings.game_type.toString();
    this.max_players_input.value = settings.max_players.toString();
    this.max_viewers_input.value = settings.max_viewers.toString();
    try {
      switch(settings.game_type) {
        case GameType.FIDDLESTICKS:
          const specific_settings = settings.game_specific_settings as GameSettingsFiddlesticks;
          this.setNumberSetting('min-round', specific_settings.min_round);
          this.setNumberSetting('max-round', specific_settings.max_round);
          this.setNumberSetting('round-points', specific_settings.round_points);
          this.setNumberSetting('trick-points', specific_settings.trick_points);
          const ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
          ai_players.setPlayers(specific_settings.ai_players);
          break;
        default:
          break;
      }
    } catch(e) {
      console.log(e);
    }
    this.room_description.value = room_description;
  }

  private setNumberSetting(el_id: string, setting?: number) {
    const el = this.game_specific_settings_els.get(el_id) as HTMLInputElement;
    if (!!el) {
      el.value = setting.toString() ?? '0';
    }
  }

  private getNumberSetting(el_id: string): HTMLInputElement {
    return this.game_specific_settings_els.get(el_id) as HTMLInputElement;
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
        const ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
        settings.game_specific_settings = {
          min_round: this.getNumberSetting('min-round')?.valueAsNumber || 0,
          max_round: this.getNumberSetting('max-round')?.valueAsNumber || 0,
          round_points: this.getNumberSetting('round-points')?.valueAsNumber || 0,
          trick_points: this.getNumberSetting('trick-points')?.valueAsNumber || 0,
          ai_players: ai_players.getPlayers(),
        } satisfies GameSettingsFiddlesticks;
        break;
      default:
        break;
    }
    return settings;
  }

  getDescription(): string {
    return this.room_description.value;
  }
}

customElements.define('dwg-lobby-game-settings', DwgLobbyGameSettings);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-lobby-game-settings': DwgLobbyGameSettings;
  }
}
