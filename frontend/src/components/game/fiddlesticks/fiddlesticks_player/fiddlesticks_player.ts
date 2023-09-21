import {DwgElement} from '../../../dwg_element';
import {createMessage} from '../../../lobby/data_models';
import {StandardCard, cardToImagePath, cardToName} from '../../util/card_util';
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
  betting_container: HTMLDivElement;
  bet_input: HTMLInputElement;
  bet_button: HTMLButtonElement;

  player: FiddlesticksPlayer;
  client_player = false;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('dealer_container');
    this.configureElement('score_container');
    this.configureElement('bet_container');
    this.configureElement('tricks_container');
    this.configureElement('cards_container');
    this.configureElement('betting_container');
    this.configureElement('bet_input');
    this.configureElement('bet_button');
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
    this.client_player = true;
    this.bet_button.addEventListener('click', () => {
      this.bet_button.disabled = true;
      const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', `{"amount":${this.bet_input.value}}`, 'bet');
      this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, bubbles: true}));
    });
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
      card_el.classList.add('card');
      card_el.innerText = cardToName(card);
      this.cards_container.appendChild(card_el);
      const card_img = document.createElement('img');
      card_img.classList.add('card-img');
      card_img.src = cardToImagePath(card);
      card_el.appendChild(card_img);
    }
  }

  betting() {
    if (!this.client_player) {
      return;
    }
    this.bet_input.valueAsNumber = 0;
    this.bet_input.max = this.player.cards.length.toString();
    this.betting_container.classList.add('show');
  }

  setBet(amount: number) {
    if (this.client_player) {
      this.betting_container.classList.remove('show');
    }
    this.player.bet = amount;
    this.bet_container.innerText = amount.toString();
  }

  playing() {
    if (!this.client_player) {
      return;
    }
    // TODO: implement
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
