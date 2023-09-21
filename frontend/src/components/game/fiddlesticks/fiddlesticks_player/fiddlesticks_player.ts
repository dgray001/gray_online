import {DwgElement} from '../../../dwg_element';
import {StandardCard, cardToName} from '../../util/card_util';
import {FiddlesticksPlayer} from '../fiddlesticks';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  name_container: HTMLSpanElement;
  dealer_container: HTMLSpanElement;
  score_container: HTMLSpanElement;
  bet_container: HTMLSpanElement;
  tricks_container: HTMLSpanElement;
  cards_container: HTMLDivElement;

  player: FiddlesticksPlayer;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('dealer_container');
    this.configureElement('score_container');
    this.configureElement('bet_container');
    this.configureElement('tricks_container');
    this.configureElement('cards_container');
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

  setClientPlayer() {
    this.classList.add('client-player');
    // TODO: implement event listeners
  }

  setDealer(dealer: boolean) {
    this.dealer_container.classList.toggle('show', dealer);
  }

  newRound() {
    this.player.bet = -1;
    this.player.tricks = 0;
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
  }

  setHiddenCards(num: number) {
    this.newRound();
    this.player.cards = [];
    // TODO: some kind of animation with number of cards
  }

  setCards(cards: StandardCard[]) {
    this.newRound();
    this.player.cards = cards;
    for (const card of cards) {
      const card_el = document.createElement('div');
      // TODO: add event listener ?
      card_el.innerText = cardToName(card);
      this.cards_container.appendChild(card_el);
    }
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
