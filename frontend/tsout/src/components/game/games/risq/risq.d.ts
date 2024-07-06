import { DwgElement } from '../../../dwg_element';
import { UpdateMessage } from '../../data_models';
import { DwgGame } from '../../game';
import { GameRisq, GameRisqFromServer, RisqPlayer } from './risq_data';
import './risq.scss';
import '../../util/canvas_board/canvas_board';
import './space_dialog/space_dialog';
export declare class DwgRisq extends DwgElement {
    private board;
    private game;
    private player_id;
    private hex_r;
    private hex_a;
    private canvas_center;
    private last_transform;
    private canvas_size;
    private mouse_canvas;
    private mouse_coordinate;
    private hovered_space?;
    private hovered_zone?;
    private icons;
    private last_time;
    private draw_detail;
    private left_panel;
    private right_panel;
    constructor();
    private createIcon;
    getIcon(name: string): HTMLImageElement;
    initialize(abstract_game: DwgGame, game: GameRisqFromServer): Promise<void>;
    getGame(): GameRisq;
    getPlayer(): RisqPlayer | undefined;
    private goToVillageCenter;
    private board_resize_lock;
    private boardResize;
    canvasSize(): DOMRect;
    toggleRightPanel(open?: boolean): void;
    closeLeftPanel(): void;
    gameUpdate(update: UpdateMessage): Promise<void>;
    private setNewGameData;
    private applyStartTurn;
    private draw;
    private getDrawDetail;
    private mousemove;
    private mouseleave;
    private mousedown;
    private mouseup;
    private canvasToCoordinate;
    private coordinateToCanvas;
    private removeHoveredFlags;
    private updateHoveredFlags;
    private getBoardNeighbors;
    private getBoardRows;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-risq': DwgRisq;
    }
}
