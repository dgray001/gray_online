
import {DwgElement} from '../../../dwg_element';
import {GameType} from '../../data_models';
import {AiPlayerFiddlesticks} from '../game_specific_data';

import html from './ai_selector.html';
import './ai_selector.scss';

/** Data needed to configure an ai selector */
export declare interface AiSelectorData {
  game_type: GameType;
  ai_players: any[]; // type depends on game type
}

export class DwgAiSelector extends DwgElement {
  private data: AiSelectorData;

  constructor() {
    super();
    this.htmlString = html;
  }

  protected override parsedCallback(): void {
    if (!this.data) {
      throw new Error('Must set ai selector data before adding to dom');
    }
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
        for (const player of players) {
          // TODO: add player to dom
        }
        break;
      default:
        throw new Error('Unknown game type in ai selector');
    }
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