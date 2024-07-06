import { BoardTransformData } from '../../../util/canvas_board/canvas_board';
import { CanvasComponent } from '../../../util/canvas_components/canvas_component';
import { Point2D } from '../../../util/objects2d';
import { DwgRisq } from '../risq';
export declare interface LeftPanelConfig {
    w: number;
    background: string;
}
export declare enum LeftPanelDataType {
    RESOURCE = 0,
    BUILDING = 1,
    SPACE = 2,
    ZONE = 3,
    MULTIPLE_PLAYERS_UNITS = 4,
    UNITS = 5,
    UNITS_BY_TYPE = 6,
    ECONOMIC_UNITS = 7,
    MILITARY_UNITS = 8,
    UNIT = 9
}
export declare enum HoverableObjectType {
    NONE = 0,
    UNIT = 1,
    BUILDING = 2,
    RESOURCE = 3
}
export declare class RisqLeftPanel implements CanvasComponent {
    private close_button;
    private risq;
    private config;
    private size;
    private showing;
    private hovering;
    private data_type;
    private visibility;
    private data;
    private buttons;
    private hovered_zone?;
    private hovered_object?;
    private hovered_object_type;
    constructor(risq: DwgRisq, config: LeftPanelConfig);
    resolveSize(): void;
    isHovering(): boolean;
    isClicking(): boolean;
    isShowing(): boolean;
    close(): void;
    openPanel(data_type: LeftPanelDataType, visibility: number, data: any): void;
    private checkUnitsData;
    private checkUnitsByTypeData;
    draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
    private drawUnitImage;
    private drawUnits;
    private drawUnit;
    private drawResource;
    private drawBuilding;
    private drawSpace;
    private hexagon_r;
    private hexagon_c;
    private drawSpaceHexagon;
    private drawZone;
    private drawName;
    private drawImage;
    private drawSeparator;
    private drawCombatStats;
    private objectHoverLogic;
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
