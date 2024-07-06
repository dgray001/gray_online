import { DwgElement } from '../../../dwg_element';
import { LobbyUser } from '../../../lobby/data_models';
import { GamePlayer } from '../../data_models';
import './players_dialog_player.scss';
export declare class DwgPlayersDialogPlayer extends DwgElement {
    private ping_text;
    private ping_image;
    private nickname;
    private rejoin_link;
    private rejoin_link_text;
    private rejoin_url;
    private player;
    private room_id;
    private lobby_player;
    private copy_timeout;
    constructor();
    protected parsedCallback(): void;
    setData(player: GamePlayer, lobby_player: LobbyUser, room_id: number): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-players-dialog-player': DwgPlayersDialogPlayer;
    }
}
