import {DwgDialogBox} from '../../dialog_box/dialog_box';

import html from './players_dialog.html';

import './players_dialog.scss';

/** Input data for a players dialog box */
interface PlayersData {
  //
}

export class DwgPlayersDialog extends DwgDialogBox<PlayersData> {
  close_button: HTMLButtonElement;
  players_container: HTMLDivElement;

  data: PlayersData;

  constructor() {
    super();
    this.configureElement('close_button');
    this.configureElement('players_container');
  }

  override getHTML(): string {
    return html;
  }

  getData(): PlayersData {
    return this.data;
  }

  setData(data: PlayersData, parsed?: boolean) {
    this.data = data;
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-players-dialog', DwgPlayersDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-players-dialog': DwgPlayersDialog;
  }
}
