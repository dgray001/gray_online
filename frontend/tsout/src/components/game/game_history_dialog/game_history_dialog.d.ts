import { DwgDialogBox } from '../../dialog_box/dialog_box';
import { UpdateMessage } from '../data_models';
import './game_history_dialog.scss';
interface GameHistoryData {
    updates: Map<number, UpdateMessage>;
}
export declare class DwgGameHistoryDialog extends DwgDialogBox<GameHistoryData> {
    close_button: HTMLButtonElement;
    updates_container: HTMLDivElement;
    data: GameHistoryData;
    constructor();
    getHTML(): string;
    getData(): GameHistoryData;
    setData(data: GameHistoryData, parsed?: boolean): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-game-history-dialog': DwgGameHistoryDialog;
    }
}
export {};
