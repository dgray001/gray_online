
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
  ping: number;
  client_id?: number;
  room_id?: number;
}

/** Data describing a user in the lobby */
export declare interface LobbyUser {
  client_id: number;
  nickname: string;
  ping: number,
  room_id?: number;
}

/** Data describing a user returned from the server */
export declare interface LobbyUserFromServer {
  client_id: string;
  nickname: string;
  ping: string;
  room_id?: string;
}

/** Convert a server response to a TS lobby user object */
export function serverResponseToUser(server_response: LobbyUserFromServer): LobbyUser {
  return {
    client_id: parseInt(server_response.client_id),
    nickname: server_response.nickname,
    ping: parseInt(server_response.ping),
    room_id: parseInt(server_response.room_id) ?? undefined,
  }
}

/** Enum for possible games the lobby can launch */
export enum GameType {
  UNSPECIFIED = 0,
  FIDDLESTICKS = 1,
}

/** Settings object for lobby room */
export declare interface GameSettings {
  game_type: GameType;
  max_players: number;
  max_viewers: number;
}

/** Returns default settings */
export function defaultGameSettings(): GameSettings {
  return {
    game_type: GameType.UNSPECIFIED,
    max_players: 8,
    max_viewers: 16,
  }
}

/** Settings object for lobby room */
export declare interface GameSettingsFromServer {
  game_type: string;
  max_players: string;
  max_viewers: string;
}

/** Convert a server response to a TS game settings object */
export function serverResponseToGameSettings(server_response: GameSettingsFromServer): GameSettings {
  return {
    game_type: parseInt(server_response.game_type),
    max_players: parseInt(server_response.max_players),
    max_viewers: parseInt(server_response.max_viewers),
  }
}

/** Object describing lobby room data */
export declare interface LobbyRoom {
  room_id: number;
  room_name: string;
  host: LobbyUser;
  players: Map<number, LobbyUser>;
  viewers: Map<number, LobbyUser>;
  game_settings: GameSettings;
  game_id?: number;
}

/** Data describing a lobby room returned from the server */
export declare interface LobbyRoomFromServer {
  room_id: string;
  room_name: string;
  host: LobbyUserFromServer;
  players: LobbyUserFromServer[];
  viewers: LobbyUserFromServer[];
  game_settings: GameSettingsFromServer;
  game_id?: string;
}

/** Convert a server response to a TS lobby room object */
export function serverResponseToRoom(server_response: LobbyRoomFromServer): LobbyRoom {
  return {
    room_id: parseInt(server_response.room_id),
    room_name: server_response.room_name,
    host: serverResponseToUser(server_response.host),
    players: new Map(server_response.players.map((server_user) => {
      const user = serverResponseToUser(server_user);
      return [user.client_id, user];
    })),
    viewers: new Map(server_response.viewers.map((server_user) => {
      const user = serverResponseToUser(server_user);
      return [user.client_id, user];
    })),
    game_settings: serverResponseToGameSettings(server_response.game_settings),
    game_id: server_response.game_id ? parseInt(server_response.game_id) : undefined,
  }
}