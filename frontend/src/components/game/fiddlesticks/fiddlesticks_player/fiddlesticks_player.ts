import {DwgElement} from '../../../dwg_element';
import {createMessage} from '../../../lobby/data_models';
import {StandardCard, cardToIcon, cardToImagePath} from '../../util/card_util';
import {FiddlesticksPlayer} from '../fiddlesticks';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  name_container: HTMLSpanElement;
  dealer_container: HTMLSpanElement;
  score_container: HTMLSpanElement;
  bet_container: HTMLSpanElement;
  tricks_container: HTMLSpanElement;
  winner_icon: HTMLImageElement;

  player: FiddlesticksPlayer;
  client_player = false;
  card_els: HTMLDivElement[] = [];
  currently_playing = false; // if this player is playing a card

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('dealer_container');
    this.configureElement('score_container');
    this.configureElement('bet_container');
    this.configureElement('tricks_container');
    this.configureElement('winner_icon');
  }

  initialized = false;
  protected override parsedCallback(): void {
    if (!this.initialized) {
      throw new Error('Should initialize fiddlesticks player before attaching to dom');
    }
    this.name_container.innerText = this.player.player.nickname;
    this.score_container.innerText = this.player.score.toString();
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
  }

  initialize(player: FiddlesticksPlayer) {
    this.player = player;
    this.initialized = true;
  }

  gameStarted(betting: boolean, current_turn: boolean) {
    if (betting) {
      this.bet_container.innerText = this.player.bet.toString(); // TODO: shouldn't show if haven't bet yet
      this.tricks_container.innerText = '-';
      if (current_turn) {
        this.betting();
      }
    } else {
      this.bet_container.innerText = this.player.bet.toString();
      this.tricks_container.innerText = this.player.tricks.toString();
      if (current_turn) {
        this.playing();
      }
    }
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
  }

  setDealer(dealer: boolean) {
    this.dealer_container.classList.toggle('show', dealer);
  }

  /** Should be called at end and beginning of rounds */
  newRound() {
    this.player.bet = -1;
    this.player.tricks = 0;
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
  }

  setHiddenCards(_: number) {
    this.newRound();
    this.player.cards = [];
    this.card_els = [];
  }

  setCards(cards: StandardCard[]) {
    // TODO: call newRoud() where this would be called
    this.newRound();
  }

  betting() {
    if (!this.client_player) {
      return;
    }
  }

  setBet(amount: number) {
    this.player.bet = amount;
    this.bet_container.innerText = amount.toString();
  }

  endBetting() {
    this.tricks_container.innerText = '0';
  }

  playing() {
    if (!this.client_player) {
      return;
    }
    this.currently_playing = true;
  }

  playCard(index: number, card: StandardCard) {
    if (this.client_player) {
      this.currently_playing = false;
    }
    if (this.card_els.length) {
      this.card_els[index].remove();
    }
  }

  endTrick(tricks: number) {
    this.player.tricks = tricks;
    this.tricks_container.innerText = tricks.toString();
  }

  setScore(score: number) {
    this.player.score = score;
    this.score_container.innerText = score.toString();
  }

  wonGame() {
    this.winner_icon.classList.add('show');
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
