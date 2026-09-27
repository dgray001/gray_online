import { DwgElement } from '../../../../dwg_element';

import html from './fiddlesticks_settings.html';

import './fiddlesticks_settings.scss';

export class DwgFiddlesticksSettings extends DwgElement {
  constructor() {
    super();
    this.html_string = html;
  }
}

customElements.define('dwg-fiddlesticks-settings', DwgFiddlesticksSettings);

declare global {
  interface HTMLElementTagNameMap {
    'dwg-fiddlesticks-settings': DwgFiddlesticksSettings;
  }
}
