import {DwgElement} from '../dwg_element';
import {clickButton} from '../../scripts/util';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';

import {DwgLobbyUsers} from './lobby_users/lobby_users';
import {DwgLobbyRooms} from './lobby_rooms/lobby_rooms';
import {DwgLobbyRoom} from './lobby_room/lobby_room';
import {ConnectionMetadata, ServerMessage, LobbyRoom, createMessage} from './data_models';
import {handleMessage} from './message_handler';
import html from './lobby.html';

import './lobby.scss';
import '../chatbox/chatbox';
import './lobby_users/lobby_users';
import './lobby_rooms/lobby_rooms';
import './lobby_room/lobby_room';

export class DwgLobby extends DwgElement {
  name_container: HTMLSpanElement;
  ping_container: HTMLSpanElement;
  refresh_lobby_button: HTMLButtonElement;
  create_room_button: HTMLButtonElement;
  lobby_rooms: DwgLobbyRooms;
  chatbox: DwgChatbox;
  lobby_users: DwgLobbyUsers;
  lobby_room_wrapper: HTMLDivElement;
  lobby_room: DwgLobbyRoom;

  socket: WebSocket;
  connection_metadata: ConnectionMetadata = {nickname: "Anonymous", ping: 0};
  connection_lost = new Event('connection_lost');

  constructor() {
    super();
    this.htmlString = html;
    this.configureElement('name_container');
    this.configureElement('ping_container');
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
      this.socket.send(createMessage(`client-${this.connection_metadata.client_id}`, 'room-create'))
    }, {loading_text: 'Creating Room ...', re_enable_button: false});
    this.lobby_rooms.addEventListener('join_room', async (e: CustomEvent<number>) => {
      this.socket.send(createMessage(`client-${this.connection_metadata.client_id}`, 'room-join', '', e.detail.toString()))
    });
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      this.sendChatMessage(this.chatbox, 'client', 'lobby-chat', e.detail);
    });
    this.lobby_room.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      const message = e.detail;
      message.sender = `room-${this.connection_metadata.room_id}-${this.connection_metadata.client_id}`;
      this.sendChatMessage(this.lobby_room.chatbox, 'room', 'room-chat', message);
    });
    this.lobby_room.addEventListener('leave_room', async () => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-leave',
        '',
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('rename_room', (e: CustomEvent<string>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-rename',
        e.detail,
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('kick_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-kick',
        e.detail.toString(),
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('viewer_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-set-viewer',
        e.detail.toString(),
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('player_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-set-player',
        e.detail.toString(),
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('promote_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-promote',
        e.detail.toString(),
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_room.addEventListener('save_settings', (e: CustomEvent<string>) => {
      this.socket.send(e.detail);
    });
    this.lobby_room.addEventListener('launch_game', () => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-launch',
        '',
        this.connection_metadata.room_id?.toString(),
      ));
    });
    this.lobby_rooms.refreshRooms();
  }

  setSocket(new_socket: WebSocket) {
    if (!!this.socket) {
      this.socket.close(3000, "opening new connection");
    }
    this.socket = new_socket;
    this.lobby_rooms.refreshRooms();
    this.lobby_users.refreshUsers();
    this.socket.addEventListener('message', (m) => {
      try {
        const message = JSON.parse(m.data) as ServerMessage;
        handleMessage(this, message);
      } catch(e) {
        console.log("Error parsing message: ", m, e)
      }
    });
  }

  userLeftRoom(room_id: number, client_id: number, left_text: string) {
    if (room_id === this.connection_metadata.room_id) {
      this.lobby_room.leaveRoom(client_id, left_text);
      if (client_id === this.connection_metadata.client_id) {
        this.leaveRoom();
      }
    }
    this.lobby_users.leaveRoom(client_id);
    this.lobby_rooms.clientLeavesRoom(room_id, client_id);
  }

  socketActive() {
    return !!this.socket && this.socket.readyState == WebSocket.OPEN;
  }

  setNickname(nickname: string) {
    this.connection_metadata.nickname = nickname;
    this.name_container.innerText = nickname;
  }

  setPing(ping: number) {
    this.connection_metadata.ping = ping;
    this.ping_container.innerText = `ping: ${Math.round(ping)}`;
  }

  enterRoom(room: LobbyRoom, is_host: boolean) {
    this.connection_metadata.room_id = room.room_id;
    this.lobby_room.setRoom(room, is_host);
    this.lobby_room_wrapper.classList.add('show');
  }

  leaveRoom() {
    this.connection_metadata.room_id = undefined;
    this.lobby_room_wrapper.classList.remove('show');
    this.lobby_room.clearRoom();
    this.create_room_button.disabled = false;
    this.create_room_button.innerText = 'Create Room';
  }

  sendChatMessage(chatbox: DwgChatbox, sender_type: string, message_kind: string, message: ChatMessage) {
    let message_sent = false;
    try {
      if (this.socketActive()) {
        this.socket.send(createMessage(
          message.sender ?? `${sender_type}-${this.connection_metadata.client_id}`,
          message_kind,
          message.message,
          message.color,
        ));
        message_sent = true;
      }
      if (message_sent) {
        chatbox.addChat({...message, sender: this.connection_metadata.nickname});
      } else {
        chatbox.addChat({
          message: `Message unable to be sent: "${message.message}"`,
          color: 'gray',
        });
      }
    } catch(e) {
      chatbox.addChat({
        message: 'Error trying to send message',
        color: 'gray',
      });
    }
  }
}

customElements.define('dwg-lobby', DwgLobby);
