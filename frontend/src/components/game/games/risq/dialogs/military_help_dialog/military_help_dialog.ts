import { DialogSize, DwgDialogBox } from '../../../../../dialog_box/dialog_box';

import html from './military_help_dialog.html';

import './military_help_dialog.scss';

type MilitaryHelpDialogData = Record<string, never>;

export class DwgMilitaryHelpDialog extends DwgDialogBox<MilitaryHelpDialogData> {
  private close_button!: HTMLButtonElement;

  constructor() {
    super();
    this.configureElement('close_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): MilitaryHelpDialogData {
    return {};
  }

  setData(_data: MilitaryHelpDialogData, parsed?: boolean): void {
    this.classList.add(`size-${DialogSize.XLARGE}`);
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.close_button.onclick = (): void => this.closeDialog();
  }
}

customElements.define('dwg-military-help-dialog', DwgMilitaryHelpDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-military-help-dialog': DwgMilitaryHelpDialog;
  }
}
