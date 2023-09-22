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
  card_played_container: HTMLDivElement;
  card_played: HTMLImageElement;

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
    this.configureElement('cards_container');
    this.configureElement('betting_container');
    this.configureElement('bet_input');
    this.configureElement('bet_button');
    this.configureElement('card_played_container');
    this.configureElement('card_played');
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

  setHiddenCards(_: number) {
    this.newRound();
    this.player.cards = [];
    this.card_els = [];
  }

  setCards(cards: StandardCard[]) {
    this.newRound();
    this.player.cards = cards;
    this.card_els = [];
    for (const [i, card] of cards.entries()) {
      const card_el = document.createElement('div');
      card_el.classList.add('card');
      card_el.innerText = cardToName(card);
      this.cards_container.appendChild(card_el);
      const card_img = document.createElement('img');
      card_img.classList.add('card-img');
      card_img.src = cardToImagePath(card);
      card_el.appendChild(card_img);
      this.card_els.push(card_el);
      card_el.addEventListener('click', () => {
        if (!this.currently_playing) {
          return;
        }
        this.currently_playing = false;
        const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', `{"index":${i}}`, 'play-card');
        this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, 'bubbles': true}));
      });
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
    this.currently_playing = true;
  }

  playCard(index: number, card: StandardCard) {
    if (this.client_player) {
      this.currently_playing = false;
    }
    if (this.card_els.length) {
      this.card_els.splice(index, 1)[0].remove();
    }
    this.card_played.src = cardToImagePath(card);
    this.card_played_container.classList.add('show');
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
