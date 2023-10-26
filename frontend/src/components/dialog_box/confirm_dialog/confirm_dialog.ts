import {DwgDialogBox} from '../dialog_box';

import html from './confirm_dialog.html';

import './confirm_dialog.scss';

/** Input data for a message dialog */
interface ConfirmDialogData {
  question: string;
  yes_text?: string;
  no_text?: string;
}

/** Will emit a 'confirmed' event if confirmed is clicked */
export class DwgConfirmDialog extends DwgDialogBox<ConfirmDialogData> {
  message_container: HTMLDivElement;
  yes_button: HTMLButtonElement;
  no_button: HTMLButtonElement;

  constructor() {
    super();
    this.configureElement('message_container');
    this.configureElement('yes_button');
    this.configureElement('no_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): ConfirmDialogData {
    return {
      question: this.getAttribute('question') ?? '',
      yes_text: this.getAttribute('yes_text') ?? 'yes',
      no_text: this.getAttribute('no_text') ?? 'no',
    }
  }

  setData(data: ConfirmDialogData, parsed?: boolean) {
    this.setAttribute('question', data.question);
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.message_container.innerText = data.question;
    if (data.yes_text) {
      this.yes_button.innerText = data.yes_text;
    }
    if (data.no_text) {
      this.no_button.innerText = data.no_text;
    }
    this.yes_button.addEventListener('click', () => {
      this.dispatchEvent(new Event('confirmed'));
      this.closeDialog();
    });
    this.no_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-confirm-dialog', DwgConfirmDialog);

declare global{
  interface HTMLElementTagNameMap {
    'dwg-confirm-dialog': DwgConfirmDialog;
  }
}
