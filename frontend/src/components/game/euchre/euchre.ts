import {DwgElement} from '../../dwg_element';
import {GameComponent, UpdateMessage} from '../data_models';

import html from './euchre.html';
import {DwgEuchrePlayer} from './euchre_player/euchre_player';
import {GameEuchre} from './euchre_data';

import './euchre.scss';
import './euchre_player/euchre_player';
import '../../dialog_box/message_dialog/message_dialog';
import '../util/card_hand/card_hand';

export class DwgEuchre extends DwgElement implements GameComponent {
  player_container: HTMLDivElement;

  game: GameEuchre;
  player_els: DwgEuchrePlayer[] = [];
  player_id: number = -1;
  trick_card_els: HTMLDivElement[] = [];

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('player_container');
  }

  protected override parsedCallback(): void {
    console.log('DwgEuchre parsed!');
  }

  initialize(game: GameEuchre, client_id: number): void {
    this.game = game;
    this.player_els = [];
    for (const [player_id, player] of game.players.entries()) {
      const player_el = document.createElement('dwg-euchre-player');
      player_el.initialize(player);
      this.player_container.appendChild(player_el);
      this.player_els.push(player_el);
      if (player.player.client_id === client_id) {
        this.player_id = player_id;
        player_el.setClientPlayer();
      }
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
