import {DwgElement} from '../../dwg_element';
import {clickButton} from '../../../scripts/util';
import {SERVER_CHAT_NAME} from '../../chatbox/chatbox';

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
    try {
      const previous_nickname = localStorage.getItem("client_nickname");
      if (previous_nickname) {
        this.nickname.value = previous_nickname;
        this.validateName();
      }
    } catch(e) {} // if local storage isn't accessible
    this.nickname.addEventListener('keyup', (e) => {
      const valid_name = this.validateName();
      if (valid_name && e.key === 'Enter') {
        this.dispatchEvent(this.submitted);
      }
    });
    clickButton(this.connect_button, () => {
      this.dispatchEvent(this.submitted);
      return 'Connecting ...';
    }, {re_enable_button: false});
  }

  invalid_names = [SERVER_CHAT_NAME];

  private validateName(): boolean {
    if (this.invalid_names.includes(this.nickname.value)) {
      return false;
    }
    const valid_name = this.nickname.value.length > 1 && this.nickname.value.length < 17;
    this.connect_button.disabled = !valid_name;
    return valid_name;
  }
}

customElements.define('dwg-lobby-connector', DwgLobbyConnector);
