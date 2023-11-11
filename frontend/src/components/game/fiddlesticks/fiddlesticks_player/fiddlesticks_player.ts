import {DwgElement} from '../../../dwg_element';
import {createMessage} from '../../../lobby/data_models';
import {FiddlesticksPlayer} from '../fiddlesticks';
import {until} from '../../../../scripts/util';
import {messageDialog} from '../../game';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  name_container: HTMLDivElement;
  status_container: HTMLDivElement;
  score_container: HTMLSpanElement;
  bet_container: HTMLSpanElement;
  bet_input: HTMLInputElement;
  tricks_container: HTMLSpanElement;
  dealer_wrapper: HTMLDivElement;
  winner_wrapper: HTMLDivElement;

  player: FiddlesticksPlayer;
  client_player = false;
  card_els: HTMLDivElement[] = [];
  currently_playing = false; // if this player is playing a card

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('status_container');
    this.configureElement('score_container');
    this.configureElement('bet_container');
    this.configureElement('bet_input');
    this.configureElement('tricks_container');
    this.configureElement('dealer_wrapper');
    this.configureElement('winner_wrapper');
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

  async gameStarted(betting: boolean, current_turn: boolean) {
    await until(() => this.fully_parsed);
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
    this.bet_input.addEventListener('keyup', (e) => {
      if (e.key !== 'Enter') {
        return;
      }
      e.stopImmediatePropagation();
      const bet_value = this.bet_input.valueAsNumber;
      if (isNaN(bet_value) || bet_value < 0 || bet_value > this.player.cards.length) {
        messageDialog.call(this, {message: `Invalid bet value ${bet_value}; bet must be in the range of [0, ${this.player.cards.length}]`});
        return;
      }
      this.bet_input.disabled = true;
      const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', `{"amount":${this.bet_input.value}}`, 'bet');
      this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, bubbles: true}));
    });
  }

  newRound(dealer: boolean) {
    this.endRound(); // in case it wasn't called
    this.dealer_wrapper.classList.toggle('show', dealer);
    if (dealer) {
      this.status_container.innerText = 'Dealer';
    }
  }

  endRound() {
    this.player.bet = -1;
    this.player.tricks = 0;
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
    this.dealer_wrapper.classList.remove('show');
    this.status_container.innerText = '';
  }

  betting() {
    if (!this.client_player) {
      return;
    }
    this.bet_input.disabled = false;
    this.bet_input.valueAsNumber = 0;
    this.bet_input.max = this.player.cards.length.toString();
    this.bet_input.classList.add('show');
  }

  setBet(amount: number) {
    this.player.bet = amount;
    this.bet_container.innerText = amount.toString();
    if (this.client_player) {
      this.bet_input.classList.remove('show');
    }
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

  playCard() {
    if (this.client_player) {
      this.currently_playing = false;
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
    this.winner_wrapper.classList.add('show');
  }
}

customElements.define('dwg-fiddlesticks-player', DwgFiddlesticksPlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
