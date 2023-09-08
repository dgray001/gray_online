import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {clickButton} from '../../scripts/util';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';

import {DwgLobbyUsers} from './lobby_users/lobby_users';
import {DwgLobbyRooms, LobbyRoom} from './lobby_rooms/lobby_rooms';
import {DwgLobbyRoom} from './lobby_room/lobby_room';
import html from './lobby.html';

import './lobby.scss';
import '../chatbox/chatbox';
import './lobby_users/lobby_users';
import './lobby_rooms/lobby_rooms';
import './lobby_room/lobby_room';

interface LobbyMessage {
  sender: string;
  kind: string;
  content: string;
  data: string;
}

interface ConnectionMetadata {
  nickname: string;
  client_id?: number;
  room_id?: number;
}

export class DwgLobby extends DwgElement {
  name_header: HTMLSpanElement;
  refresh_lobby_button: HTMLButtonElement;
  create_room_button: HTMLButtonElement;
  lobby_rooms: DwgLobbyRooms;
  chatbox: DwgChatbox;
  lobby_users: DwgLobbyUsers;
  lobby_room_wrapper: HTMLDivElement;
  lobby_room: DwgLobbyRoom;

  socket: WebSocket;
  connection_metadata: ConnectionMetadata = {nickname: "Anonymous"};
  connection_lost = new Event('connection_lost');

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_header');
    this.configureElement('refresh_lobby_button');
    this.configureElement('create_room_button');
    this.configureElement('lobby_rooms');
    this.configureElement('chatbox');
    this.configureElement('lobby_users');
    this.configureElement('lobby_room_wrapper');
    this.configureElement('lobby_room');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with the entire lobby');
    clickButton(this.refresh_lobby_button, async () => {
      await this.lobby_rooms.refreshRooms();
    });
    clickButton(this.create_room_button, async () => {
      const response = await apiPost(`lobby/rooms/create/${this.connection_metadata.client_id}`, '');
      if (response.success) {
        return 'Joining Room ...';
      } else {
        setTimeout(() => {
          this.create_room_button.disabled = false;
          this.create_room_button.innerText = 'Create Room';
        }, 1000);
      }
      return 'Error Creating Room';
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
          this.chatbox.addChat({...message, sender: this.connection_metadata.nickname});
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
    this.lobby_rooms.refreshRooms();
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
          this.lobby_users.addUser({client_id: id, nickname: this.connection_metadata.nickname});
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
          this.lobby_users.addUser({client_id: join_client_id, nickname: message.content});
        }
        break;
      case 'lobby-left':
        const left_client_id = parseInt(message.data);
        if (left_client_id) {
          this.chatbox.addChat({
            message: `${message.content} (client id ${left_client_id}) left lobby`,
            sender: 'server',
          });
          this.lobby_users.removeUser({client_id: left_client_id, nickname: message.content});
        }
        break;
      case 'lobby-chat':
        const chat_client_id = parseInt(message.sender.replace('client-', ''));
        if (chat_client_id) {
          this.chatbox.addChat({
            message: message.content,
            sender: this.lobby_users.getUser(chat_client_id)?.nickname ?? chat_client_id.toString(),
          });
        }
        break;
      case 'room-created':
        const new_room_id = parseInt(message.data);
        const host_id = parseInt(message.content);
        const host = this.lobby_users.getUser(host_id)
        if (new_room_id && host_id && host) {
          const room: LobbyRoom = {
            room_id: new_room_id,
            host,
            users: [],
          };
          this.lobby_rooms.addRoom(room);
          if (this.connection_metadata.client_id === host_id) {
            this.enterRoom(room);
            setTimeout(() => {
              // quick fix in case server responds instantly
              this.create_room_button.innerText = "Room Created";
            }, 1);
          }
        }
        break;
      case 'room-closed':
        const closed_room_id = parseInt(message.data);
        if (closed_room_id) {
          if (this.connection_metadata.room_id === closed_room_id) {
            this.leaveRoom();
          }
          const room = this.lobby_rooms.getRoom(closed_room_id);
          this.lobby_rooms.removeRoom(closed_room_id);
          for (const client_id of [room.host.client_id,...room.users.map(user => user.client_id)]) {
            this.lobby_users.leaveRoom(client_id);
          }
        }
        break;
      case 'room-leavee':
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

  private enterRoom(room: LobbyRoom) {
    if (this.connection_metadata.room_id) {
      // TODO: broadcast leaving room
    }
    this.connection_metadata.room_id = room.room_id;
    this.lobby_room.setRoom(room);
    this.lobby_room_wrapper.classList.add('show');
  }

  private leaveRoom() {
    this.connection_metadata.room_id = undefined;
    this.lobby_room_wrapper.classList.remove('show');
    // TODO: leave room
  }
}

customElements.define('dwg-lobby', DwgLobby);
