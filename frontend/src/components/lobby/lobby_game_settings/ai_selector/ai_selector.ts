
import {DwgElement} from '../../../dwg_element';
import {GameType} from '../../data_models';
import {AiPlayerFiddlesticks} from '../game_specific_data';

import html from './ai_selector.html';
import './ai_selector.scss';
import {generateName} from './name_generator';

/** Data needed to configure an ai selector */
export declare interface AiSelectorData {
  game_type: GameType;
  ai_players: any[]; // type depends on game type
}

export class DwgAiSelector extends DwgElement {
  private player_wrapper: HTMLDivElement;
  private add_player_button: HTMLButtonElement;

  private data: AiSelectorData;
  private els: HTMLElement[] = [];

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('player_wrapper');
    this.configureElement('add_player_button');
  }

  protected override parsedCallback(): void {
    if (!this.data) {
      throw new Error('Must set ai selector data before adding to dom');
    }
    this.add_player_button.addEventListener('click', () => {
      switch(this.data.game_type) {
        case GameType.FIDDLESTICKS:
          const el = document.createElement('div');
          const nickname = generateName();
          el.innerText = nickname;
          this.data.ai_players.push({nickname} satisfies AiPlayerFiddlesticks);
          this.addPlayer(el);
          break;
        default:
          throw new Error('Unknown game type in ai selector');
      }
    });
  }

  setData(data: AiSelectorData) {
    this.data = data;
    this.setPlayers(data.ai_players);
  }

  setPlayers(ai_players: any[]) {
    this.data.ai_players = ai_players;
    switch(this.data.game_type) {
      case GameType.FIDDLESTICKS:
        const players: AiPlayerFiddlesticks[] = ai_players;
        for (const [id, player] of players.entries()) {
          const el = document.createElement('div');
          el.innerText = player.nickname;
          this.addPlayer(el);
        }
        break;
      default:
        throw new Error('Unknown game type in ai selector');
    }
  }

  private addPlayer(el: HTMLElement) {
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
    remove_button.addEventListener('click', () => {
      wrapper.remove();
      const index = parseInt(el.id.replace('player-', ''));
      this.data.ai_players.splice(index, 1);
      this.els.splice(index, 1);
      for (const [id, el] of this.els.entries()) {
        if (id < index) {
          continue;
        }
        el.id = `player-${id}`;
      }
    });
    el.id = `player-${this.els.length}`;
    this.els.push(el);
    wrapper.appendChild(el);
    wrapper.appendChild(remove_button);
    this.player_wrapper.appendChild(wrapper);
  }

  getPlayers(): any[] {
    switch(this.data.game_type) {
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