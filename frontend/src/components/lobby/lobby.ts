import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {LobbyRoom} from '../lobby_room/lobby_room';
import {createLobbyRoomSelector} from '../lobby_room_selector/lobby_room_selector';
import {clickButton, untilTimer} from '../../scripts/util';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';

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
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent) => {
      let message_sent = false;
      try {
        const message: ChatMessage = e.detail;
        if (this.socketActive()) {
          this.socket.send(this.createMessage(
            message.sender ?? `client-${this.connection_metadata.client_id}`,
            'lobby-chat',
            message.message,
            message.color,
          ));
          message_sent = true;
        }
        if (message_sent) {
          this.chatbox.addChat({...message, sender: this.connection_metadata.client_id.toString()});
        } else {
          this.chatbox.addChat({
            message: `Message unable to be sent: "${message.message}"`,
            color: 'gray',
          });
        }
      } catch(e) {
        this.chatbox.addChat({
          message: 'Error trying to send message',
          color: 'gray',
        });
      }
    });
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
      case 'lobby-you-joined':
        const id = parseInt(message.data);
        if (id) {
          this.connection_metadata.client_id = id;
          this.setNickname(message.content);
          this.chatbox.addChat({
            message: `You (${this.connection_metadata.nickname}) joined lobby with client id ${id}`,
            sender: 'server',
          });
          this.socket.send(this.createMessage(`client-${id}`, 'lobby-join', this.connection_metadata.nickname, `${id}`));
        } else {
          this.socket.close(3001, 'you-joined-lobby message did not return properly formed client id');
          this.dispatchEvent(this.connection_lost);
        }
        break;
      case 'lobby-join':
        const join_client_id = parseInt(message.data);
        if (join_client_id) {
          this.chatbox.addChat({
            message: `${message.content} joined lobby with client id ${join_client_id}`,
            sender: 'server',
          });
        }
        break;
      case 'lobby-left':
        const left_client_id = parseInt(message.data);
        if (left_client_id) {
          this.chatbox.addChat({
            message: `${message.content} (client id ${left_client_id}) left lobby`,
            sender: 'server',
          });
        }
        break;
      case 'lobby-chat':
        const chat_client_id = parseInt(message.sender.replace('client-', ''));
        if (chat_client_id) {
          this.chatbox.addChat({
            message: message.content,
            sender: chat_client_id.toString(),
          });
        }
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
