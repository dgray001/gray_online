import {DwgElement} from '../dwg_element';
import {clickButton} from '../../scripts/util';
import {ChatMessage, DwgChatbox, SERVER_CHAT_NAME} from '../chatbox/chatbox';
import {getUrlParam, removeUrlParam, setUrlParam} from '../../scripts/url';

import {DwgLobbyUsers} from './lobby_users/lobby_users';
import {DwgLobbyRooms, JoinRoomData} from './lobby_rooms/lobby_rooms';
import {DwgLobbyRoom} from './lobby_room/lobby_room';
import {ConnectionMetadata, ServerMessage, LobbyRoom, createMessage} from './data_models';
import {handleMessage} from './message_handler';
import html from './lobby.html';

import './lobby.scss';
import '../chatbox/chatbox';
import './lobby_users/lobby_users';
import './lobby_rooms/lobby_rooms';
import './lobby_room/lobby_room';

const LOBBY_PING_TIME = 5000; // time between lobby refreshes

const URL_PARAM_ROOM = 'room_id';

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
  lobby_users_button: HTMLButtonElement;
  lobby_users_backdrop: HTMLDivElement;

  socket: WebSocket;
  connection_metadata: ConnectionMetadata = {nickname: "Anonymous", ping: 0};

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
    this.configureElement('lobby_users_button');
    this.configureElement('lobby_users_backdrop');
  }

  protected override parsedCallback(): void {
    this.chatbox.setPlaceholder('Chat with entire lobby');
    clickButton(this.refresh_lobby_button, async () => {
      await this.refreshLobbyRooms();
    });
    clickButton(this.create_room_button, async () => {
      this.socket.send(createMessage(`client-${this.connection_metadata.client_id}`, 'room-create'))
    }, {loading_text: 'Creating Room ...', re_enable_button: false});
    this.lobby_rooms.addEventListener('join_room', async (e: CustomEvent<JoinRoomData>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        e.detail.join_as_player ? 'room-join-player' : 'room-join-viewer',
        '',
        e.detail.room_id.toString(),
      ));
    });
    this.chatbox.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      this.sendChatMessage(this.chatbox, 'lobby-chat', {...e.detail,
        sender: `client-${this.connection_metadata.client_id}`}, this.connection_metadata.nickname);
    });
    this.lobby_room.addEventListener('chat_sent', (e: CustomEvent<ChatMessage>) => {
      const message = e.detail;
      if (message.message.startsWith('\\l')) {
        message.message = message.message.slice(2).trim();
        this.sendChatMessage(this.chatbox, 'lobby-chat', {...e.detail,
          sender: `client-${this.connection_metadata.client_id}`}, this.connection_metadata.nickname);
      } else if (message.message.startsWith('\\u')) {
        message.message = message.message.slice(2).trim(); // TODO: implement
      } else if (message.message.startsWith('\\g')) {
        message.message = message.message.slice(2).trim();
        const room = this.lobby_room.getRoom();
        if (!!room.game_id) {
          this.socket.send(createMessage(
            message.sender ?? `game-${this.connection_metadata.client_id}`,
            'game-chat',
            message.message,
            message.color,
          ));
        } else {
          this.lobby_room.chatbox.addChat({
            message: 'Cannot send message to unlaunched game',
            color: 'gray',
          }, true);
        }
      } else {
        if (message.message.startsWith('\\r')) {
          message.message = message.message.slice(2).trim();
        }
        const server_message = message.sender === SERVER_CHAT_NAME;
        const display_sender = server_message ? message.sender : this.connection_metadata.nickname;
        const sender = `room-${this.connection_metadata.room_id}-${this.connection_metadata.client_id}`;
        message.sender = server_message ? `${sender}-${message.sender}` : sender;
        this.sendChatMessage(this.lobby_room.chatbox, 'room-chat', message, display_sender);
      }
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
    this.lobby_room.addEventListener('rejoin_game', () => {
      this.dispatchEvent(new CustomEvent('rejoin_game', {'detail': this.lobby_room.room}));
    });
    this.lobby_users_button.addEventListener('click', () => {
      this.lobby_users.classList.toggle('show');
      this.lobby_users_backdrop.classList.toggle('show');
    });
    this.lobby_users_backdrop.addEventListener('click', () => {
      this.lobby_users.classList.remove('show');
      this.lobby_users_backdrop.classList.remove('show');
    });
    this.refreshLobbyRooms();
    this.lobby_users.refreshUsers();
    setInterval(() => {
      this.pingServer();
      this.ping_container.innerText = `ping: ${Math.round(this.connection_metadata.ping)}`;
    }, LOBBY_PING_TIME);
  }

  setSocket(new_socket: WebSocket) {
    if (!!this.socket) {
      this.socket.close(3000, "opening new connection");
    }
    this.waitingOnConnectedTimes = DwgLobby.DEFAULT_CONNECTION_TIMES;
    this.socket = new_socket;
    this.socket.addEventListener('message', (m) => {
      try {
        const message = JSON.parse(m.data) as ServerMessage;
        handleMessage(this, message);
      } catch(e) {
        console.error("Error parsing message: ", m, e)
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
    this.waitingOnConnectedTimes = DwgLobby.DEFAULT_CONNECTION_TIMES;
    this.connection_metadata.ping = ping;
  }

  async refreshLobbyRooms(check_url = false) {
    const current_room = await this.lobby_rooms.refreshRooms(this.connection_metadata.client_id ?? -1);
    if (check_url) {
      const url_room_id = parseInt(getUrlParam(URL_PARAM_ROOM));
      const room = this.lobby_rooms.getRoom(url_room_id);
      if (!!room && current_room?.room_id !== room.room_id) {
        this.socket.send(createMessage(
          `client-${this.connection_metadata.client_id}`,
          'room-join',
          '',
          url_room_id.toString(),
        ));
        return;
      } else {
        removeUrlParam(URL_PARAM_ROOM);
      }
    }
    if (!!current_room && current_room.room_id !== this.lobby_room.room?.room_id) {
      this.enterRoom(current_room, current_room.host.client_id === (this.connection_metadata.client_id ?? -1));
    } else if (!current_room) {
      this.leaveRoom();
    }
  }

  enterRoom(room: LobbyRoom, is_host: boolean) {
    this.connection_metadata.room_id = room.room_id;
    this.lobby_room.setRoom(room, is_host, this.connection_metadata.client_id);
    this.lobby_room_wrapper.classList.add('show');
    setUrlParam(URL_PARAM_ROOM, room.room_id.toString());
    if (!room.game_id) {
      this.can_auto_launch_room = true; // only auto-launch if game not already launched
    }
  }

  leaveRoom() {
    this.connection_metadata.room_id = undefined;
    this.lobby_room_wrapper.classList.remove('show');
    this.create_room_button.disabled = false;
    this.create_room_button.innerText = 'Create Room';
    if (this.lobby_room.inRoom()) {
      this.lobby_room.clearRoom();
      removeUrlParam(URL_PARAM_ROOM);
    }
  }

  sendChatMessage(chatbox: DwgChatbox, message_kind: string, message: ChatMessage, display_sender: string) {
    let message_sent = false;
    message.message = message.message.trim();
    try {
      if (this.socketActive()) {
        this.socket.send(createMessage(
          message.sender,
          message_kind,
          message.message,
          message.color,
        ));
        message_sent = true;
      }
      if (message_sent) {
        chatbox.addChat({...message, sender: display_sender}, true);
      } else {
        chatbox.addChat({
          message: `Message unable to be sent: "${message.message}"`,
          color: 'gray',
        }, true);
      }
    } catch(e) {
      chatbox.addChat({
        message: 'Error trying to send message',
        color: 'gray',
      }, true);
    }
  }

  private can_auto_launch_room = false;
  canAutoLaunchRoom(): boolean {
    return this.can_auto_launch_room;
  }

  returnFromGame() {
    this.can_auto_launch_room = false;
  }

  static DEFAULT_CONNECTION_TIMES = 3;
  waitingOnConnectedTimes = 0;
  pingServer() {
    if (this.classList.contains('connector-open') || this.classList.contains('hide')) {
      return;
    }
    if (!this.socketActive()) {
      this.dispatchEvent(new Event('connected_lost'));
      return;
    }
    if (!this.classList.contains('connected')) {
      this.dispatchEvent(new Event('connection_lost'));
      return;
    }
    this.waitingOnConnectedTimes--;
    if (this.waitingOnConnectedTimes < 1) {
      this.dispatchEvent(new Event('connection_lost'));
      return;
    }
    this.lobby_users.refreshUsers();
    this.refreshLobbyRooms();
    if (this.lobby_room_wrapper.classList.contains('show')) {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-refresh',
        '',
        (this.lobby_room.room?.room_id ?? 0).toString(),
      ));
    }
  }
}

customElements.define('dwg-lobby', DwgLobby);
