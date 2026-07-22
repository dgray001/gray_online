import { DwgElement } from '../../../dwg_element';
import type { GameComponent, UpdateMessage } from '../../data_models';
import type { DwgGame } from '../../game';

import html from './egyptian_rat_slap.html';
import type { GameEgyptianRatSlap } from './egyptian_rat_slap_data';

import './egyptian_rat_slap.scss';

export class DwgEgyptianRatSlap extends DwgElement implements GameComponent {
  private game!: GameEgyptianRatSlap;
  private player_id: number = -1;

  constructor() {
    super();
    this.html_string = html;
  }

  async initialize(abstract_game: DwgGame, game: GameEgyptianRatSlap): Promise<void> {
    this.player_id = abstract_game.playerId();
    this.game = game;
  }

  async gameUpdate(update: UpdateMessage): Promise<void> {
    try {
      switch (update.kind) {
        default:
          console.log(`Unknown game update type ${update.kind}`);
          break;
      }
    } catch (e) {
      console.log(`Error during game update ${JSON.stringify(update)}: ${e}`);
    }
  }

  updateDialogComponent(update: UpdateMessage): HTMLElement {
    const update_el = document.createElement('div');
    update_el.innerText = `ID: ${update.update_id}, Kind: ${update.kind}, data: ${JSON.stringify(update.content)}`;
    return update_el;
  }
}

customElements.define('dwg-egyptian-rat-slap', DwgEgyptianRatSlap);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-egyptian-rat-slap': DwgEgyptianRatSlap;
  }
}
