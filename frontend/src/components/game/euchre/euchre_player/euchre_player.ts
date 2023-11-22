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
    this.initialized = true;
  }

  setClientPlayer() {
    this.classList.add('client-player');
    this.client_player = true;
  }
}

customElements.define('dwg-euchre-player', DwgEuchrePlayer);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-euchre-player': DwgEuchrePlayer;
  }
}
