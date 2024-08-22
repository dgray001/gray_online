import {DwgElement} from "../../../dwg_element";
import {GameComponent, UpdateMessage} from "../../data_models";
import {DwgGame} from "../../game";

import html from './test_game.html';
import {GameTestGame} from "./test_game_data";

import './test_game.scss';

export class DwgTestGame extends DwgElement implements GameComponent {

  constructor() {
    super();
    this.htmlString = html;
  }

  protected override parsedCallback(): void {
  }

  async initialize(abstract_game: DwgGame, game: GameTestGame): Promise<void> {
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

customElements.define('dwg-test-game', DwgTestGame);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-test-game': DwgTestGame;
  }
}
