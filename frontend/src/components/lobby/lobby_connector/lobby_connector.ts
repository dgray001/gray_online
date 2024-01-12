import {DwgElement} from '../../dwg_element';
import {clickButton} from '../../../scripts/util';
import {SERVER_CHAT_NAME} from '../../chatbox/chatbox';
import {ConnectionMetadata} from '../data_models';

import html from './lobby_connector.html';
import './lobby_connector.scss';
import { apiGet } from '../../../scripts/api';

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

  reconnect_data: ConnectData = {nickname: '', try_reconnect: true, client_id: 0};

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

  protected override async parsedCallback(): Promise<void> {
    this.connect_button.disabled = true;
    let try_reconnect = false;
    try {
      const client_id_time = parseInt(localStorage.getItem("client_id_time"));
      const client_id = parseInt(localStorage.getItem("client_id"));
      const previous_nickname = localStorage.getItem("client_nickname");
      if (!!previous_nickname) {
        const previous_name_valid = this.validateName(previous_nickname);
        if (previous_name_valid && !!client_id && !!client_id_time &&
          (Date.now() - client_id_time < 1000 * 60 * 60 * 24))
        {
          this.previous_nickname.innerText = previous_nickname;
          this.reconnect_data = {nickname: previous_nickname, try_reconnect: true, client_id};
          try_reconnect = true;
        }
      }
    } catch(e) {} // if local storage isn't accessible
    if (try_reconnect) {
      this.reconnect_wrapper.classList.add('show');
    } else {
      this.connect_wrapper.classList.add('show');
    }
    const pong = await apiGet<string>('ping');
    if (!pong.success) {
      this.status_message.innerHTML = 'Cannot connect to server at this time; you can try refreshing the page';
      return;
    }
    this.nickname.addEventListener('keyup', (e) => {
      const valid_name = this.validateInputName();
      if (valid_name && e.key === 'Enter') {
        this.connect({nickname: this.nickname.value, try_reconnect: false});
      }
    });
    clickButton(this.connect_button, () => {
      if (!this.validateInputName()) {
        return;
      }
      this.connect({nickname: this.nickname.value, try_reconnect: false});
      return 'Connecting ...';
    }, {re_enable_button: false});
    clickButton(this.reconnect_button, () => {
      this.connect(this.reconnect_data);
      return 'Reconnecting ...';
    }, {re_enable_button: false});
    clickButton(this.new_connection_button, () => {
      this.reconnect_wrapper.classList.remove('show');
      this.connect_wrapper.classList.add('show');
    });
  }

  tryReconnecting(message: string, connection_metadata: ConnectionMetadata) {
    this.status_message.innerText = message;
    this.reconnect_button.disabled = false;
    this.reconnect_button.innerText = "Reconnect to Lobby";
    this.connect_button.disabled = false;
    this.connect_button.innerText = "Reconnect to Lobby";
    this.reconnect_data.nickname = connection_metadata.nickname;
    this.reconnect_data.client_id = connection_metadata.client_id;
    if (this.validateName(this.reconnect_data.nickname) && !!this.reconnect_data.client_id) {
      this.reconnect_wrapper.classList.add('show');
      this.connect_wrapper.classList.remove('show');
    } else {
      this.reconnect_wrapper.classList.remove('show');
      this.connect_wrapper.classList.add('show');
    }
    this.classList.remove('hide');
  }

  connect(connect_data: ConnectData) {
    this.dispatchEvent(new CustomEvent<ConnectData>('connect', {'detail': connect_data}))
  }

  invalid_names = [SERVER_CHAT_NAME];

  private validateInputName(): boolean {
    const valid = this.validateName(this.nickname.value);
    this.connect_button.disabled = !valid;
    return valid;
  }

  private validateName(name: string): boolean {
    if (this.invalid_names.includes(name)) {
      return false;
    }
    return name.length > 1 && name.length < 17;
  }
}

customElements.define('dwg-lobby-connector', DwgLobbyConnector);
