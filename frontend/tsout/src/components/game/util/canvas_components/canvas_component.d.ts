import { BoardTransformData } from "../canvas_board/canvas_board";
import { Point2D } from "../objects2d";
export declare interface CanvasComponent {
    isHovering: () => boolean;
    isClicking: () => boolean;
    draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) => void;
    mousemove: (m: Point2D, transform: BoardTransformData) => boolean;
    mousedown: (e: MouseEvent) => boolean;
    mouseup: (e: MouseEvent) => void;
    xi: () => number;
    xf: () => number;
    yi: () => number;
    yf: () => number;
    w: () => number;
    h: () => number;
}
export declare interface DrawConfig {
    fill_style: string;
    stroke_style: string;
    stroke_width: number;
    hover_fill_style?: string;
    hover_stroke_style?: string;
    hover_stroke_width?: number;
    click_fill_style?: string;
    click_stroke_style?: string;
    click_stroke_width?: number;
    draw_clicked_when_unhovered?: boolean;
    fixed_position?: boolean;
}
export declare function configDraw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, config: DrawConfig, hovering: boolean, clicking: boolean, draw: () => void): void;
export declare interface Rotation {
    direction: boolean;
    angle: number;
}
