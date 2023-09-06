import {DwgElement} from '../dwg_element';

import html from './test.html';

class DwgTest2 extends DwgElement {
  test: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('test', 'test-span');
  }

  protected override parsedCallback(): void {
    this.test.innerHTML = "Hello World 2!"
  }
}

customElements.define('dwg-test2', DwgTest2);