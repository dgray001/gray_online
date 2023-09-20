import {DwgElement} from '../../dwg_element';
import {GameBase, GameComponent, GamePlayer} from '../data_models';
import {StandardCard, cardToImagePath} from '../util/card_util';
import {DwgFiddlesticksPlayer} from './fiddlesticks_player/fiddlesticks_player';
import {ServerMessage} from '../../lobby/data_models';

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

/** Data describing a deal-round game-update */
declare interface DealRound {
  round: number;
  dealer: number;
  trump: StandardCard;
  cards: StandardCard[];
}

export class DwgFiddlesticks extends DwgElement implements GameComponent {
  round_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  status_container: HTMLSpanElement;
  trump_card_img: HTMLImageElement;
  trick_cards: HTMLDivElement;
  player_container: HTMLDivElement;

  game: GameFiddlesticks;
  player_els: DwgFiddlesticksPlayer[] = [];
  player_id: number = -1;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('trick_number');
    this.configureElement('status_container');
    this.configureElement('trump_card_img');
    this.configureElement('trick_cards');
    this.configureElement('player_container');
  }

  protected override parsedCallback(): void {
    this.status_container.innerText = 'Starting game ...';
  }

  initialize(game: GameFiddlesticks, client_id: number): void {
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-fiddlesticks-player');
      player_el.initialize(player);
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (player.player.client_id === client_id) {
        this.player_id = player_id;
        player_el.setPlayer();
      }
    }
  }

  gameUpdate(update: ServerMessage): void {
    try {
      switch(update.data) {
        case "deal-round":
          const dealRoundData = JSON.parse(update.content) as DealRound;
          this.round_number.innerText = dealRoundData.round.toString();
          this.trump_card_img.src = cardToImagePath(dealRoundData.trump);
          for (const [player_id, player_el] of this.player_els.entries()) {
            if (dealRoundData.dealer === player_id) {
              player_el.setDealer(true);
            } else {
              player_el.setDealer(false);
            }
            if (player_id === this.player_id) {
              player_el.setCards(dealRoundData.cards);
            } else {
              player_el.setHiddenCards(dealRoundData.round);
            }
          }
          break;
        default:
          console.log(`Unknown game update type ${update.data} from ${update.sender}`);
          break;
      }
    } catch(e) {
      console.log(`Error during game update ${update}: ${e}`);
    }
  }
}

customElements.define('dwg-fiddlesticks', DwgFiddlesticks);
