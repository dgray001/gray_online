import {DwgElement} from '../DwgElement';

import html from './test.html';

class DwgTest extends DwgElement {
  test: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('test', 'test-span');
  }

  protected override parsedCallback(): void {
    this.test.innerHTML = "Hello World!"
  }
}

customElements.define('dwg-test', DwgTest);