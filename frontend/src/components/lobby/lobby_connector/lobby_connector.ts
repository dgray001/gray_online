import {DwgElement} from '../../dwg_element';
import {clickButton} from '../../../scripts/util';

import html from './lobby_connector.html';
import './lobby_connector.scss';

export class DwgLobbyConnector extends DwgElement {
  card: HTMLDivElement;
  nickname: HTMLInputElement;
  connect_button: HTMLButtonElement;
  status_message: HTMLDivElement;

  submitted = new Event('submitted');

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('card');
    this.configureElement('nickname');
    this.configureElement('connect_button');
    this.configureElement('status_message');
  }

  protected override parsedCallback(): void {
    this.connect_button.disabled = true;
    this.nickname.addEventListener('keyup', (e) => {
      const valid_name = this.validName(this.nickname.value);
      this.connect_button.disabled = !valid_name;
      if (valid_name && e.key === 'Enter') {
        this.dispatchEvent(this.submitted);
      }
    });
    clickButton(this.connect_button, () => {
      this.dispatchEvent(this.submitted);
      return 'Connecting ...';
    }, {re_enable_button: false});
  }

  private validName(name: string): boolean {
    return !!name.length;
  }
}

customElements.define('dwg-lobby-connector', DwgLobbyConnector);
