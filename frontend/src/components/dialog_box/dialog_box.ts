import { DwgElement } from '../dwg_element';

import html from './dialog_box.html';

import './dialog_box.scss';

/** Shared sizing option for any dialog; maps to a `size-*` class on the dialog element */
export enum DialogSize {
  SMALL = 'small',
  MEDIUM = 'medium',
  LARGE = 'large',
  XLARGE = 'xlarge',
  XXLARGE = 'xxlarge',
}

export interface DialogHotkey {
  key?: string;
  ctrl?: boolean;
  shift?: boolean;
  alt?: boolean;
}

export abstract class DwgDialogBox<T> extends DwgElement {
  protected close_hotkey?: DialogHotkey;

  override async connectedCallback() {
    this.html_string = html.replace('id="content-container">', `id="content-container">${this.getHTML()}`);
    this.classList.add('dwg-dialog-box');
    await super.connectedCallback();
    window.addEventListener('keydown', this.handleCloseHotkey);
  }

  override disconnectedCallback() {
    window.removeEventListener('keydown', this.handleCloseHotkey);
    super.disconnectedCallback();
  }

  private handleCloseHotkey = (event: KeyboardEvent): void => {
    const hotkey = this.close_hotkey;
    const matches_close_hotkey =
      hotkey?.key === event.key.toLowerCase() &&
      hotkey.ctrl === event.ctrlKey &&
      hotkey.shift === event.shiftKey &&
      hotkey.alt === event.altKey;
    if (!event.repeat && (event.key === 'Escape' || matches_close_hotkey)) {
      event.preventDefault();
      this.closeDialog();
    }
  };

  protected override parsedCallback(): void {
    this.setData(this.getData(), true);
  }

  closeDialog() {
    this.remove();
  }

  abstract getHTML(): string;
  abstract getData(): T; // usually from attributes
  abstract setData(data: T, parsed?: boolean): void;
}
