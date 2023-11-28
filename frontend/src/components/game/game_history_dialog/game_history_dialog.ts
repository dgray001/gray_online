import {DwgDialogBox} from '../../dialog_box/dialog_box';
import {UpdateMessage} from '../data_models';

import html from './game_history_dialog.html';

import './game_history_dialog.scss';

/** Input data for a game history dialog box */
interface GameHistoryData {
  updates: Map<number, UpdateMessage>;
  // TODO: add field for game history entry type (one for each game)
}

export class DwgGameHistoryDialog extends DwgDialogBox<GameHistoryData> {
  close_button: HTMLButtonElement;
  updates_container: HTMLDivElement;

  data: GameHistoryData;

  constructor() {
    super();
    this.configureElement('close_button');
    this.configureElement('updates_container');
  }

  override getHTML(): string {
    return html;
  }

  getData(): GameHistoryData {
    return this.data;
  }

  setData(data: GameHistoryData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    const sorted_updates = [...data.updates.values()].sort((a, b) => a.update_id - b.update_id);
    for (const update of sorted_updates) {
      const update_el = document.createElement('div');
      update_el.innerText = `ID: ${update.update_id}, Kind: ${update.kind}, data: ${JSON.stringify(update.update)}`;
      this.updates_container.append(update_el);
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
