import { DwgDialogBox } from '../../dialog_box/dialog_box';
import { LobbyRoom } from '../../lobby/data_models';
import { Game } from '../data_models';
import './game_info_dialog.scss';
interface GameInfoData {
    room: LobbyRoom;
    game: Game;
}
export declare class DwgGameInfoDialog extends DwgDialogBox<GameInfoData> {
    private close_button;
    private room_title;
    private game_title;
    private num_players_current;
    private num_players_max;
    private game_specific_settings;
    private room_description;
    private data;
    constructor();
    getHTML(): string;
    getData(): GameInfoData;
    setData(data: GameInfoData, parsed?: boolean): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-game-info-dialog': DwgGameInfoDialog;
    }
}
export {};
