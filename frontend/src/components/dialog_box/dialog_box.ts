import { DwgElement } from '../dwg_element';

import html from './dialog_box.html';

import './dialog_box.scss';

/** Shared sizing option for any dialog; maps to a `size-*` class on the dialog element */
export enum DialogSize {
  SMALL = 'small',
  MEDIUM = 'medium',
  LARGE = 'large',
}

export abstract class DwgDialogBox<T> extends DwgElement {
  override async connectedCallback() {
    this.html_string = html.replace('id="content-container">', `id="content-container">${this.getHTML()}`);
    this.classList.add('dwg-dialog-box');
    await super.connectedCallback();
  }

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
