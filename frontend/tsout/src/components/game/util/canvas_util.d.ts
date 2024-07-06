import { Point2D } from "./objects2d";
export declare function drawHexagon(ctx: CanvasRenderingContext2D, c: Point2D, r: number, o?: number): void;
export declare function drawCircle(ctx: CanvasRenderingContext2D, c: Point2D, r: number): void;
export declare function drawEllipse(ctx: CanvasRenderingContext2D, c: Point2D, r: Point2D): void;
export declare function drawRect(ctx: CanvasRenderingContext2D, pi: Point2D, w: number, h: number, r?: number): void;
export declare function drawNgon(ctx: CanvasRenderingContext2D, n: number, c: Point2D, r: number, o?: number): void;
export declare function drawLine(ctx: CanvasRenderingContext2D, pi: Point2D, pf: Point2D): void;
export declare interface DrawTextConfig {
    p: Point2D;
    w: number;
    fill_style?: string;
    stroke_style?: string;
    stroke_width?: number;
    font?: string;
    align?: CanvasTextAlign;
    baseline?: CanvasTextBaseline;
    direction?: CanvasDirection;
}
export declare function drawText(ctx: CanvasRenderingContext2D, s: string, config: DrawTextConfig): void;
