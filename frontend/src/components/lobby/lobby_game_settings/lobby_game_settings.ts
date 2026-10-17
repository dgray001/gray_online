import { clickButton, DEV, enumKeys } from '../../../scripts/util';
import { apiGet } from '../../../scripts/api';
import { DwgElement } from '../../dwg_element';
import type { GameSettings } from '../data_models';
import { GameType } from '../data_models';
import type { GameSettingsFiddlesticks, GameSettingsRisq } from './game_specific_data';
import type { AiSelectorData, DwgAiSelector } from './ai_selector/ai_selector';
import {
  StartingStockpile,
  MapResourceLevel,
  STARTING_STOCKPILE_LABELS,
  MAP_RESOURCE_LABELS,
} from './resource_settings';
import {
  MapType,
  StartingDevelopment,
  StartBalance,
  STARTING_DEVELOPMENT_LABELS,
  START_BALANCE_LABELS,
} from './setup_settings';

import html from './lobby_game_settings.html';
import './lobby_game_settings.scss';
import './ai_selector/ai_selector';

export class DwgLobbyGameSettings extends DwgElement {
  private game_chooser!: HTMLSelectElement;
  private max_players_input!: HTMLInputElement;
  private max_viewers_input!: HTMLInputElement;
  private game_specific_settings!: HTMLDivElement;
  private room_description!: HTMLTextAreaElement;
  private save_button!: HTMLButtonElement;

