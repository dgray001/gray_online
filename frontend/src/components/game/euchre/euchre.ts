import {DwgElement} from '../../dwg_element';
import {GameComponent, UpdateMessage} from '../data_models';
import {DwgCardHand} from '../util/card_hand/card_hand';
import {createMessage} from '../../lobby/data_models';
import {clientOnMobile} from '../../../scripts/util';

import html from './euchre.html';
import {DwgEuchrePlayer} from './euchre_player/euchre_player';
import {GameEuchre, getPlayersTeam} from './euchre_data';

import './euchre.scss';
import './euchre_player/euchre_player';
import '../../dialog_box/message_dialog/message_dialog';
import '../util/card_hand/card_hand';

export class DwgEuchre extends DwgElement implements GameComponent {
  round_number: HTMLSpanElement;
  trick_number: HTMLSpanElement;
  status_container: HTMLSpanElement;
  trick_cards: HTMLDivElement;
  table_container: HTMLDivElement;
  player_container: HTMLDivElement;
  players_cards: DwgCardHand;

  game: GameEuchre;
  player_els: DwgEuchrePlayer[] = [];
  player_id: number = -1;
  trick_card_els: HTMLDivElement[] = [];

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('round_number');
    this.configureElement('trick_number');
    this.configureElement('status_container');
    this.configureElement('player_container');
    this.configureElement('trick_cards');
    this.configureElement('table_container');
    this.configureElement('player_container');
    this.configureElement('players_cards');
  }

  protected override parsedCallback(): void {
    this.round_number.innerText = '-';
    this.trick_number.innerText = '-';
    this.status_container.innerText = 'Starting game ...';
    this.players_cards.addEventListener('play_card', (e: CustomEvent<number>) => {
      // TODO: check if card is playable
      const game_update = createMessage(`player-${this.player_id}`, 'game-update', `{"index":${e.detail}}`, 'play-card');
      this.dispatchEvent(new CustomEvent('game_update', {'detail': game_update, 'bubbles': true}));
    });
    if (clientOnMobile()) {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table400.png)');
    } else {
      this.table_container.style.setProperty('background-image', 'url(/images/card_table800.png)');
    }
  }

  initialize(game: GameEuchre, client_id: number): void {
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-euchre-player');
      player_el.initialize(player, getPlayersTeam(this.game, player_id));
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (player.player.client_id === client_id) {
        this.player_id = player_id;
        player_el.setClientPlayer();
      }
    }
    // TODO: if view just don't use player_id
    if (this.player_id > -1) {
      for (let i = 0; i < this.player_els.length; i++) {
        let id = this.player_id + i;
        if (id >= this.player_els.length) {
          id -= this.player_els.length;
        }
        this.game.players[id].order = i;
        this.player_els[id].style.setProperty('--order', i.toString());
        this.player_els[id].style.setProperty('--num-players', this.player_els.length.toString());
      }
    }
    this.trick_cards.style.setProperty('--num-players', this.player_els.length.toString());
    if (game.game_base.game_ended) {
      // TODO: show ended game state
    }
    else if (game.game_base.game_started) {
      // TODO: implement
    }
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch(update.kind) {
        default:
          console.log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch(e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }
}

customElements.define('dwg-euchre', DwgEuchre);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-euchre': DwgEuchre;
  }
}
