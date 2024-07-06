import { DwgElement } from '../../dwg_element';
import { LobbyUser } from '../data_models';
import './lobby_users.scss';
import './lobby_user/lobby_user';
export declare class DwgLobbyUsers extends DwgElement {
    private refresh_button;
    private loading_message;
    private user_container;
    private users;
    constructor();
    protected parsedCallback(): void;
    private first_load;
    refreshUsers(force_load_message?: boolean): Promise<void>;
    addUser(user: LobbyUser): void;
    updatePing(client_id: number, ping: number): void;
    private updatePings;
    removeUser(client_id: number): void;
    getUser(user_id: number): LobbyUser;
    joinRoom(user_id: number, room_id: number): void;
    leaveRoom(user_id: number): void;
}
