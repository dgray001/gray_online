/// <reference types="node" />
export declare const DEV = true;
export declare const loop: (times: number, callback: (i?: number) => void) => void;
export declare const asyncLoop: (times: number, callback: Function) => Promise<void>;
export declare function until(condition: () => boolean, poll_timer?: number): Promise<void>;
export declare function untilTimer(timer: number): Promise<void>;
export declare function capitalize(str: string, word_split?: string): string;
interface ClickButtonOptions {
    loading_text?: string;
    disable_button?: boolean;
    re_enable_button?: boolean;
}
type clickButtonReturn = string | void;
export declare function clickButton(button: HTMLButtonElement, fn: () => clickButtonReturn | Promise<clickButtonReturn>, input_options?: ClickButtonOptions): void;
export declare function enumKeys<O extends object, K extends keyof O = keyof O>(obj: O): K[];
export declare function setIntervalX(fn: (counter?: number) => void, delay: number, repetitions: number, fn_end?: () => void): NodeJS.Timer;
export declare function createLock(throwaway_extras?: boolean): (fn: () => Promise<unknown>) => Promise<unknown>;
export declare function download(content: string, fileName: string, contentType?: string): void;
export declare function clientOnMobile(): boolean;
export declare function copyToClipboard(text: string): Promise<boolean>;
export {};
