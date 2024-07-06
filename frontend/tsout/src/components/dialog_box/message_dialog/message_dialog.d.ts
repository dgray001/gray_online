import { DwgDialogBox } from '../dialog_box';
import './message_dialog.scss';
export declare interface MessageDialogData {
    message: string;
    button_text?: string;
}
export declare class DwgMessageDialog extends DwgDialogBox<MessageDialogData> {
    message_container: HTMLDivElement;
    ok_button: HTMLButtonElement;
    constructor();
    getHTML(): string;
    getData(): MessageDialogData;
    setData(data: MessageDialogData, parsed?: boolean): void;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-message-dialog': DwgMessageDialog;
    }
}
