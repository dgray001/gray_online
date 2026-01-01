/** Interface describing messages sent from the server to the lobby */
export declare interface ServerMessage {
  sender: string;
  kind: string;
  content: string;
  data: string;
}

/** Function to convert string data into a lobby message object */
export function createMessage(sender: string, kind: string, content?: string, data?: string): string {
  return JSON.stringify({ sender, kind, content, data } as ServerMessage);
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
  ping: number;
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
    room_id: parseInt(server_response.room_id ?? '') ?? undefined,
  };
}

/** Enum for possible games the lobby can launch */
export enum GameType {
  UNSPECIFIED = 0,
  FIDDLESTICKS = 1,
  EUCHRE = 2,
  RISQ = 3,
  TEST_GAME = 4,
}

/** Type of the lowercase strings of possible game types */
export type GameTypeLowerKeys = Lowercase<keyof typeof GameType>;

/** Strictly-typed const of lowercase GameType keys */
export const LOWERCASE_GAME_TYPE_KEYS = Object.keys(GameType)
  .filter((key) => isNaN(Number(key)))
  .map((key) => key.toLowerCase()) as Lowercase<keyof typeof GameType>[];

/** Function to check whether an input string is a proper game type (should make lowercase first) */
export function isValidGameTypeString(input: string): input is GameTypeLowerKeys {
  return LOWERCASE_GAME_TYPE_KEYS.includes(input.toLowerCase() as GameTypeLowerKeys);
}

/**
 * Converts a lowercase game string back to the numeric GameType enum.
 * Guaranteed to return a valid GameType if input is validated by isValidGameTypeString.
 */
export function getGameTypeFromLowercaseString(input: GameTypeLowerKeys): GameType {
  const upper_key = input.toUpperCase() as keyof typeof GameType;
  return GameType[upper_key];
}

/** Base properties shared by all games */
export declare interface BaseGameSettings {
  max_players: number;
  max_viewers: number;
}

/** A 'holder' interface to act as a registry that other folders can fill in */
export interface GameSettingsRegistry {}

/** Settings object for lobby room */
export type GameSettings = {
  [K in GameType]: BaseGameSettings &
    (K extends keyof GameSettingsRegistry
      ? {
          game_type: K;
          game_specific_settings: GameSettingsRegistry[K];
        }
      : {
          game_type: K;
          game_specific_settings?: never; // if game type doesn't have entry in registry
        });
}[GameType];

/** Returns default base game settings */
export function defaultBaseGameSettings(): BaseGameSettings {
  return {
    max_players: 8,
    max_viewers: 16,
  };
}

/** Returns default settings */
export function defaultGameSettings(): GameSettings {
  return {
    ...defaultBaseGameSettings(),
    game_type: GameType.UNSPECIFIED,
  };
}

/** Settings object for lobby room */
export declare interface GameSettingsFromServer {
  game_type: string;
  max_players: string;
  max_viewers: string;
  game_specific_settings: object;
}

/** Convert a server response to a TS game settings object */
export function serverResponseToGameSettings(server_response: GameSettingsFromServer): GameSettings {
  return {
    game_type: parseInt(server_response.game_type),
    max_players: parseInt(server_response.max_players),
    max_viewers: parseInt(server_response.max_viewers),
    // @ts-ignore: trust the backend here
    game_specific_settings: server_response.game_specific_settings,
  };
}

/** Object describing lobby room data */
export declare interface LobbyRoom {
  room_id: number;
  room_name: string;
  room_description: string;
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
  room_description: string;
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
    room_description: server_response.room_description,
    host: serverResponseToUser(server_response.host),
    players: new Map(
      server_response.players.map((server_user) => {
        const user = serverResponseToUser(server_user);
        return [user.client_id, user];
      })
    ),
    viewers: new Map(
      server_response.viewers.map((server_user) => {
        const user = serverResponseToUser(server_user);
        return [user.client_id, user];
      })
    ),
    game_settings: serverResponseToGameSettings(server_response.game_settings),
    game_id: server_response.game_id ? parseInt(server_response.game_id) : undefined,
  };
}
