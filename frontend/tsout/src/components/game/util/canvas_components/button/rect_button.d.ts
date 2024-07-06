import { BoardTransformData } from '../../canvas_board/canvas_board';
import { Point2D } from '../../objects2d';
import { DrawConfig, Rotation } from '../canvas_component';
import { ButtonConfig, DwgButton } from './button';
export declare interface RectButtonConfig {
    button_config: ButtonConfig;
    p: Point2D;
    w: number;
    h: number;
    draw_config: DrawConfig;
    move_animation_speed?: number;
    rotate_animation_speed?: number;
    image_path?: string;
    rotation?: number;
}
export declare abstract class DwgRectButton extends DwgButton {
    private rect_config;
    private center_p;
    private radius_p;
    private speed_p;
    private target_p;
    private target_callback;
    private reached_target;
    private rotate_target;
    private rotate_speed;
    private rotate_reached;
    private rotate_callback;
    private img;
    constructor(config: RectButtonConfig);
    setPosition(p: Point2D, callback?: () => void, no_animation?: boolean): void;
    setRotation(rotation: Rotation, callback?: () => void, no_animation?: boolean): void;
    private refreshPositionDependencies;
    xi(): number;
    yi(): number;
    xf(): number;
    yf(): number;
    xc(): number;
    yc(): number;
    w(): number;
    h(): number;
    protected _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
    mouseOver(m: Point2D, transform: BoardTransformData): boolean;
}
