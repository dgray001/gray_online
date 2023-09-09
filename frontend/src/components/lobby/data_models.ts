
/** Interface describing messages sent from the server to the lobby */
export declare interface LobbyMessage {
  sender: string;
  kind: string;
  content: string;
  data: string;
}

/** Function to convert string data into a lobby message object */
export function createMessage(sender: string, kind: string, content?: string, data?: string): string {
  return JSON.stringify({sender, kind, content, data} as LobbyMessage);
}

/** Interface describing the connection between the frontend and the server */
export declare interface ConnectionMetadata {
  nickname: string;
  client_id?: number;
  room_id?: number;
}

/** Data describing a user in the lobby */
export declare interface LobbyUser {
  client_id: number;
  nickname: string;
  room_id?: number;
}

/** Data describing a user returned from the server */
export declare interface LobbyUserFromServer {
  client_id: string;
  nickname: string;
  room_id?: string;
}

/** Convert a server response to a TS lobby user object */
export function serverResponseToUser(server_response: LobbyUserFromServer): LobbyUser {
  return {
    client_id: parseInt(server_response.client_id),
    nickname: server_response.nickname,
    room_id: parseInt(server_response.room_id) ?? undefined,
  }
}

/** Object describing lobby room data */
export declare interface LobbyRoom {
  room_id: number;
  room_name: string;
  host: LobbyUser;
  users: Map<number, LobbyUser>;
}

/** Data describing a lobby room returned from the server */
export declare interface LobbyRoomFromServer {
  room_id: string;
  room_name: string;
  host: LobbyUserFromServer;
  users: LobbyUserFromServer[];
}

/** Convert a server response to a TS lobby room object */
export function serverResponseToRoom(server_response: LobbyRoomFromServer): LobbyRoom {
  return {
    room_id: parseInt(server_response.room_id),
    room_name: server_response.room_name,
    host: serverResponseToUser(server_response.host),
    users: new Map(server_response.users.map((server_user) => {
      const user = serverResponseToUser(server_user);
      return [user.client_id, user];
    })),
  }
}