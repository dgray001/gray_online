import { DwgElement } from '../../../dwg_element';
import type { GameComponent, UpdateMessage } from '../../data_models';
import type { DwgGame } from '../../game';
import { createMessage } from '../../../lobby/data_models';

import html from './test_game.html';
import type { GameTestGame } from './test_game_data';

import './test_game.scss';

export class DwgTestGame extends DwgElement implements GameComponent {
  private show_info!: HTMLButtonElement;
  private end_game!: HTMLButtonElement;

  private game!: GameTestGame;
  private player_id: number = -1;

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('show_info');
    this.configureElement('end_game');
  }

  protected override parsedCallback(): void {
    this.show_info.addEventListener('click', () => {
      console.log(this.game);
    });
    this.end_game.addEventListener('click', () => {
      const game_update = createMessage('player', 'game-update', '', 'end_game');
      this.dispatchEvent(new CustomEvent('game_update', { detail: game_update, bubbles: true }));
    });
  }

  async initialize(abstract_game: DwgGame, game: GameTestGame): Promise<void> {
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

customElements.define('dwg-test-game', DwgTestGame);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-test-game': DwgTestGame;
  }
}
