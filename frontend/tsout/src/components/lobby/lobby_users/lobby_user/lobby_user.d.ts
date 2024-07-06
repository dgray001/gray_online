import { DwgElement } from '../../../dwg_element';
import { LobbyUser } from '../../data_models';
import './lobby_user.scss';
export declare class DwgLobbyUser extends DwgElement {
    private ping_image;
    private ping_text;
    private user_name;
    private buttons;
    private user;
    constructor();
    protected parsedCallback(): void;
    updateUser(user: LobbyUser): void;
    private setData;
    updatePing(ping: number): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-lobby-user': DwgLobbyUser;
    }
}
