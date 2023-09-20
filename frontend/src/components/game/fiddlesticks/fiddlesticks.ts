import {DwgElement} from '../../dwg_element';
import {Game, GameBase, GameComponent, GamePlayer} from '../data_models';
import {StandardCard} from '../util/card_util';

import html from './fiddlesticks.html';
import './fiddlesticks.scss';

/** Data describing a game of fiddlesticks */
export declare interface GameFiddlesticks {
  game_base: GameBase;
  players: FiddlesticksPlayer[];
  round: number;
  max_round: number;
  rounds_increasing: boolean;
  dealer: number;
  turn: number;
  betting: boolean;
}

/** Data describing a fiddlesticks player */
export declare interface FiddlesticksPlayer {
  player: GamePlayer;
  cards: StandardCard[];
  score: number;
  bet: number;
  tricks: number;
}

export class DwgFiddlesticks extends DwgElement implements GameComponent {
  example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgFiddlesticks parsed!');
  }

  initialize(game: Game): void {
    // TODO: implement
  }
}

customElements.define('dwg-fiddlesticks', DwgFiddlesticks);
