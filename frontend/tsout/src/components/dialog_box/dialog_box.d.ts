import { DwgElement } from '../dwg_element';
import './dialog_box.scss';
export declare abstract class DwgDialogBox<T> extends DwgElement {
    content_container: HTMLDivElement;
    constructor();
    connectedCallback(): Promise<void>;
    protected parsedCallback(): void;
    closeDialog(): void;
    abstract getHTML(): string;
    abstract getData(): T;
    abstract setData(data: T, parsed?: boolean): void;
}
