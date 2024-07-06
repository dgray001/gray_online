import { DwgElement } from '../../../dwg_element';
import { LobbyUser } from '../../data_models';
import './room_user.scss';
export declare class DwgRoomUser extends DwgElement {
    private ping_image;
    private ping_text;
    private user_name;
    private buttons;
    private user;
    private is_host;
    private is_player;
    private is_self;
    private room_launched;
    constructor();
    protected parsedCallback(): void;
    setConfig(user: LobbyUser, is_host: boolean, is_player: boolean, is_self: boolean, room_launched: boolean): void;
    updatePing(ping: number): void;
    private getUserButton;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-room-user': DwgRoomUser;
    }
}
