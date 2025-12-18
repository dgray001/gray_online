import { DwgDialogBox } from '../../dialog_box/dialog_box';
import type { GameComponent, UpdateMessage } from '../data_models';

import html from './game_history_dialog.html';

import './game_history_dialog.scss';

/** Input data for a game history dialog box */
interface GameHistoryData {
  updates: Map<number, UpdateMessage>;
  game: GameComponent;
}

export class DwgGameHistoryDialog extends DwgDialogBox<GameHistoryData> {
  private close_button!: HTMLButtonElement;
  private updates_container!: HTMLDivElement;

  private data!: GameHistoryData;

  constructor() {
    super();
    this.configureElements('close_button', 'updates_container');
  }

  override getHTML(): string {
    return html;
  }

  override getData(): GameHistoryData {
    return this.data;
  }

  override setData(data: GameHistoryData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    const sorted_updates = [...data.updates.values()].sort((a, b) => b.update_id - a.update_id);
    for (const update of sorted_updates) {
      const el = data.game.updateDialogComponent(update);
      el.classList.add('update');
      this.updates_container.appendChild(el);
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-game-history-dialog', DwgGameHistoryDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-game-history-dialog': DwgGameHistoryDialog;
  }
}