  private game_specific_settings_els = new Map<string, HTMLElement>();
  private risq_maps: string[] = [];
  private risq_map = 'script:hexagon';

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'game_chooser',
      'max_players_input',
      'max_viewers_input',
      'game_specific_settings',
      'room_description',
      'save_button'
    );
  }

  protected override parsedCallback(): void {
    for (const game_key of enumKeys(GameType)) {
      if (!GameType[game_key]) {
        continue;
      }
      if (!DEV && game_key === GameType[GameType.TEST_GAME].toString()) {
        continue;
      }
      const option = document.createElement('option');
      option.value = GameType[game_key].toString();
      option.innerText = game_key;
      this.game_chooser.appendChild(option);
    }
    clickButton(this.save_button, () => {
      if (
        parseInt(this.game_chooser.value) === GameType.RISQ &&
        !this.getNumberSetting('population-limit')!.reportValidity()
      ) {
        return;
      }
      this.dispatchEvent(new Event('saved'));
    });
    this.game_chooser.addEventListener('change', () => {
      this.setGameChooserValue(this.game_chooser.value);
    });
  }

  private setGameChooserValue(value: string): void {
    this.game_chooser.value = value;
    const game_type = parseInt(value) || 0;
    const is_euchre = game_type === GameType.EUCHRE;
    const max_players = game_type === GameType.RISQ ? 12 : is_euchre ? 4 : 8;
    this.max_players_input.min = is_euchre ? '4' : '1';
    this.max_players_input.max = max_players.toString();
    this.max_players_input.disabled = is_euchre;
    if (is_euchre || this.max_players_input.valueAsNumber > max_players) {
      this.max_players_input.value = max_players.toString();
    }
    const settings = this.getGameSpecificHTML(game_type);
    this.game_specific_settings.replaceChildren(...settings);
  }

  getGameSpecificHTML(game_type: GameType): HTMLElement[] {
    const children: HTMLElement[] = [];
    this.game_specific_settings_els.clear();
    switch (game_type) {
      case GameType.FIDDLESTICKS:
        const min_round = this.createNumberElement('min-round', 'Min Round', 1, 17);
        const max_round = this.createNumberElement('max-round', 'Max Round', 1, 17);
        children.push(this.createRowElement([min_round, max_round]));
        const round_points = this.createNumberElement('round-points', 'Round Points', 0, 20, 10);
        const trick_points = this.createNumberElement('trick-points', 'Trick Points', 0, 10, 1);
        children.push(this.createRowElement([round_points, trick_points]));
        children.push(this.createLabelElement('AI Players'));
        const ai_selector = document.createElement('dwg-ai-selector');
        ai_selector.setData({
          game_type,
          ai_players: [],
        } satisfies AiSelectorData);
        ai_selector.id = 'ai-players';
        this.game_specific_settings_els.set('ai-players', ai_selector);
        children.push(ai_selector);
        break;
      case GameType.RISQ:
        this.risq_map = 'script:hexagon';
        const map_type = this.createSelectElement(
          'map-type',
          'Map Type',
          [
            { value: MapType.SCRIPT, text: 'Script' },
            { value: MapType.CUSTOM, text: 'Custom' },
          ],
          MapType.SCRIPT
        );
        const map_size = this.createSelectElement('map-size', 'Map Size', [
          { value: '', text: 'Recommended' },
          { value: '1', text: 'Minuscule' },
          { value: '2', text: 'Tiny' },
          { value: '3', text: 'Smaller' },
          { value: '4', text: 'Small' },
          { value: '5', text: 'Medium' },
          { value: '6', text: 'Large' },
          { value: '7', text: 'Larger' },
          { value: '8', text: 'Huge' },
          { value: '9', text: 'Gigantic' },
        ]);
        const map_select = this.createSelectElement('map', 'Map', []);
        const visibility = this.createSelectElement(
          'visibility',
          'Visibility',
          [
            { value: '1', text: 'Default' },
            { value: '2', text: 'Explored' },
            { value: '3', text: 'All Visible' },
          ],
          '1'
        );

        children.push(this.createRowElement([map_type, map_select, visibility]));
        const starting_stockpile = this.createSelectElement(
          'starting-stockpile',
          'Starting Stockpile',
          Object.entries(STARTING_STOCKPILE_LABELS)
            .filter(([value]): boolean => Number(value) !== StartingStockpile.NONE)
            .map(([value, text]): { value: string; text: string } => ({ value, text })),
          StartingStockpile.STANDARD.toString()
        );
        const map_resources = this.createSelectElement(
          'map-resources',
          'Map Resources',
          Object.entries(MAP_RESOURCE_LABELS)
            .filter(([value]): boolean => Number(value) !== MapResourceLevel.NONE)
            .map(([value, text]): { value: string; text: string } => ({ value, text })),
          MapResourceLevel.STANDARD.toString()
        );
        const starting_development = this.createSelectElement(
          'starting-development',
          'Starting Development',
          Object.entries(STARTING_DEVELOPMENT_LABELS)
            .filter(([value]): boolean => Number(value) !== StartingDevelopment.NONE)
            .map(([value, text]): { value: string; text: string } => ({ value, text })),
          StartingDevelopment.STANDARD.toString()
        );
        children.push(this.createRowElement([starting_stockpile, map_resources, starting_development]));
        children.push(
          this.createRowElement([this.createNumberElement('population-limit', 'Population Limit', 25, 200, 100)])
        );
        this.getNumberSetting('population-limit')!.required = true;
        const start_balance = this.createSelectElement(
          'start-balance',
          'Start Balance',
          Object.entries(START_BALANCE_LABELS)
            .filter(([value]): boolean => Number(value) !== StartBalance.NONE)
            .map(([value, text]): { value: string; text: string } => ({ value, text })),
          StartBalance.MIRRORED.toString()
        );
        const script_settings = this.createRowElement([map_size, start_balance]);
        this.game_specific_settings_els.set('script-settings', script_settings);
        children.push(script_settings);

        const map_element = this.game_specific_settings_els.get('map') as HTMLSelectElement;
        this.game_specific_settings_els.get('map-type')!.addEventListener('change', (): void => {
          this.risq_map = '';
          this.updateRisqMapOptions();
        });
        map_element.addEventListener('change', (): void => {
          this.risq_map = map_element.value;
        });
        this.updateRisqMapOptions();
        apiGet<string[]>('risq/maps').then((res) => {
          if (res.success && this.game_specific_settings_els.get('map') === map_element) {
            this.risq_maps = res.result;
            this.updateRisqMapOptions();
          }
        });

        children.push(this.createLabelElement('AI Players'));
        const risq_ai_selector = document.createElement('dwg-ai-selector');
        risq_ai_selector.setData({
          game_type,
          ai_players: [],
        } satisfies AiSelectorData);
        risq_ai_selector.id = 'ai-players';
        this.game_specific_settings_els.set('ai-players', risq_ai_selector);
        children.push(risq_ai_selector);
        break;
      default:
        break;
    }
    return children;
  }

  private updateRisqMapOptions(): void {
    const type = this.getSelectSetting('map-type');
    const select = this.game_specific_settings_els.get('map') as HTMLSelectElement;
    this.game_specific_settings_els.get('script-settings')!.hidden = type !== MapType.SCRIPT;
    const names = this.risq_maps.filter((name: string): boolean => name.startsWith(`${type}:`));
    select.replaceChildren();
    for (const name of names) {
      const option = document.createElement('option');
      option.value = name;
      option.innerText = name.slice(name.indexOf(':') + 1);
      select.appendChild(option);
    }
    select.value = names.includes(this.risq_map) ? this.risq_map : (names[0] ?? '');
    if (select.value) {
      this.risq_map = select.value;
    }
  }

  private createLabelElement(label: string): HTMLSpanElement {
    const label_el = document.createElement('label');
    label_el.innerText = label;
    return label_el;
  }

  private createNumberElement(
    id: string,
    label: string,
    min: number,
    max: number,
    default_value?: number
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

  private createSelectElement(
    id: string,
    label: string,
    options: { value: string; text: string }[],
    default_value?: string
  ): HTMLSpanElement {
    const el = document.createElement('select');
    el.id = id;
    for (const opt of options) {
      const option = document.createElement('option');
      option.value = opt.value;
      option.innerText = opt.text;
      el.appendChild(option);
    }
    if (default_value !== undefined) {
      el.value = default_value;
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

  setSettings(settings: GameSettings, room_description: string): void {
    this.setGameChooserValue(settings.game_type.toString());
    this.max_players_input.value = settings.max_players.toString();
    this.max_viewers_input.value = settings.max_viewers.toString();
    try {
      switch (settings.game_type) {
        case GameType.FIDDLESTICKS:
          const specific_settings = settings.game_specific_settings as GameSettingsFiddlesticks;
          this.setNumberSetting('min-round', specific_settings.min_round);
          this.setNumberSetting('max-round', specific_settings.max_round);
          this.setNumberSetting('round-points', specific_settings.round_points);
          this.setNumberSetting('trick-points', specific_settings.trick_points);
          const ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
          if (!!ai_players) {
            ai_players.setPlayers(specific_settings.ai_players);
          }
          break;
        case GameType.RISQ:
          const risq_settings = settings.game_specific_settings as GameSettingsRisq;
          for (const [id, value] of [
            ['starting-stockpile', risq_settings.starting_stockpile || StartingStockpile.STANDARD],
            ['map-resources', risq_settings.map_resources || MapResourceLevel.STANDARD],
            ['starting-development', risq_settings.starting_development || StartingDevelopment.STANDARD],
            ['start-balance', risq_settings.start_balance || StartBalance.MIRRORED],
          ] as const) {
            const el = this.game_specific_settings_els.get(id) as HTMLSelectElement;
            if (el) {
              el.value = value.toString();
            }
          }
          const risq_ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
          this.setNumberSetting('population-limit', risq_settings.population_limit ?? 100);
          if (!!risq_ai_players) {
            risq_ai_players.setPlayers(risq_settings.ai_players);
          }
          if (risq_settings.map_size !== undefined) {
            const el = this.game_specific_settings_els.get('map-size') as HTMLSelectElement;
            if (el) {
              el.value = risq_settings.map_size.toString();
            }
          }
          if (risq_settings.map !== undefined) {
            this.risq_map = risq_settings.map;
            const el = this.game_specific_settings_els.get('map-type') as HTMLSelectElement;
            el.value = risq_settings.map.startsWith(`${MapType.CUSTOM}:`) ? MapType.CUSTOM : MapType.SCRIPT;
            this.updateRisqMapOptions();
          }
          if (risq_settings.visibility !== undefined) {
            const el = this.game_specific_settings_els.get('visibility') as HTMLSelectElement;
            if (el) {
              el.value = risq_settings.visibility.toString();
            }
          }
          break;
        default:
          break;
      }
    } catch (e) {
      console.error(e);
    }
    this.room_description.value = room_description;
  }

  private setNumberSetting(el_id: string, setting?: number): void {
    const el = this.game_specific_settings_els.get(el_id) as HTMLInputElement;
    if (!!el) {
      el.value = setting?.toString() ?? '0';
    }
  }

  private getNumberSetting(el_id: string): HTMLInputElement {
    return this.game_specific_settings_els.get(el_id) as HTMLInputElement;
  }

  private getSelectSetting(el_id: string): string | undefined {
    const el = this.game_specific_settings_els.get(el_id) as HTMLSelectElement;
    return el?.value;
  }

  clearSettings(): void {
    this.game_chooser.value = '';
    this.max_players_input.value = '';
    this.max_players_input.min = '1';
    this.max_players_input.max = '8';
    this.max_players_input.disabled = false;
    this.max_viewers_input.value = '';
    this.game_specific_settings.replaceChildren();
    this.game_specific_settings_els.clear();
  }

  getSettings(): GameSettings {
    const game_type = parseInt(this.game_chooser.value) || 0;
    const settings = {
      game_type,
      max_players: game_type === GameType.EUCHRE ? 4 : this.max_players_input.valueAsNumber || 0,
      max_viewers: this.max_viewers_input.valueAsNumber || 0,
    } as GameSettings;
    switch (settings.game_type) {
      case GameType.FIDDLESTICKS:
        const fiddlesticks_ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
        settings.game_specific_settings = {
          min_round: this.getNumberSetting('min-round')?.valueAsNumber || 0,
          max_round: this.getNumberSetting('max-round')?.valueAsNumber || 0,
          round_points: this.getNumberSetting('round-points')?.valueAsNumber || 0,
          trick_points: this.getNumberSetting('trick-points')?.valueAsNumber || 0,
          ai_players: fiddlesticks_ai_players.getPlayers(),
        } satisfies GameSettingsFiddlesticks;
        break;
      case GameType.RISQ:
        const risq_ai_players = this.game_specific_settings_els.get('ai-players') as DwgAiSelector;
        const map_size_val = this.getSelectSetting('map-size');
        const scripted = this.getSelectSetting('map-type') === MapType.SCRIPT;
        settings.game_specific_settings = {
          ai_players: risq_ai_players.getPlayers(),
          map_size: scripted && map_size_val ? parseInt(map_size_val) : undefined,
          map: this.getSelectSetting('map') || this.risq_map || undefined,
          visibility: parseInt(this.getSelectSetting('visibility') || '1'),
          starting_stockpile: parseInt(
            this.getSelectSetting('starting-stockpile') || StartingStockpile.STANDARD.toString()
          ),
          map_resources: parseInt(this.getSelectSetting('map-resources') || MapResourceLevel.STANDARD.toString()),
          starting_development: parseInt(
            this.getSelectSetting('starting-development') || StartingDevelopment.STANDARD.toString()
          ),
          population_limit: this.getNumberSetting('population-limit')?.valueAsNumber || 100,
          start_balance: scripted
            ? parseInt(this.getSelectSetting('start-balance') || StartBalance.MIRRORED.toString())
            : undefined,
        } satisfies GameSettingsRisq;
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
