import { DwgElement } from '../../../../dwg_element';
import { createMessage } from '../../../../lobby/data_models';
import type { FiddlesticksPlayer } from '../fiddlesticks_data';
import { DEV, until, untilTimer } from '../../../../../scripts/util';
import { messageDialog } from '../../../game';
import { Sounds } from '../../../../../sounds/Sounds';

import html from './fiddlesticks_player.html';

import './fiddlesticks_player.scss';

export class DwgFiddlesticksPlayer extends DwgElement {
  private name_container!: HTMLDivElement;
  private status_container!: HTMLDivElement;
  private score_container!: HTMLSpanElement;
  private bet_container!: HTMLSpanElement;
  private bet_input_wrapper!: HTMLDivElement;
  private bet_input!: HTMLInputElement;
  private bet_button!: HTMLButtonElement;
  private tricks_container!: HTMLSpanElement;
  private dealer_wrapper!: HTMLDivElement;
  private winner_wrapper!: HTMLDivElement;
  private bet_animation!: HTMLDivElement;

  private initialized = false;
  private player!: FiddlesticksPlayer;
  private client_player = false;

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'name_container',
      'status_container',
      'score_container',
      'bet_container',
      'bet_input_wrapper',
      'bet_input',
      'bet_button',
      'tricks_container',
      'dealer_wrapper',
      'winner_wrapper',
      'bet_animation'
    );
  }

  protected override parsedCallback(): void {
    if (!this.initialized) {
      throw new Error('Should initialize fiddlesticks player before attaching to dom');
    }
    this.name_container.innerText = this.player.player.nickname;
    this.score_container.innerText = this.player.score.toString();
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
    this.addTooltipToggle(this.dealer_wrapper);
    this.addTooltipToggle(this.winner_wrapper);
  }

  /** Toggles wrapper's tooltip on click, auto-hiding it after 2s */
  private addTooltipToggle(wrapper: HTMLDivElement) {
    let hide_timeout: NodeJS.Timeout | undefined = undefined;
    wrapper.addEventListener('click', () => {
      if (hide_timeout) {
        clearTimeout(hide_timeout);
      }
      wrapper.classList.toggle('show-tooltip');
      if (wrapper.classList.contains('show-tooltip')) {
        hide_timeout = setTimeout(() => {
          wrapper.classList.remove('show-tooltip');
        }, 2000);
      }
    });
  }

  initialize(player: FiddlesticksPlayer) {
    this.player = player;
    this.initialized = true;
  }

  async gameStarted(betting: boolean, current_turn: boolean, dealer: boolean, turn_start_time = 0, turn_duration = 0) {
    await until(() => this.fully_parsed);
    if (betting) {
      this.bet_container.innerText = this.player.has_bet ? this.player.bet.toString() : '-';
      this.tricks_container.innerText = '-';
      if (current_turn) {
        this.betting(turn_start_time, turn_duration);
      }
    } else {
      this.bet_container.innerText = this.player.bet.toString();
      this.tricks_container.innerText = this.player.tricks.toString();
      if (current_turn) {
        this.playing(turn_start_time, turn_duration);
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
    const bet_value = Number(this.bet_input.value);
    if (!Number.isInteger(bet_value) || bet_value < 0 || bet_value > this.player.cards.length) {
      messageDialog.call(this, {
        message: `Invalid bet value ${bet_value}; bet must be in the range of [0, ${this.player.cards.length}]`,
      });
      return;
    }
    this.bet_input.disabled = true;
    this.bet_button.disabled = true;
    const game_update = createMessage(
      `player-${this.player.player.player_id}`,
      'game-update',
      `{"amount":${bet_value}}`,
      'bet'
    );
    this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
  }

  async newRound(dealer: boolean) {
    await this.endRound(); // in case it wasn't called
    this.setDealer(dealer);
  }

  setDealer(dealer: boolean) {
    this.dealer_wrapper.classList.toggle('show', dealer);
    if (dealer) {
      this.status_container.innerText = 'Dealer';
    }
  }

  async endRound() {
    await until(() => this.fully_parsed);
    this.player.bet = -1;
    this.player.tricks = 0;
    this.bet_container.innerText = '-';
    this.tricks_container.innerText = '-';
    this.dealer_wrapper.classList.remove('show');
    this.status_container.innerText = '';
  }

  betting(turn_start_time: number, turn_duration: number) {
    this.classList.add('turn');
    this.startTimer(turn_start_time, turn_duration);
    if (!this.client_player) {
      return;
    }
    Sounds.play('turn_notification');
    this.bet_input.disabled = false;
    this.bet_button.disabled = false;
    this.bet_input.value = DEV ? '0' : '';
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

  private timer_request?: number;
  private startTimer(turn_start_time: number, turn_duration: number) {
    this.stopTimer();
    const tick = () => {
      const elapsed = Date.now() - turn_start_time;
      const turn_percent_left = Math.max(0, 100 - (elapsed / turn_duration) * 100);
      this.style.setProperty('--turn-percent-left', turn_percent_left.toString());
      if (turn_percent_left > 0 && this.classList.contains('turn')) {
        this.timer_request = requestAnimationFrame(tick);
      }
    };
    this.timer_request = requestAnimationFrame(tick);
  }

  private stopTimer() {
    if (this.timer_request) {
      cancelAnimationFrame(this.timer_request);
    }
    this.style.setProperty('--turn-percent-left', '0');
  }

  playing(turn_start_time: number, turn_duration: number) {
    this.classList.add('turn');
    this.startTimer(turn_start_time, turn_duration);
    if (!this.client_player) {
      return;
    }
    Sounds.play('turn_notification');
  }

  playCard() {
    this.classList.remove('turn');
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
