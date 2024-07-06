export declare class ColorRGB {
    private data;
    constructor(r: number, g: number, b: number, a?: number);
    setColor(r: number, g: number, b: number, a?: number): void;
    getString(): string;
    addColor(r: number, g: number, b: number, a?: number): ColorRGB;
    dAlpha(da: number): ColorRGB;
    getBrightness(): number;
    private cleanInput;
    private cleanData;
}
