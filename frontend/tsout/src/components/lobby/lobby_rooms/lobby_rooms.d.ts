import { DwgElement } from '../../dwg_element';
import { GameSettings, LobbyRoom, LobbyUser } from '../data_models';
import { DwgRoomSelector } from './room_selector/room_selector';
import './lobby_rooms.scss';
import './room_selector/room_selector';
export declare interface JoinRoomData {
    room_id: number;
    join_as_player: boolean;
}
interface RoomData {
    data: LobbyRoom;
    el: DwgRoomSelector;
    refreshed: boolean;
}
export declare class DwgLobbyRooms extends DwgElement {
    private loading_message;
    private room_container;
    lock: (fn: () => Promise<unknown>) => Promise<unknown>;
    refreshing_rooms: boolean;
    rooms: Map<number, RoomData>;
    constructor();
    protected parsedCallback(): void;
    private first_load;
    refreshRooms(client_id: number, show_load_message?: boolean): Promise<LobbyRoom>;
    addRoom(room: LobbyRoom): void;
    removeRoom(room_id: number): void;
    userDisconnected(client_id: number): void;
    getRoom(room_id: number): RoomData;
    roomUpdated(room: RoomData): void;
    renameRoom(room_id: number, new_name: string): void;
    updateRoomDescription(room_id: number, new_description: string): void;
    promoteUser(room_id: number, user_id: number): void;
    playerToViewer(room_id: number, user_id: number): void;
    viewerToPlayer(room_id: number, user_id: number): void;
    playerJoinsRoom(room_id: number, joinee: LobbyUser): void;
    viewerJoinsRoom(room_id: number, joinee: LobbyUser): void;
    clientLeavesRoom(room_id: number, client_id: number): void;
    updateRoomSettings(room_id: number, new_settings: GameSettings): void;
    launchRoom(room_id: number, game_id: number): void;
    gameOver(room_id: number): void;
}
export {};
