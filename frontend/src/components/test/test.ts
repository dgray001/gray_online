import {DwgElement} from '../dwg_element';

import html from './test.html';

import '../test2/test';

class DwgTest extends DwgElement {
  test: HTMLElement;
  testy: HTMLElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('test', 'test');
    this.configureElement('testy', 'test-span');
  }

  protected override parsedCallback(): void {
    this.testy.innerHTML = "Hello World 1!";
  }
}

customElements.define('dwg-test', DwgTest);