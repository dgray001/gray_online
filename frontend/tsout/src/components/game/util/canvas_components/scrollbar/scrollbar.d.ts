import { BoundedNumber } from '../../../../../scripts/math';
import { BoardTransformData } from '../../canvas_board/canvas_board';
import { Point2D } from '../../objects2d';
import { DwgButton } from '../button/button';
import { CanvasComponent } from '../canvas_component';
export declare interface ScrollbarConfig {
    value: BoundedNumber;
    step_size: number;
}
export declare abstract class DwgScrollbar implements CanvasComponent {
    private config;
    private buttons;
    private hovering;
    private clicking;
    constructor(config: ScrollbarConfig);
    protected addButton(button: DwgButton): void;
    setConfig(config: ScrollbarConfig): void;
    value(): number;
    minValue(): number;
    maxValue(): number;
    scroll(dif: number): number;
    protected setScroll(v: number): void;
    isHovering(): boolean;
    isClicking(): boolean;
    draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
    mousemove(m: Point2D, transform: BoardTransformData): boolean;
    mousedown(e: MouseEvent): boolean;
    mouseup(e: MouseEvent): void;
    abstract xi(): number;
    abstract xf(): number;
    abstract yi(): number;
    abstract yf(): number;
    abstract w(): number;
    abstract h(): number;
}
