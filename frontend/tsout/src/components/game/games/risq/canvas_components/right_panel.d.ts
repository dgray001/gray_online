import { BoardTransformData } from '../../../util/canvas_board/canvas_board';
import { CanvasComponent } from '../../../util/canvas_components/canvas_component';
import { Point2D } from '../../../util/objects2d';
import { DwgRisq } from '../risq';
export declare interface RightPanelConfig {
    w: number;
    is_open: boolean;
    background: string;
}
export declare class RisqRightPanel implements CanvasComponent {
    private open_button;
    private risq;
    private config;
    private opening;
    private hovering;
    constructor(risq: DwgRisq, config: RightPanelConfig);
    isHovering(): boolean;
    isClicking(): boolean;
    isOpen(): boolean;
    toggle(open?: boolean, initial?: boolean): void;
    draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
    private drawSeparator;
    private drawPopulation;
    private drawResource;
    private drawScore;
    mousemove(m: Point2D, transform: BoardTransformData): boolean;
    mousedown(e: MouseEvent): boolean;
    mouseup(e: MouseEvent): void;
    xi(): number;
    yi(): number;
    xf(): number;
    yf(): number;
    xc(): number;
    yc(): number;
    w(): number;
    h(): number;
}
