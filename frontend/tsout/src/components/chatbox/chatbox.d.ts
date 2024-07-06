import { DwgElement } from '../dwg_element';
import './chatbox.scss';
export declare interface ChatMessage {
    message: string;
    sender?: string;
    color?: string;
}
export declare const SERVER_CHAT_NAME = "!!server!!";
export declare class DwgChatbox extends DwgElement {
    chat_container: HTMLDivElement;
    chat_input: HTMLInputElement;
    send_chat: HTMLButtonElement;
    new_messages_button: HTMLButtonElement;
    new_messages_number: HTMLSpanElement;
    convert_emoticons: boolean;
    constructor();
    protected parsedCallback(): void;
    scrolledUp(): boolean;
    adjustScroll(): void;
    scrolledToBottom(): void;
    static adjust_scroll_limit: number;
    last_new_messages_button_timer: number;
    last_new_messages_button_count: number;
    new_chat_elements: HTMLDivElement[];
    addChat(message: ChatMessage, you_sent?: boolean): void;
    sendChat(): void;
    setPlaceholder(placeholder: string): void;
    focus(): void;
    clear(): void;
    private inputEmpty;
    private getInput;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-chatbox': DwgChatbox;
    }
}
