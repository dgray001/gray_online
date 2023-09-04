export declare const DEV = true;
export declare const loop: (times: number, callback: Function) => void;
export declare const asyncLoop: (times: number, callback: Function) => Promise<void>;
export declare function until(condition: () => boolean, poll_timer?: number): Promise<void>;
export declare function capitalize(str: string): string;
