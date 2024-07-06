import { DwgElement } from '../../../dwg_element';
import { LobbyRoom } from '../../data_models';
import './room_selector.scss';
export declare class DwgRoomSelector extends DwgElement {
    room_name: HTMLDivElement;
    room_curr_players: HTMLSpanElement;
    room_max_players: HTMLSpanElement;
    room_game: HTMLDivElement;
    button_join_player: HTMLButtonElement;
    button_join_viewer: HTMLButtonElement;
    room: LobbyRoom;
    constructor();
    protected parsedCallback(): void;
    updateRoom(room: LobbyRoom): void;
    private setRoomData;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-room-selector': DwgRoomSelector;
    }
}
