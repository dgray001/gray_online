import type { BoundedNumber} from '../../../../../scripts/math';
import { setBoundedNumber, validateBoundedNumber } from '../../../../../scripts/math';
import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { DwgButton } from '../button/button';
import type { CanvasComponent } from '../canvas_component';

/** Config describing a scrollbar */
export declare interface ScrollbarConfig {
  value: BoundedNumber;
  step_size: number;
}

export abstract class DwgScrollbar implements CanvasComponent {
  private config: ScrollbarConfig;
  private buttons: DwgButton[];
  private hovering = false;
  private clicking = false;

  constructor(config: ScrollbarConfig) {
    this.setConfig(config);
  }

  protected addButton(button: DwgButton) {
    this.buttons.push(button);
  }

  setConfig(config: ScrollbarConfig) {
    config.value = validateBoundedNumber(config.value);
    config.step_size = Math.min(config.step_size, config.value.value_max - config.value.value_min);
    if (config.step_size < 0) {
      config.step_size = -1 * config.step_size;
    }
    this.config = config;
  }

  value(): number {
    return this.config.value.value;
  }

  minValue(): number {
    return this.config.value.value_min;
  }

  maxValue(): number {
    return this.config.value.value_max;
  }

  /** Returns the actual dif from this operation */
  scroll(dif: number): number {
    const prev = this.value();
    this.setScroll(prev + dif);
    return this.value() - prev;
  }

  protected setScroll(v: number) {
    setBoundedNumber(this.config.value, v);
  }

  isHovering() {
    return this.hovering;
  }

  isClicking() {
    return this.clicking;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
    }
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    const button_hovering: boolean[] = [];
    for (const button of this.buttons) {
      button_hovering.push(button.mousemove(m, transform));
    }
    this.hovering = button_hovering.some((v) => !!v);
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    const button_hovering: boolean[] = [];
    for (const button of this.buttons) {
      button_hovering.push(button.mousedown(e));
    }
    this.clicking = button_hovering.some((v) => !!v);
    return this.clicking;
  }

  mouseup(e: MouseEvent): void {
    for (const button of this.buttons) {
      button.mouseup(e);
    }
  }

  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
