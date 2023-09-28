import {DwgElement} from '../dwg_element';

import html from './dialog_box.html';

import './dialog_box.scss';

export abstract class DwgDialogBox<T> extends DwgElement {
  content_container: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('content_container');
  }

  protected override parsedCallback(): void {
    this.content_container.appendChild(this.getContent());
    this.setData(this.getData());
  }

  abstract getContent(): Node;
  abstract getData(): T; // usually from attributes
  abstract setData(data: T): void;
}
