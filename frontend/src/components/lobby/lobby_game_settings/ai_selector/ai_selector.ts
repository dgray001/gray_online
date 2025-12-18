import { DwgElement } from '../../../dwg_element';
import { GameType } from '../../data_models';
import type { AiPlayerFiddlesticks } from '../game_specific_data';
import { generateName } from './name_generator';

import html from './ai_selector.html';
import './ai_selector.scss';

/** Possible ai player types */
export type AiPlayerType = AiPlayerFiddlesticks;

/** Data needed to configure an ai selector */
export declare interface AiSelectorData {
  game_type: GameType;
  ai_players: AiPlayerType[]; // type depends on game type
}

export class DwgAiSelector extends DwgElement {
  private player_wrapper!: HTMLDivElement;
  private add_player_button!: HTMLButtonElement;

  private data!: AiSelectorData;
  private els: HTMLElement[] = [];

  constructor() {
    super();
    this.html_string = html;
    this.configureElements('player_wrapper', 'add_player_button');
  }

  protected override parsedCallback(): void {
    if (!this.data) {
      throw new Error('Must set ai selector data before adding to dom');
    }
    this.setPlayers(this.data.ai_players);
    this.add_player_button.addEventListener('click', () => {
      switch (this.data.game_type) {
        case GameType.FIDDLESTICKS:
          const el = document.createElement('div');
          const nickname = generateName();
          el.innerText = nickname;
          this.data.ai_players.push({
            nickname,
          } satisfies AiPlayerFiddlesticks);
          this.player_wrapper.appendChild(this.addPlayer(el));
          break;
        default:
          throw new Error('Unknown game type in ai selector');
      }
    });
  }

  setData(data: AiSelectorData) {
    if (this.fully_parsed) {
      throw new Error('Cannot set data when already added to dom');
    }
    this.data = data;
    this.els = [];
  }

  setPlayers(ai_players: AiPlayerType[]) {
    this.data.ai_players = ai_players;
    this.els = [];
    switch (this.data.game_type) {
      case GameType.FIDDLESTICKS:
        const players: AiPlayerFiddlesticks[] = ai_players;
        const player_els = players.map((p) => {
          const el = document.createElement('div');
          el.innerText = p.nickname;
          return this.addPlayer(el);
        });
        this.player_wrapper.replaceChildren(...player_els);
        break;
      default:
        throw new Error('Unknown game type in ai selector');
    }
  }

  private addPlayer(el: HTMLElement): HTMLDivElement {
    const wrapper = document.createElement('div');
    wrapper.classList.add('player');
    wrapper.classList.add('set-h');
    const remove_button = document.createElement('button');
    remove_button.classList.add('remove-button');
    remove_button.classList.add('set-size');
    const remove_icon = document.createElement('img');
    remove_icon.classList.add('max-size');
    remove_icon.src = '/images/icons/close64.png';
    remove_icon.draggable = false;
    remove_button.appendChild(remove_icon);
    el.id = `player-${this.els.length}`;
    this.els.push(el);
    remove_button.addEventListener('click', () => {
      wrapper.remove();
      const index = parseInt(el.id.replace('player-', ''));
      console.log(this.data.ai_players, index);
      this.data.ai_players.splice(index, 1);
      console.log(this.data.ai_players);
      this.els.splice(index, 1);
      for (const [id, el] of this.els.entries()) {
        if (id < index) {
          continue;
        }
        el.id = `player-${id}`;
      }
      console.log(this.getPlayers().length);
    });
    wrapper.appendChild(el);
    wrapper.appendChild(remove_button);
    return wrapper;
  }

  getPlayers(): AiPlayerType[] {
    switch (this.data.game_type) {
      case GameType.FIDDLESTICKS:
        return this.data.ai_players;
      default:
        throw new Error('Unknown game type in ai selector');
    }
  }
}

customElements.define('dwg-ai-selector', DwgAiSelector);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-ai-selector': DwgAiSelector;
  }
}
