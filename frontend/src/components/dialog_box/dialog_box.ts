import { DwgElement } from '../dwg_element';
import { until } from '../../scripts/util';

import html from './dialog_box.html';

import './dialog_box.scss';

export abstract class DwgDialogBox<T> extends DwgElement {
  private content_container!: HTMLDivElement;

  constructor() {
    super();
    this.html_string = html;
    this.configureElement('content_container');
  }

  override async connectedCallback() {
    super.connectedCallback(); // don't await
    await until(() => {
      const container = this.querySelector<HTMLDivElement>('#content-container');
      if (container) {
        this.content_container = container;
      }
      return !!container;
    });
    this.classList.add('dwg-dialog-box');
    this.content_container.innerHTML = this.getHTML();
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
