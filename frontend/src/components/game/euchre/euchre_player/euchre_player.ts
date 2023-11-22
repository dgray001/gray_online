import { until } from '../../../../scripts/util';
import {DwgElement} from '../../../dwg_element';
import {EuchrePlayer, EuchreTeam} from '../euchre_data';

import html from './euchre_player.html';

import './euchre_player.scss';

export class DwgEuchrePlayer extends DwgElement {
  name_container: HTMLDivElement;
  status_container: HTMLDivElement;
  score_container: HTMLSpanElement;
  tricks_container: HTMLSpanElement;

  initialized = false;
  player: EuchrePlayer;
  team: EuchreTeam;
  client_player = false;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('status_container');
    this.configureElement('score_container');
    this.configureElement('tricks_container');
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

  async gameStarted(bidding: boolean, current_turn: boolean, dealer: boolean) {
    await until(() => this.fully_parsed);
    if (bidding) {
      this.tricks_container.innerText = '-';
      if (current_turn) {
        this.bidding();
      }
    } else {
      this.tricks_container.innerText = this.team.tricks.toString();
      if (current_turn) {
        this.playing();
      }
    }
    this.setDealer(dealer);
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
  }

  newRound(dealer: boolean) {
    this.endRound(); // in case it wasn't called
    this.setDealer(dealer);
  }

  endRound() {
    this.tricks_container.innerText = '-';
    this.status_container.innerText = '';
  }

  setDealer(dealer: boolean) {
    // TODO: implement
  }

  bidding() {
    // TODO: implement
  }

  setBid() {
    // TODO: implement
  }

  substitutingCard() {
    // TODO: implement
  }

  substitutedCard() {
    // TODO: implement
  }

  playing() {
    // TODO: implement
  }

  playCard() {
    // TODO: implement
  }
}

customElements.define('dwg-euchre-player', DwgEuchrePlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-euchre-player': DwgEuchrePlayer;
  }
}
