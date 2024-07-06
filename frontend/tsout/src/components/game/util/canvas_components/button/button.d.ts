import { BoardTransformData } from '../../canvas_board/canvas_board';
import { Point2D } from '../../objects2d';
import { CanvasComponent } from '../canvas_component';
export declare enum ClickSource {
    UNKNOWN = 0,
    LEFT_MOUSE = 1,
    RIGHT_MOUSE = 2,
    MIDDLE_MOUSE = 3,
    ENTER_KEY = 4,
    HOLD_CLICK = 5
}
export declare function MouseEventToClickSource(e: MouseEvent): ClickSource;
export declare interface ButtonConfig {
    hold_click?: boolean;
    hold_click_delay?: number;
    hold_click_time?: number;
    hold_click_hover?: boolean;
    only_left_click?: boolean;
}
export declare abstract class DwgButton implements CanvasComponent {
    private hovering;
    private clicking;
    private click_hold_timer;
    private hold_clicks;
    private config;
    constructor(config: ButtonConfig);
    isHovering(): boolean;
    protected setHovering(hovering: boolean): void;
    isClicking(): boolean;
    protected setClicking(clicking: boolean): void;
    draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
    mousemove(m: Point2D, transform: BoardTransformData): boolean;
    mousedown(e: MouseEvent): boolean;
    mouseup(e: MouseEvent): void;
    protected abstract _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, t: number): void;
    abstract mouseOver(m: Point2D, transform: BoardTransformData): boolean;
    protected abstract hovered(): void;
    protected abstract unhovered(): void;
    protected abstract clicked(source: ClickSource): void;
    protected abstract released(source: ClickSource): void;
    abstract xi(): number;
    abstract xf(): number;
    abstract yi(): number;
    abstract yf(): number;
    abstract w(): number;
    abstract h(): number;
}
