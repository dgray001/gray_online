import { DwgDialogBox, DialogSize } from '../dialog_box';
import { GameType } from '../../lobby/data_models';

import html from './settings_dialog.html';

import './settings_dialog.scss';
import '../../game/games/fiddlesticks/fiddlesticks_settings/fiddlesticks_settings';
import '../../game/games/risq/dialogs/risq_settings/risq_settings';

export declare interface SettingsDialogData {
  game_type: GameType;
}

export class DwgSettingsDialog extends DwgDialogBox<SettingsDialogData> {
  private settings_body!: HTMLDivElement;
  private close_button!: HTMLButtonElement;

  private data: SettingsDialogData = { game_type: GameType.UNSPECIFIED };

  constructor() {
    super();
    this.configureElements('settings_body', 'close_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): SettingsDialogData {
    return this.data;
  }

  setData(data: SettingsDialogData, parsed?: boolean): void {
    this.data = data;
    const size = this.resolveSize(data);
    this.classList.add(`size-${size}`);
    if (!parsed && !this.fully_parsed) {
      return;
    }
    switch (data.game_type) {
      case GameType.FIDDLESTICKS:
        this.settings_body.replaceChildren(document.createElement('dwg-fiddlesticks-settings'));
        break;
      case GameType.RISQ:
        this.settings_body.replaceChildren(document.createElement('dwg-risq-settings'));
        break;
      default:
        this.settings_body.innerText = 'No settings for this game';
        break;
    }
    this.close_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }

  private resolveSize(data: SettingsDialogData): DialogSize {
    switch (data.game_type) {
      case GameType.FIDDLESTICKS:
      case GameType.RISQ:
        return DialogSize.XLARGE;
      default:
        return DialogSize.SMALL;
    }
  }
}

customElements.define('dwg-settings-dialog', DwgSettingsDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-settings-dialog': DwgSettingsDialog;
  }
}
