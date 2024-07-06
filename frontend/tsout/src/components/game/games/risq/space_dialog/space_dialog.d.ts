import { DwgDialogBox } from '../../../../dialog_box/dialog_box';
import { DwgRisq } from '../risq';
import { RisqSpace } from '../risq_data';
import './space_dialog.scss';
export declare interface SpaceDialogData {
    space: RisqSpace;
    game: DwgRisq;
}
export declare class DwgSpaceDialog extends DwgDialogBox<SpaceDialogData> {
    private wrapper;
    private canvas;
    private close_button;
    private data;
    private size;
    private radius;
    private ctx;
    private draw_interval?;
    private mouse;
    private hovered_zone?;
    private hovered_space?;
    private hovered;
    private icons;
    constructor();
    private createIcon;
    getHTML(): string;
    closeDialog(): void;
    getData(): SpaceDialogData;
    setData(data: SpaceDialogData, parsed?: boolean): void;
    private draw;
    private mousemove;
    private unhoverZone;
    private resolveHoveredZone;
    private mousedown;
    private mouseup;
    private keyup;
    private setZoneFill;
    private drawZone;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-space-dialog': DwgSpaceDialog;
    }
}
