import {DwgElement} from '../dwg_element';
import {DwgLobby} from '../lobby/lobby';
import '../lobby/lobby';

import html from './page_home.html';
import './page_home.scss';

export class DwgPageHome extends DwgElement {
  lobby: DwgLobby;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('lobby', 'lobby');
  }

  protected override parsedCallback(): void {}
}

customElements.define('dwg-page-home', DwgPageHome);
