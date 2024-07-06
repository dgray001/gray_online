import { DwgElement } from '../../../dwg_element';
import { Point2D } from '../objects2d';
import './canvas_board.scss';
export declare interface CanvasBoardInitializationData {
    board_size: Point2D;
    max_scale: number;
    fill_space?: boolean;
    allow_side_move?: boolean;
    draw: (ctx: CanvasRenderingContext2D, transform: BoardTransformData) => void;
    mousemove: (m: Point2D, transform: BoardTransformData) => void;
    mouseleave: () => void;
    mousedown: (e: MouseEvent) => boolean;
    mouseup: (e: MouseEvent) => void;
    zoom_config: ZoomConfig;
}
export declare interface CanvasBoardSize {
    board_size: Point2D;
    el_size: DOMRect;
}
export declare interface BoardTransformData {
    scale: number;
    view: Point2D;
}
export declare interface ZoomConfig {
    zoom_constant: number;
    max_zoom?: number;
    min_zoom?: number;
}
export declare class DwgCanvasBoard extends DwgElement {
    private canvas;
    private cursor;
    private ctx;
    private data;
    private orig_size;
    private transform;
    private zoom_config;
    private hovered;
    private holding_keys;
    private cursor_move_threshold;
    private dragging;
    private mouse;
    private cursor_in_range;
    private bounding_rect;
    private resize_observer;
    constructor();
    getBoundingRect(): DOMRect;
    initialize(data: CanvasBoardInitializationData): Promise<CanvasBoardSize>;
    updateSize(data: CanvasBoardInitializationData, override_rect?: DOMRect): Promise<boolean>;
    private setSize;
    private addEventListeners;
    private tick;
    scaleView(scale: number): void;
    setView(view: Point2D): void;
    getMaxScale(): number;
    setMaxScale(max_scale: number): void;
    setScale(scale: number): number;
}
declare global {
    interface HTMLElementTagNameMap {
        'dwg-canvas-board': DwgCanvasBoard;
    }
}
