export declare interface ServerMessage {
    sender: string;
    kind: string;
    content: string;
    data: string;
}
export declare function createMessage(sender: string, kind: string, content?: string, data?: string): string;
export declare interface ConnectionMetadata {
    nickname: string;
    ping: number;
    client_id?: number;
    room_id?: number;
}
export declare interface LobbyUser {
    client_id: number;
    nickname: string;
    ping: number;
    room_id?: number;
}
export declare interface LobbyUserFromServer {
    client_id: string;
    nickname: string;
    ping: string;
    room_id?: string;
}
export declare function serverResponseToUser(server_response: LobbyUserFromServer): LobbyUser;
export declare enum GameType {
    UNSPECIFIED = 0,
    FIDDLESTICKS = 1,
    EUCHRE = 2,
    RISQ = 3
}
export declare interface GameSettings {
    game_type: GameType;
    max_players: number;
    max_viewers: number;
    game_specific_settings?: object;
}
export declare function defaultGameSettings(): GameSettings;
export declare interface GameSettingsFromServer {
    game_type: string;
    max_players: string;
    max_viewers: string;
    game_specific_settings: object;
}
export declare function serverResponseToGameSettings(server_response: GameSettingsFromServer): GameSettings;
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
export declare function serverResponseToRoom(server_response: LobbyRoomFromServer): LobbyRoom;
