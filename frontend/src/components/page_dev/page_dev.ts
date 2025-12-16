import {DwgElement} from '../dwg_element';

import html from './page_dev.html';

import './page_dev.scss';

export class DwgPageDev extends DwgElement {
  private example: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('example');
  }

  protected override parsedCallback(): void {
    console.log('DwgPageDev parsed!');
  }
}

customElements.define('dwg-page-dev', DwgPageDev);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-page-dev': DwgPageDev;
  }
}
