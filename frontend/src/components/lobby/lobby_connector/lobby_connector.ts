import {DwgElement} from '../../dwg_element';
import {clickButton} from '../../../scripts/util';
import {SERVER_CHAT_NAME} from '../../chatbox/chatbox';

import html from './lobby_connector.html';
import './lobby_connector.scss';

/** Connection Data */
export declare interface ConnectData {
  nickname: string;
  try_reconnect: boolean;
  client_id?: number; // defined if try_reconnect is true
}

export class DwgLobbyConnector extends DwgElement {
  card: HTMLDivElement;
  reconnect_wrapper: HTMLDivElement;
  previous_nickname: HTMLDivElement;
  reconnect_button: HTMLButtonElement;
  new_connection_button: HTMLButtonElement;
  connect_wrapper: HTMLDivElement;
  nickname: HTMLInputElement;
  connect_button: HTMLButtonElement;
  status_message: HTMLDivElement;

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('card');
    this.configureElement('reconnect_wrapper');
    this.configureElement('previous_nickname');
    this.configureElement('reconnect_button');
    this.configureElement('new_connection_button');
    this.configureElement('connect_wrapper');
    this.configureElement('nickname');
    this.configureElement('connect_button');
    this.configureElement('status_message');
  }

  protected override parsedCallback(): void {
    this.connect_button.disabled = true;
    let try_reconnect = false;
    try {
      const client_id_time = parseInt(localStorage.getItem("client_id_time"));
      const client_id = parseInt(localStorage.getItem("client_id"));
      const previous_nickname = localStorage.getItem("client_nickname");
      if (!!previous_nickname) {
        this.nickname.value = previous_nickname;
        const previous_name_valid = this.validateName();
        if (previous_name_valid && !!client_id && !!client_id_time &&
          (Date.now() - client_id_time < 1000 * 60 * 60 * 24))
        {
          this.previous_nickname.innerText = previous_nickname;
          clickButton(this.reconnect_button, () => {
            this.connect({nickname: previous_nickname, try_reconnect: true, client_id});
            return 'Reconnecting ...';
          }, {re_enable_button: false});
          clickButton(this.new_connection_button, () => {
            this.reconnect_wrapper.classList.remove('show');
            this.connect_wrapper.classList.add('show');
          });
          try_reconnect = true;
        }
      }
    } catch(e) {} // if local storage isn't accessible
    if (try_reconnect) {
      this.reconnect_wrapper.classList.add('show');
    } else {
      this.connect_wrapper.classList.add('show');
    }
    this.nickname.addEventListener('keyup', (e) => {
      const valid_name = this.validateName();
      if (valid_name && e.key === 'Enter') {
        this.connect({nickname: this.nickname.value, try_reconnect: false});
      }
    });
    clickButton(this.connect_button, () => {
      if (!this.validateName()) {
        return;
      }
      this.connect({nickname: this.nickname.value, try_reconnect: false});
      return 'Connecting ...';
    }, {re_enable_button: false});
  }

  connect(connect_data: ConnectData) {
    this.dispatchEvent(new CustomEvent<ConnectData>('connect', {'detail': connect_data}))
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
