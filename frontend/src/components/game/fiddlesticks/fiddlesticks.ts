import {DwgElement} from '../../dwg_element';
import {GameBase, GameComponent, GamePlayer} from '../data_models';
import {StandardCard} from '../util/card_util';
import {DwgFiddlesticksPlayer} from './fiddlesticks_player/fiddlesticks_player';

import html from './fiddlesticks.html';

import './fiddlesticks.scss';
import './fiddlesticks_player/fiddlesticks_player';

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
  round_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  trump_suit: HTMLSpanElement;
  trump_card_img: HTMLImageElement;
  trick_cards: HTMLDivElement;
  player_container: HTMLDivElement;

  game: GameFiddlesticks;
  player_els = new Map<number, DwgFiddlesticksPlayer>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('trick_number');
    this.configureElement('trump_suit');
    this.configureElement('trump_card_img');
    this.configureElement('trick_cards');
    this.configureElement('player_container');
  }

  protected override parsedCallback(): void {
  }

  initialize(game: GameFiddlesticks): void {
    this.game = game;
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-fiddlesticks-player');
      player_el.initialize(player);
      this.player_container.appendChild(player_el);
      this.player_els.set(player_id, player_el);
    }
  }
}

customElements.define('dwg-fiddlesticks', DwgFiddlesticks);
