import {DwgElement} from '../dwg_element';
import {apiPost} from '../../scripts/api';
import {clickButton} from '../../scripts/util';
import {ChatMessage, DwgChatbox} from '../chatbox/chatbox';

import {DwgLobbyUsers} from './lobby_users/lobby_users';
import {DwgLobbyRooms} from './lobby_rooms/lobby_rooms';
import {DwgLobbyRoom} from './lobby_room/lobby_room';
import {ConnectionMetadata, LobbyMessage, LobbyRoom, createMessage} from './data_models';
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
        this.connection_metadata.room_id.toString(),
      ));
    });
    this.lobby_room.addEventListener('rename_room', (e: CustomEvent<string>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-rename',
        e.detail,
        this.connection_metadata.room_id.toString(),
      ));
    });
    this.lobby_room.addEventListener('kick_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-kick',
        e.detail.toString(),
        this.connection_metadata.room_id.toString(),
      ));
    });
    this.lobby_room.addEventListener('promote_player', (e: CustomEvent<number>) => {
      this.socket.send(createMessage(
        `client-${this.connection_metadata.client_id}`,
        'room-promote',
        e.detail.toString(),
        this.connection_metadata.room_id.toString(),
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
    if (message.kind !== 'ping-update') {
      console.log(message);
    }
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
          this.lobby_users.addUser({client_id: id, nickname: this.connection_metadata.nickname, ping: 0});
        } else {
          this.socket.close(3001, 'you-joined-lobby message did not return properly formed client id');
          this.dispatchEvent(this.connection_lost);
        }
        break;
      case 'lobby-joined':
        const join_client_id = parseInt(message.data);
        if (join_client_id) {
          this.chatbox.addChat({
            message: `${message.content} joined lobby with client id ${join_client_id}`,
            sender: 'server',
          });
          this.lobby_users.addUser({client_id: join_client_id, nickname: message.content, ping: 0});
        }
        break;
      case 'lobby-left':
        const left_client_id = parseInt(message.data);
        if (left_client_id) {
          this.chatbox.addChat({
            message: `${message.content} (client id ${left_client_id}) left lobby`,
            sender: 'server',
          });
          const client = this.lobby_users.getUser(left_client_id);
          this.lobby_users.removeUser(left_client_id);
          this.lobby_rooms.userDisconnected(left_client_id);
          if (client && client.room_id === this.connection_metadata.room_id &&
            client.client_id === this.lobby_room.getHost()?.client_id) {
              this.leaveRoom();
            } else {
              this.lobby_room.leaveRoom(left_client_id, 'disconnected from');
            }
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
      case 'ping-update':
        const update_ping_id = parseInt(message.data);
        const ping = parseInt(message.content);
        if (update_ping_id && !isNaN(ping)) {
          if (this.connection_metadata.client_id === update_ping_id) {
            this.setPing(ping);
          }
          this.lobby_room.updatePing(update_ping_id, ping);
          this.lobby_users.updatePing(update_ping_id, ping);
        }
        break;
      case 'room-created':
        const new_room_id = parseInt(message.data);
        const host_id = parseInt(message.sender.replace('client-', ''));
        const host = this.lobby_users.getUser(host_id);
        if (new_room_id && host_id && host) {
          const room: LobbyRoom = {
            room_id: new_room_id,
            room_name: `${host.nickname}'s room`,
            host,
            users: new Map(),
          };
          room.users.set(host.client_id, host);
          this.lobby_rooms.addRoom(room);
          if (this.connection_metadata.client_id === host_id) {
            this.enterRoom(room, true);
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
          if (room) {
            for (const client_id of room.users.keys()) {
              this.lobby_users.leaveRoom(client_id);
            }
          }
        }
        break;
      case 'room-left':
        const room_leave_id = parseInt(message.sender.replace('room-', ''));
        const client_leave_id = parseInt(message.data.replace('client-', ''));
        if (room_leave_id && client_leave_id) {
          this.userLeftRoom(room_leave_id, client_leave_id, 'left');
        }
        break;
      case 'room-joined':
        const room_join_id = parseInt(message.sender.replace('room-', ''));
        const client_join_id = parseInt(message.data.replace('client-', ''));
        const joinee = this.lobby_users.getUser(client_join_id);
        const room = this.lobby_rooms.getRoom(room_join_id);
        if (room_join_id && client_join_id && joinee) {
          this.lobby_users.joinRoom(client_join_id, room_join_id);
          this.lobby_rooms.clientJoinsRoom(room_join_id, joinee);
          if (room_join_id === this.connection_metadata.room_id) {
            this.lobby_room.joinRoom(joinee);
          } else if (client_join_id === this.connection_metadata.client_id && room) {
            this.enterRoom(this.lobby_rooms.getRoom(room_join_id), false);
          }
        }
        break;
      case 'room-chat':
        const room_chat_sender_split = message.sender.split('-');
        if (room_chat_sender_split.length < 3) {
          break;
        }
        const room_chat_room_id = parseInt(room_chat_sender_split[1]);
        const room_chat_client_id = parseInt(room_chat_sender_split[2]);
        if (room_chat_room_id && room_chat_client_id &&
          this.connection_metadata.room_id === room_chat_room_id &&
          this.lobby_room.hasClient(room_chat_client_id))
        {
          this.lobby_room.chatbox.addChat({
            message: message.content,
            sender: this.lobby_room.getClient(room_chat_client_id)?.nickname ?? room_chat_client_id.toString(),
          });
        }
        break;
      case 'room-renamed':
        const room_renamed_id = parseInt(message.sender.replace('room-', ''));
        const renamer_client_id = parseInt(message.data);
        if (room_renamed_id && renamer_client_id) {
          if (renamer_client_id === this.connection_metadata.client_id) {
            // TODO: loader for renaming room
          }
          if (this.lobby_room.room && this.lobby_room.room.room_id === room_renamed_id) {
            this.lobby_room.renameRoom(message.content, renamer_client_id);
          }
          this.lobby_rooms.renameRoom(room_renamed_id, message.content);
        }
        break;
      case 'room-kicked':
        const room_kick_id = parseInt(message.sender.replace('room-', ''));
        const client_kick_id = parseInt(message.data);
        if (room_kick_id && client_kick_id) {
          this.userLeftRoom(room_kick_id, client_kick_id, 'was kicked from');
        }
        break;
      case 'room-promoted':
        const room_promote_id = parseInt(message.sender.replace('room-', ''));
        const client_promote_id = parseInt(message.data);
        if (room_promote_id && client_promote_id) {
          if (room_promote_id === this.connection_metadata.room_id) {
            this.lobby_room.promoteUser(client_promote_id, client_promote_id === this.connection_metadata.client_id);
          }
          this.lobby_rooms.promoteUser(room_promote_id, client_promote_id);
        }
        break;
      case 'room-join-failed':
      case 'room-leave-failed':
      case 'room-rename-failed':
      case 'room-kick-failed':
      case 'room-promote-failed':
        throw new Error(message.content);
      default:
        console.log("Unknown message type", message.kind, "from", message.sender);
        break;
    }
  }

  private userLeftRoom(room_id: number, client_id: number, left_text: string) {
    if (room_id === this.connection_metadata.room_id) {
      this.lobby_room.leaveRoom(client_id, left_text);
      if (client_id === this.connection_metadata.client_id) {
        this.leaveRoom();
      }
    }
    this.lobby_users.leaveRoom(client_id);
    this.lobby_rooms.clientLeavesRoom(room_id, client_id);
  }

  private socketActive() {
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

  private enterRoom(room: LobbyRoom, is_host: boolean) {
    this.connection_metadata.room_id = room.room_id;
    this.lobby_room.setRoom(room, is_host);
    this.lobby_room_wrapper.classList.add('show');
  }

  private leaveRoom() {
    this.connection_metadata.room_id = undefined;
    this.lobby_room_wrapper.classList.remove('show');
    this.lobby_room.clearRoom();
    this.create_room_button.disabled = false;
    this.create_room_button.innerText = 'Create Room';
  }

  private sendChatMessage(chatbox: DwgChatbox, sender_type: string, message_kind: string, message: ChatMessage) {
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
