import {DwgDialogBox} from '../dialog_box';

import html from './message_dialog.html';

import './message_dialog.scss';

/** Input data for a message dialog */
export declare interface MessageDialogData {
  message: string;
  button_text?: string;
}

export class DwgMessageDialog extends DwgDialogBox<MessageDialogData> {
  message_container: HTMLDivElement;
  ok_button: HTMLButtonElement;

  constructor() {
    super();
    this.configureElement('message_container');
    this.configureElement('ok_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): MessageDialogData {
    return {
      message: this.getAttribute('message') ?? '',
      button_text: this.getAttribute('button_text') ?? undefined,
    }
  }

  setData(data: MessageDialogData, parsed?: boolean) {
    this.setAttribute('message', data.message);
    if (data.button_text) {
      this.setAttribute('button_text', data.button_text);
    }
    if (!parsed && !this.fully_parsed) {
      return;
    }
    this.message_container.innerText = data.message;
    if (data.button_text) {
      this.ok_button.innerText = data.button_text;
    }
    this.ok_button.addEventListener('click', () => {
      this.closeDialog();
    });
  }
}

customElements.define('dwg-message-dialog', DwgMessageDialog);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-message-dialog': DwgMessageDialog;
  }
}
