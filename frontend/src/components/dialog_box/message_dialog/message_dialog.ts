import { DwgDialogBox, DialogSize } from '../dialog_box';

import html from './message_dialog.html';

import './message_dialog.scss';

/** Input data for a message dialog */
export declare interface MessageDialogData {
  message: string;
  button_text?: string;
  size?: DialogSize;
}

export class DwgMessageDialog extends DwgDialogBox<MessageDialogData> {
  private message_container!: HTMLDivElement;
  private ok_button!: HTMLButtonElement;

  constructor() {
    super();
    this.configureElements('message_container', 'ok_button');
  }

  override getHTML(): string {
    return html;
  }

  getData(): MessageDialogData {
    return {
      message: this.getAttribute('message') ?? '',
      button_text: this.getAttribute('button_text') ?? undefined,
      size: (this.getAttribute('size') as DialogSize) ?? DialogSize.MEDIUM,
    };
  }

  setData(data: MessageDialogData, parsed?: boolean) {
    this.setAttribute('message', data.message);
    this.setAttribute('size', data.size ?? DialogSize.MEDIUM);
    this.classList.add(`size-${data.size ?? DialogSize.MEDIUM}`);
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
