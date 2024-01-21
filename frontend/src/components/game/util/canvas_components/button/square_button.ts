import {Point2D} from '../../objects2d';
import {DrawConfig} from '../canvas_component';
import {ButtonConfig} from './button';
import {DwgRectButton} from './rect_button';

/** Config data for a square button */
export declare interface SquareButtonConfig {
  button_config: ButtonConfig;
  p: Point2D;
  s: number;
  draw_config: DrawConfig;
  move_animation_speed?: number;
  rotate_animation_speed?: number;
  image_path?: string;
  rotation?: number;
}

export abstract class DwgSquareButton extends DwgRectButton {
  constructor(config: SquareButtonConfig) {
    super({...config, w: config.s, h: config.s});
  }
}
