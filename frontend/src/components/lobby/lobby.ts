import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {LobbyRoom} from '../lobby_room/lobby_room';
import {createLobbyRoomSelector} from '../lobby_room_selector/lobby_room_selector';
import {clickButton, untilTimer} from '../../scripts/util';
import {DwgChatbox} from '../chatbox/chatbox';

import html from './lobby.html';

import './lobby.scss';
import '../chatbox/chatbox';

interface LobbyMessage {
  sender: string;
  kind: string;
  content: string;
  data: string;
}

interface ConnectionMetadata {
  client_id?: number;
  nickname: string;
}

export class DwgLobby extends DwgElement {
  name_header: HTMLSpanElement;
  refresh_lobby_button: HTMLButtonElement;
  create_room_button: HTMLButtonElement;
  room_container: HTMLDivElement;
  chatbox: DwgChatbox;

  socket: WebSocket;
  connection_metadata: ConnectionMetadata = {nickname: "Anonymous"};
  connection_lost = new Event('connection_lost');

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_header');
    this.configureElement('refresh_lobby_button');
    this.configureElement('create_room_button');
    this.configureElement('room_container');
    this.configureElement('chatbox');
  }

  protected override parsedCallback(): void {
    clickButton(this.refresh_lobby_button, async () => {
      await this.refreshLobby();
    });
    clickButton(this.create_room_button, async () => {
      const response = await apiPost(`lobby/rooms/create/${this.connection_metadata.client_id}`, '');
      return 'Room Created';
    }, {loading_text: 'Creating Room ...', re_enable_button: false});
    this.refreshLobby();
  }

  setSocket(new_socket: WebSocket) {
    if (!!this.socket) {
      this.socket.close(3000, "opening new connection");
    }
    this.socket = new_socket;
    this.socket.addEventListener('message', (m) => {
      try {
        const message = JSON.parse(m.data) as LobbyMessage;
        this.handleMessage(message);
      } catch(e) {
        console.log("Error parsing message: ", m, e)
      }
    });
  }

  private async refreshLobby() {
    this.room_container.innerHTML = ' ... loading';
    await untilTimer(500);
    const response = await apiPost<LobbyRoom[]>('lobby/rooms/get', '');
    if (response.success) {
      let html = '';
      for (const room of response.result) {
        html += createLobbyRoomSelector(room);
      }
      this.room_container.innerHTML = html;
    } else {
      this.room_container.innerHTML = `Error loading rooms: ${response.error_message}`;
    }
  }

  private handleMessage(message: LobbyMessage) {
    if (!this.socketActive()) {
      return;
    }
    console.log(message);
    switch(message.kind) {
      case 'you-joined-lobby':
        const id = parseInt(message.data);
        if (id) {
          this.connection_metadata.client_id = id;
          this.setNickname(message.content);
          this.chatbox.addChat({message: `<div class="grayed">You (${this.connection_metadata.nickname}) joined lobby with client id ${id}</div>`});
          this.socket.send(this.createMessage(`client-${id}`, 'joined-lobby', this.connection_metadata.nickname, `${id}`));
        } else {
          this.socket.close(3001, 'you-joined-lobby message did not return properly formed client id');
          this.dispatchEvent(this.connection_lost);
        }
        break;
      case 'joined-lobby':
        const join_user_id = parseInt(message.data);
        if (join_user_id) {
          this.chatbox.addChat({message: `<div class="grayed">${message.content} joined lobby with client id ${join_user_id}</div>`});
        }
        break;
      case 'left-lobby':
        const left_user_id = parseInt(message.data);
        if (left_user_id) {
          this.chatbox.addChat({message: `<div class="grayed">${message.content} (client id ${left_user_id}) left lobby</div>`});
        }
        break;
      default:
        console.log("Unknown message type", message.kind, "from", message.sender);
        break;
    }
  }

  private socketActive() {
    return !!this.socket && this.socket.readyState == WebSocket.OPEN;
  }

  private createMessage(sender: string, kind: string, content?: string, data?: string): string {
    return JSON.stringify({sender, kind, content, data} as LobbyMessage);
  }

  private setNickname(nickname: string) {
    this.connection_metadata.nickname = nickname;
    this.name_header.innerText = nickname;
  }
}

customElements.define('dwg-lobby', DwgLobby);
