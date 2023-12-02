import {DwgDialogBox} from '../../dialog_box/dialog_box';

import html from './game_info_dialog.html';

import './game_info_dialog.scss';

/** Input data for a game info dialog box */
interface GameInfoData {
}

export class DwgGameInfoDialog extends DwgDialogBox<GameInfoData> {
  close_button: HTMLButtonElement;

  data: GameInfoData;

  constructor() {
    super();
    this.configureElement('close_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): GameInfoData {
    return this.data;
  }

  setData(data: GameInfoData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-game-info-dialog', DwgGameInfoDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-game-info-dialog': DwgGameInfoDialog;
  }
}
