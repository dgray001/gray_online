import { DwgDialogBox } from '../../dialog_box/dialog_box';
import { LobbyUser } from '../../lobby/data_models';
import { GamePlayer } from '../data_models';
import './players_dialog.scss';
import './players_dialog_player/players_dialog_player';
interface PlayersData {
    room_id: number;
    players: GamePlayer[];
    lobby_players: Map<number, LobbyUser>;
}
export declare class DwgPlayersDialog extends DwgDialogBox<PlayersData> {
    close_button: HTMLButtonElement;
    players_container: HTMLDivElement;
    data: PlayersData;
    constructor();
    getHTML(): string;
    getData(): PlayersData;
    setData(data: PlayersData, parsed?: boolean): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-players-dialog': DwgPlayersDialog;
    }
}
export {};
