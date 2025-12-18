import { capitalize, until, untilTimer } from '../../../../../scripts/util';
import { DwgElement } from '../../../../dwg_element';
import { createMessage } from '../../../../lobby/data_models';
import type { EuchrePlayer, EuchreTeam, GameEuchre } from '../euchre_data';

import html from './euchre_player.html';

import './euchre_player.scss';

export class DwgEuchrePlayer extends DwgElement {
  private name_container!: HTMLDivElement;
  private status_container!: HTMLDivElement;
  private score_container!: HTMLSpanElement;
  private tricks_container!: HTMLSpanElement;
  private icons_wrapper!: HTMLDivElement;
  private bid_animation!: HTMLDivElement;
  private bid_input_wrapper!: HTMLDivElement;
  private bid_input_message!: HTMLDivElement;
  private going_alone!: HTMLInputElement;
  private bid_button!: HTMLButtonElement;
  private spades_button!: HTMLButtonElement;
  private diamonds_button!: HTMLButtonElement;
  private clubs_button!: HTMLButtonElement;
  private hearts_button!: HTMLButtonElement;
  private pass_button!: HTMLButtonElement;

  private initialized = false;
  private player!: EuchrePlayer;
  private team!: EuchreTeam;
  private client_player = false;
  private dealer = false;
  private icons = new Map<string, HTMLDivElement>();

  constructor() {
    super();
    this.html_string = html;
    this.configureElements(
      'name_container',
      'status_container',
      'score_container',
      'tricks_container',
      'icons_wrapper',
      'bid_animation',
      'bid_input_wrapper',
      'bid_input_message',
      'going_alone',
      'bid_button',
      'spades_button',
      'diamonds_button',
      'clubs_button',
      'hearts_button',
      'pass_button'
    );
  }

  protected override parsedCallback(): void {
    if (!this.initialized) {
      throw new Error('Should initialize euchre player before attaching to dom');
    }
    this.name_container.innerText = this.player.player.nickname;
    this.score_container.innerText = this.team.score.toString();
    this.tricks_container.innerText = '-';
  }

  initialize(player: EuchrePlayer, team: EuchreTeam) {
    this.player = player;
    this.team = team;
    this.classList.add(`team${team.team_id}`);
    this.initialized = true;
  }

