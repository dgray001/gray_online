import { DwgDialogBox } from '../dialog_box';
import './confirm_dialog.scss';
interface ConfirmDialogData {
    question: string;
    yes_text?: string;
    no_text?: string;
}
export declare class DwgConfirmDialog extends DwgDialogBox<ConfirmDialogData> {
    message_container: HTMLDivElement;
    yes_button: HTMLButtonElement;
    no_button: HTMLButtonElement;
    constructor();
    getHTML(): string;
    getData(): ConfirmDialogData;
    setData(data: ConfirmDialogData, parsed?: boolean): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-confirm-dialog': DwgConfirmDialog;
    }
}
export {};
