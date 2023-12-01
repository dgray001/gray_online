import {capitalize, until, untilTimer} from '../../../../../scripts/util';
import {DwgElement} from '../../../../dwg_element';
import {EuchrePlayer, EuchreTeam} from '../euchre_data';

import html from './euchre_player.html';

import './euchre_player.scss';

export class DwgEuchrePlayer extends DwgElement {
  name_container: HTMLDivElement;
  status_container: HTMLDivElement;
  score_container: HTMLSpanElement;
  tricks_container: HTMLSpanElement;
  icons_wrapper: HTMLDivElement;
  bid_animation: HTMLDivElement;
  bid_input_wrapper: HTMLDivElement;
  bid_button: HTMLButtonElement;
  pass_button: HTMLButtonElement;
  bid_choose_trump_input_wrapper: HTMLDivElement;
  trump_button_spades: HTMLButtonElement;
  trump_button_diamonds: HTMLButtonElement;
  trump_button_clubs: HTMLButtonElement;
  trump_button_hearts: HTMLButtonElement;
  trump_pass_button: HTMLButtonElement;

  initialized = false;
  player: EuchrePlayer;
  team: EuchreTeam;
  client_player = false;
  dealer = false;
  icons = new Map<string, HTMLDivElement>();

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('status_container');
    this.configureElement('score_container');
    this.configureElement('tricks_container');
    this.configureElement('icons_wrapper');
    this.configureElement('bid_animation');
    this.configureElement('bid_input_wrapper');
    this.configureElement('bid_button');
    this.configureElement('pass_button');
    this.configureElement('bid_choose_trump_input_wrapper');
    this.configureElement('trump_button_spades');
    this.configureElement('trump_button_diamonds');
    this.configureElement('trump_button_clubs');
    this.configureElement('trump_button_hearts');
    this.configureElement('trump_pass_button');
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

  async gameStarted(bidding: boolean, bidding_choosing_trump: boolean, dealer_substituting_card: boolean, current_turn: boolean, dealer: boolean) {
    await until(() => this.fully_parsed);
    this.setDealer(dealer);
    if (bidding || bidding_choosing_trump || dealer_substituting_card) {
      this.tricks_container.innerText = '-';
      if (dealer_substituting_card) {
        if (dealer) {
          this.substitutingCard();
        }
      } else if (current_turn) {
        this.bidding(bidding_choosing_trump, dealer);
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
    // TODO: add event listeners for bidding and passing
  }

  private toggleIcon(icon: string, src: string, tooltip: string, toggle: boolean) {
    if (toggle) {
      const icon_wrapper = document.createElement('div');
      const icon_el = document.createElement('img');
      icon_el.draggable = false;
      icon_el.alt = icon;
      icon_el.classList.add('icon');
      icon_el.src = `../../../../../images/icons/${src}32.png`;
      const tooltip_el = document.createElement('div');
      tooltip_el.classList.add('icon-tooltip');
      tooltip_el.innerText = tooltip;
      icon_wrapper.appendChild(icon_el);
      icon_wrapper.appendChild(tooltip_el);
      this.icons_wrapper.append(icon_wrapper);
      this.status_container.innerText = capitalize(icon);
    } else if (this.icons.has(icon)) {
      this.icons.get(icon).remove();
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
  }

  setDealer(dealer: boolean) {
    this.dealer = dealer;
    this.toggleIcon('dealer', 'cards', 'Dealt Round', dealer);
  }

  bidding(bidding_choosing_trump: boolean, is_dealer: boolean) {
    this.classList.add('turn');
    if (!this.client_player) {
      return;
    }
    if (bidding_choosing_trump) {
      this.bid_choose_trump_input_wrapper.classList.add('show');
      this.trump_pass_button.disabled = is_dealer;
    } else {
      this.bid_input_wrapper.classList.add('show');
    }
  }

  async setPassAnimation() {
    const animation_time = 500;
    this.bid_animation.innerText = '> Pass <';
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
      this.bid_choose_trump_input_wrapper.classList.remove('show');
      this.bid_input_wrapper.classList.remove('show');
    }
  }

  async setBidAnimation() {
    const animation_time = 500;
    this.bid_animation.innerText = this.dealer ? '> I\'ll take it! <' : '> Pick it up! <';
    this.bid_animation.style.transitionDuration = `${animation_time}ms`;
    this.bid_animation.classList.add('transition');
    await untilTimer(2 * animation_time);
    this.bid_animation.classList.remove('transition');
    await untilTimer(animation_time);
  }

  setBid(makers: boolean) {
    if (makers) {
      this.toggleIcon('maker', 'sword', 'Maker Team', true);
    } else {
      this.toggleIcon('defender', 'shield', 'Defender Team', true);
    }
    this.classList.remove('turn');
    if (this.client_player) {
      this.bid_choose_trump_input_wrapper.classList.remove('show');
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