  async gameStarted(game: GameEuchre, current_turn: boolean, dealer: boolean) {
    await until(() => this.fully_parsed);
    this.setDealer(dealer);
    if (game.bidding || game.bidding_choosing_trump || game.dealer_substituting_card) {
      this.tricks_container.innerText = '-';
      if (game.dealer_substituting_card) {
        if (dealer) {
          this.substitutingCard();
        }
      } else if (current_turn) {
        this.bidding(game.bidding_choosing_trump, dealer, game.card_face_up?.suit);
      }
    } else {
      this.tricks_container.innerText = this.team.tricks.toString();
      if (current_turn) {
        this.playing();
      }
    }
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
    // can be called before parsed
    until(() => this.fully_parsed).then(() => {
      this.bid_button.addEventListener('click', () => {
        this.disableButtons();
        const update = { going_alone: this.going_alone.checked };
        const game_update = createMessage(
          `player-${this.player.player.player_id}`,
          'game-update',
          JSON.stringify(update),
          'bid'
        );
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
      this.spades_button.addEventListener('click', () => {
        this.disableButtons();
        const update = { going_alone: this.going_alone.checked, trump_suit: 4 };
        const game_update = createMessage(
          `player-${this.player.player.player_id}`,
          'game-update',
          JSON.stringify(update),
          'bid-choose-trump'
        );
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
      this.diamonds_button.addEventListener('click', () => {
        this.disableButtons();
        const update = { going_alone: this.going_alone.checked, trump_suit: 1 };
        const game_update = createMessage(
          `player-${this.player.player.player_id}`,
          'game-update',
          JSON.stringify(update),
          'bid-choose-trump'
        );
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
      this.clubs_button.addEventListener('click', () => {
        this.disableButtons();
        const update = { going_alone: this.going_alone.checked, trump_suit: 2 };
        const game_update = createMessage(
          `player-${this.player.player.player_id}`,
          'game-update',
          JSON.stringify(update),
          'bid-choose-trump'
        );
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
      this.hearts_button.addEventListener('click', () => {
        this.disableButtons();
        const update = { going_alone: this.going_alone.checked, trump_suit: 3 };
        const game_update = createMessage(
          `player-${this.player.player.player_id}`,
          'game-update',
          JSON.stringify(update),
          'bid-choose-trump'
        );
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
      this.pass_button.addEventListener('click', () => {
        this.disableButtons();
        const game_update = createMessage(`player-${this.player.player.player_id}`, 'game-update', '{}', 'pass');
        this.dispatchEvent(
          new CustomEvent('game_update', {
            detail: game_update,
            bubbles: true,
          })
        );
      });
    });
  }

  private disableButtons() {
    this.bid_button.disabled = true;
    this.spades_button.disabled = true;
    this.diamonds_button.disabled = true;
    this.clubs_button.disabled = true;
    this.hearts_button.disabled = true;
    this.pass_button.disabled = true;
  }

  private toggleIcon(icon: string, src: string, tooltip: string, toggle: boolean) {
    if (toggle) {
      const icon_wrapper = document.createElement('div');
      icon_wrapper.classList.add('icon-wrapper');
      const icon_el = document.createElement('img');
      icon_el.draggable = false;
      icon_el.alt = icon;
      icon_el.classList.add('icon');
      icon_el.src = `/images/icons/${src}32.png`;
      const tooltip_el = document.createElement('div');
      tooltip_el.classList.add('icon-tooltip');
      tooltip_el.innerText = tooltip;
      icon_wrapper.appendChild(icon_el);
      icon_wrapper.appendChild(tooltip_el);
      this.icons_wrapper.append(icon_wrapper);
      this.status_container.innerText = capitalize(icon);
      this.icons.set(icon, icon_wrapper);
    } else if (this.icons.has(icon)) {
      this.icons.get(icon)?.remove();
      this.icons.delete(icon);
    }
  }

  newRound(dealer: boolean) {
    this.endRound(); // in case it wasn't called
    this.setDealer(dealer);
  }

  endRound() {
    this.tricks_container.innerText = '-';
    this.status_container.innerText = '';
    this.toggleIcon('dealer', '', '', false);
    this.toggleIcon('maker', '', '', false);
    this.toggleIcon('defender', '', '', false);
    this.toggleIcon('alone', '', '', false);
    this.classList.remove('going-alone-ally');
  }

  setDealer(dealer: boolean) {
    this.dealer = dealer;
    this.toggleIcon('dealer', 'cards', 'Dealt Round', dealer);
  }

  bidding(bidding_choosing_trump: boolean, is_dealer: boolean, card_face_up_suit?: number) {
    this.classList.add('turn');
    if (!this.client_player) {
      return;
    }
    this.bid_input_wrapper.classList.add('show');
    this.bid_input_wrapper.classList.toggle('choosing-trump', !!bidding_choosing_trump);
    this.going_alone.checked = false;
    if (bidding_choosing_trump) {
      this.bid_input_message.innerText = 'Choose Trump';
      this.spades_button.disabled = false;
      this.spades_button.classList.remove('disabled');
      this.diamonds_button.disabled = false;
      this.diamonds_button.classList.remove('disabled');
      this.clubs_button.disabled = false;
      this.clubs_button.classList.remove('disabled');
      this.hearts_button.disabled = false;
      this.hearts_button.classList.remove('disabled');
      switch (card_face_up_suit) {
        case 1:
          this.diamonds_button.disabled = true;
          this.diamonds_button.classList.add('disabled');
          break;
        case 2:
          this.clubs_button.disabled = true;
          this.clubs_button.classList.add('disabled');
          break;
        case 3:
          this.hearts_button.disabled = true;
          this.hearts_button.classList.add('disabled');
          break;
        case 4:
          this.spades_button.disabled = true;
          this.spades_button.classList.add('disabled');
          break;
        default:
          break;
      }
      this.pass_button.disabled = is_dealer;
    } else {
      this.bid_input_message.innerText = 'Bid or Pass';
      this.bid_button.disabled = false;
      this.pass_button.disabled = false;
    }
  }

  async setPassAnimation() {
    const animation_time = 500;
    this.bid_animation.innerText = 'Pass';
    this.bid_animation.style.transitionDuration = `${animation_time}ms`;
    this.bid_animation.classList.add('transition');
    await untilTimer(2 * animation_time);
    this.bid_animation.classList.remove('transition');
    await untilTimer(animation_time);
    this.setPass();
  }

  setPass() {
    this.classList.remove('turn');
    if (this.client_player) {
      this.bid_input_wrapper.classList.remove('show');
    }
  }

  async setBidAnimation(going_alone: boolean, trump_suit_name?: string) {
    const animation_time = 500;
    if (going_alone) {
      this.bid_animation.innerText = 'Going alone!';
    } else if (!!trump_suit_name) {
      this.bid_animation.innerText = `${trump_suit_name}s`;
    } else {
      this.bid_animation.innerText = this.dealer ? "I'll take it!" : 'Pick it up!';
    }
    this.bid_animation.style.transitionDuration = `${animation_time}ms`;
    this.bid_animation.classList.add('transition');
    await untilTimer(2 * animation_time);
    this.bid_animation.classList.remove('transition');
    await untilTimer(animation_time);
  }

  setBid(makers: boolean, going_alone?: boolean, going_alone_ally?: boolean) {
    if (makers) {
      this.toggleIcon('maker', 'sword', 'Maker Team', true);
    } else {
      this.toggleIcon('defender', 'shield', 'Defender Team', true);
    }
    if (going_alone) {
      if (going_alone_ally) {
        this.classList.add('going-alone-ally');
        this.status_container.innerText = '';
      } else {
        this.toggleIcon('alone', 'fist', 'Going Alone', true);
      }
    }
    this.classList.remove('turn');
    if (this.client_player) {
      this.bid_input_wrapper.classList.remove('show');
    }
  }

  substitutingCard() {
    this.classList.add('turn');
  }

  substitutedCard() {
    this.classList.remove('turn');
  }

  endBidding() {
    this.tricks_container.innerText = '0';
  }

  playing() {
    this.classList.add('turn');
  }

  playCard() {
    this.classList.remove('turn');
  }

  endTrick(tricks: number) {
    this.team.tricks = tricks;
    this.tricks_container.innerText = tricks.toString();
  }

  setScore(score: number) {
    this.team.score = score;
    this.score_container.innerText = score.toString();
  }

  wonGame() {
    this.toggleIcon('winner', 'crown', 'Won Game', true);
  }
}

customElements.define('dwg-euchre-player', DwgEuchrePlayer);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-euchre-player': DwgEuchrePlayer;
  }
}
