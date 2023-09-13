import {GameSettings, LobbyMessage, LobbyRoom, LobbyRoomFromServer, serverResponseToRoom} from "./data_models";
import {DwgLobby} from "./lobby";

/** Handles messages for the frontend lobby */
export function handleMessage(lobby: DwgLobby, message: LobbyMessage) {
  if (!lobby.socketActive()) {
    return;
  }
  if (message.kind !== 'ping-update') {
    console.log(message);
  }
  switch(message.kind) {
    case 'lobby-you-joined':
      const id = parseInt(message.data);
      if (id) {
        lobby.connection_metadata.client_id = id;
        lobby.setNickname(message.content);
        lobby.chatbox.addChat({
          message: `You (${lobby.connection_metadata.nickname}) joined lobby with client id ${id}`,
          sender: 'server',
        });
        lobby.lobby_users.addUser({client_id: id, nickname: lobby.connection_metadata.nickname, ping: 0});
      } else {
        lobby.socket.close(3001, 'you-joined-lobby message did not return properly formed client id');
        lobby.dispatchEvent(lobby.connection_lost);
      }
      break;
    case 'lobby-joined':
      const join_client_id = parseInt(message.data);
      if (join_client_id) {
        lobby.chatbox.addChat({
          message: `${message.content} joined lobby with client id ${join_client_id}`,
          sender: 'server',
        });
        lobby.lobby_users.addUser({client_id: join_client_id, nickname: message.content, ping: 0});
      }
      break;
    case 'lobby-left':
      const left_client_id = parseInt(message.data);
      if (left_client_id) {
        lobby.chatbox.addChat({
          message: `${message.content} (client id ${left_client_id}) left lobby`,
          sender: 'server',
        });
        const client = lobby.lobby_users.getUser(left_client_id);
        lobby.lobby_users.removeUser(left_client_id);
        lobby.lobby_rooms.userDisconnected(left_client_id);
        if (client && client.room_id === lobby.connection_metadata.room_id &&
          client.client_id === lobby.lobby_room.getHost()?.client_id) {
            lobby.leaveRoom();
          } else {
            lobby.lobby_room.leaveRoom(left_client_id, 'disconnected from');
          }
      }
      break;
    case 'lobby-chat':
      const chat_client_id = parseInt(message.sender.replace('client-', ''));
      if (chat_client_id) {
        lobby.chatbox.addChat({
          message: message.content,
          sender: lobby.lobby_users.getUser(chat_client_id)?.nickname ?? chat_client_id.toString(),
        });
      }
      break;
    case 'ping-update':
      const update_ping_id = parseInt(message.data);
      const ping = parseInt(message.content);
      if (update_ping_id && !isNaN(ping)) {
        if (lobby.connection_metadata.client_id === update_ping_id) {
          lobby.setPing(ping);
        }
        lobby.lobby_room.updatePing(update_ping_id, ping);
        lobby.lobby_users.updatePing(update_ping_id, ping);
      }
      break;
    case 'room-created':
      const setRoom = (room: LobbyRoom, host_id: number) => {
        lobby.lobby_rooms.addRoom(room);
        if (lobby.connection_metadata.client_id === host_id) {
          lobby.enterRoom(room, true);
          setTimeout(() => {
            // quick fix in case server responds instantly
            lobby.create_room_button.innerText = "Room Created";
          }, 1);
        }
      }
      try {
        const host_id = parseInt(message.sender.replace('client-', ''));
        const host = lobby.lobby_users.getUser(host_id);
        const server_room = JSON.parse(message.content) as LobbyRoomFromServer;
        const room = serverResponseToRoom(server_room);
        host.room_id = room.room_id;
        setRoom.bind(this, room, host_id)();
      } catch(e) {
        const new_room_id = parseInt(message.data);
        if (new_room_id) {
          // TODO: send get room request
        }
      }
      break;
    case 'room-closed':
      const closed_room_id = parseInt(message.data);
      if (closed_room_id) {
        if (lobby.connection_metadata.room_id === closed_room_id) {
          lobby.leaveRoom();
        }
        const room = lobby.lobby_rooms.getRoom(closed_room_id);
        lobby.lobby_rooms.removeRoom(closed_room_id);
        if (room) {
          for (const client_id of room.players.keys()) {
            lobby.lobby_users.leaveRoom(client_id);
          }
        }
      }
      break;
    case 'room-left':
      const room_leave_id = parseInt(message.sender.replace('room-', ''));
      const client_leave_id = parseInt(message.data.replace('client-', ''));
      if (room_leave_id && client_leave_id) {
        lobby.userLeftRoom(room_leave_id, client_leave_id, 'left');
      }
      break;
    case 'room-joined-player':
      const room_join_id = parseInt(message.sender.replace('room-', ''));
      const client_join_id = parseInt(message.data.replace('client-', ''));
      const joinee = lobby.lobby_users.getUser(client_join_id);
      const room = lobby.lobby_rooms.getRoom(room_join_id);
      if (room_join_id && client_join_id && joinee) {
        lobby.lobby_users.joinRoom(client_join_id, room_join_id);
        lobby.lobby_rooms.playerJoinsRoom(room_join_id, joinee);
        if (room_join_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.joinRoom(joinee, true);
        } else if (client_join_id === lobby.connection_metadata.client_id && room) {
          lobby.enterRoom(lobby.lobby_rooms.getRoom(room_join_id), false);
        }
      }
      break;
    case 'room-joined-viewer':
      const viewer_room_join_id = parseInt(message.sender.replace('room-', ''));
      const viewer_client_join_id = parseInt(message.data.replace('client-', ''));
      const viewer_joinee = lobby.lobby_users.getUser(client_join_id);
      const viewer_room = lobby.lobby_rooms.getRoom(room_join_id);
      if (viewer_room_join_id && viewer_client_join_id && viewer_joinee) {
        lobby.lobby_users.joinRoom(viewer_client_join_id, viewer_room_join_id);
        lobby.lobby_rooms.viewerJoinsRoom(viewer_room_join_id, viewer_joinee);
        if (viewer_room_join_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.joinRoom(viewer_joinee, false);
        } else if (viewer_client_join_id === lobby.connection_metadata.client_id && viewer_room) {
          lobby.enterRoom(lobby.lobby_rooms.getRoom(viewer_room_join_id), false);
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
        lobby.connection_metadata.room_id === room_chat_room_id &&
        lobby.lobby_room.hasPlayer(room_chat_client_id))
      {
        lobby.lobby_room.chatbox.addChat({
          message: message.content,
          sender: lobby.lobby_room.getClient(room_chat_client_id)?.nickname ?? room_chat_client_id.toString(),
        });
      }
      break;
    case 'room-renamed':
      const room_renamed_id = parseInt(message.sender.replace('room-', ''));
      const renamer_client_id = parseInt(message.data);
      if (room_renamed_id && renamer_client_id) {
        if (renamer_client_id === lobby.connection_metadata.client_id) {
          // TODO: loader for renaming room
        }
        if (lobby.lobby_room.room && lobby.lobby_room.room.room_id === room_renamed_id) {
          lobby.lobby_room.renameRoom(message.content, renamer_client_id);
        }
        lobby.lobby_rooms.renameRoom(room_renamed_id, message.content);
      }
      break;
    case 'room-kicked':
      const room_kick_id = parseInt(message.sender.replace('room-', ''));
      const client_kick_id = parseInt(message.data);
      if (room_kick_id && client_kick_id) {
        lobby.userLeftRoom(room_kick_id, client_kick_id, 'was kicked from');
      }
      break;
    case 'room-promoted':
      const room_promote_id = parseInt(message.sender.replace('room-', ''));
      const client_promote_id = parseInt(message.data);
      if (room_promote_id && client_promote_id) {
        if (room_promote_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.promoteUser(client_promote_id, client_promote_id === lobby.connection_metadata.client_id);
        }
        lobby.lobby_rooms.promoteUser(room_promote_id, client_promote_id);
      }
      break;
    case 'room-viewer-set':
      const room_viewer_id = parseInt(message.sender.replace('room-', ''));
      const client_viewer_id = parseInt(message.data);
      if (room_viewer_id && client_viewer_id) {
        if (room_viewer_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.playerToViewer(client_viewer_id);
        }
        lobby.lobby_rooms.playerToViewer(room_viewer_id, client_viewer_id);
      }
      break;
    case 'room-player-set':
      const room_player_id = parseInt(message.sender.replace('room-', ''));
      const client_player_id = parseInt(message.data);
      if (room_player_id && client_player_id) {
        if (room_player_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.viewerToPlayer(client_player_id);
        }
        lobby.lobby_rooms.viewerToPlayer(room_player_id, client_player_id);
      }
      break;
      break;
    case 'room-settings-updated':
      try {
        const room_id = parseInt(message.sender.replace('room-', ''));
        const new_settings = JSON.parse(message.data) as GameSettings;
        lobby.lobby_rooms.updateRoomSettings(room_id, new_settings);
        if (room_id === lobby.connection_metadata.room_id) {
          lobby.lobby_room.updateSettings(new_settings);
        }
      } catch(e) {}
      break;
    case 'room-join-failed':
    case 'room-leave-failed':
    case 'room-rename-failed':
    case 'room-kick-failed':
    case 'room-promote-failed':
    case 'room-set-player-failed':
    case 'room-set-viewer-failed':
    case 'room-settings-updated-failed':
      throw new Error(message.content);
    default:
      console.log("Unknown message type", message.kind, "from", message.sender);
      break;
  }
}