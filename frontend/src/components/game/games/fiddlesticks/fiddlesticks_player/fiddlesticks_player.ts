import {DwgElement} from '../../../../dwg_element';
import {createMessage} from '../../../../lobby/data_models';
import {FiddlesticksPlayer} from '../fiddlesticks_data';
import {until, untilTimer} from '../../../../../scripts/util';
import {messageDialog} from '../../../game';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  name_container: HTMLDivElement;
  status_container: HTMLDivElement;
  score_container: HTMLSpanElement;
  bet_container: HTMLSpanElement;
  bet_input_wrapper: HTMLDivElement;
  bet_input: HTMLInputElement;
  bet_button: HTMLButtonElement;
  tricks_container: HTMLSpanElement;
  dealer_wrapper: HTMLDivElement;
  winner_wrapper: HTMLDivElement;
  bet_animation: HTMLDivElement;

  initialized = false;
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
    this.configureElement('bet_input_wrapper');
    this.configureElement('bet_input');
    this.configureElement('bet_button');
    this.configureElement('tricks_container');
    this.configureElement('dealer_wrapper');
    this.configureElement('winner_wrapper');
    this.configureElement('bet_animation');
  }

  protected override parsedCallback(): void {
    if (!this.initialized) {
      throw new Error('Should initialize fiddlesticks player before attaching to dom');
    }
    this.name_container.innerText = this.player.player.nickname;
    this.score_container.innerText = this.player.score.toString();
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
    let dealer_timeout: NodeJS.Timeout = undefined;
    this.dealer_wrapper.addEventListener('click', () => {
      if (dealer_timeout) {
        clearTimeout(dealer_timeout);
      }
      this.dealer_wrapper.classList.toggle('show-tooltip');
      if (this.dealer_wrapper.classList.contains('show-tooltip')) {
        dealer_timeout = setTimeout(() => {
          this.dealer_wrapper.classList.remove('show-tooltip');
        }, 2000);
      }
    });
    let winner_timeout: NodeJS.Timeout = undefined;
    this.winner_wrapper.addEventListener('click', () => {
      if (winner_timeout) {
        clearTimeout(winner_timeout);
      }
      this.winner_wrapper.classList.toggle('show-tooltip');
      if (this.winner_wrapper.classList.contains('show-tooltip')) {
        winner_timeout = setTimeout(() => {
          this.winner_wrapper.classList.remove('show-tooltip');
        }, 2000);
      }
    });
  }

  initialize(player: FiddlesticksPlayer) {
    this.player = player;
    this.initialized = true;
  }

  async gameStarted(betting: boolean, current_turn: boolean, dealer: boolean) {
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
    this.setDealer(dealer);
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
    // can be called before parsed
    until(() => this.fully_parsed).then(() => {
      this.bet_input.addEventListener('keyup', (e) => {
        e.stopImmediatePropagation();
        if (e.key === 'Enter') {
          this.sendBetEvent();
        }
      });
      this.bet_button.addEventListener('click', () => {
        this.sendBetEvent();
      });
    });
  }

  sendBetEvent() {
    const bet_value = this.bet_input.valueAsNumber;
    if (isNaN(bet_value) || bet_value < 0 || bet_value > this.player.cards.length) {
      messageDialog.call(this, {message: `Invalid bet value ${bet_value}; bet must be in the range of [0, ${this.player.cards.length}]`});
      return;
    }
    this.bet_input.disabled = true;
    this.bet_button.disabled = true;
    const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', `{"amount":${this.bet_input.value}}`, 'bet');
    this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, bubbles: true}));
  }

  newRound(dealer: boolean) {
    this.endRound(); // in case it wasn't called
    this.setDealer(dealer);
  }

  setDealer(dealer: boolean) {
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
    this.classList.add('turn');
    if (!this.client_player) {
      return;
    }
    this.bet_input.disabled = false;
    this.bet_button.disabled = false;
    this.bet_input.valueAsNumber = 0;
    this.bet_input.max = this.player.cards.length.toString();
    this.bet_input_wrapper.classList.add('show');
  }

  async setBetAnimation(amount: number) {
    const animation_time = 500;
    this.bet_animation.innerText = amount.toString();
    this.bet_animation.style.transitionDuration = `${animation_time}ms`;
    this.bet_animation.classList.add('transition');
    await untilTimer(2 * animation_time);
    this.bet_animation.classList.remove('transition');
    await untilTimer(animation_time);
    this.setBet(amount);
  }

  setBet(amount: number) {
    this.player.bet = amount;
    this.bet_container.innerText = amount.toString();
    this.classList.remove('turn');
    if (this.client_player) {
      this.bet_input_wrapper.classList.remove('show');
    }
  }

  endBetting() {
    this.tricks_container.innerText = '0';
  }

  playing() {
    this.classList.add('turn');
    if (!this.client_player) {
      return;
    }
    this.currently_playing = true;
  }

  playCard() {
    this.classList.remove('turn');
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

declare global {
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-player': DwgFiddlesticksPlayer;
  }
}
