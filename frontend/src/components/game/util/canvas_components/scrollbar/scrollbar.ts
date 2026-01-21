import type { BoundedNumber } from '../../../../../scripts/math';
import { setBoundedNumber, validateBoundedNumber } from '../../../../../scripts/math';
import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { DwgButton } from '../button/button';
import type { CanvasComponent } from '../canvas_component';

/** Data describing a scrollbar config */
export declare interface ScrollbarConfig {
  value: BoundedNumber;
  step_size: number;
  // will divide the scroll by this in pixel mode
  scroll_pixel_constant?: number;
  // how many lines a 'page' is for a page scroll event
  scroll_pages?: number;
}

export abstract class DwgScrollbar<T extends DwgButton = DwgButton> implements CanvasComponent {
  private config!: ScrollbarConfig;
  protected buttons: T[] = [];
  private hovering = false;
  private clicking = false;
  protected last_mousemove_m?: Point2D;
  protected last_mousemove_transform?: BoardTransformData;
  private scroll_pixel_constant = 80;
  private scroll_pages = 20;

  constructor(config: ScrollbarConfig) {
    this.setConfig(config);
  }

  protected addButton(button: T) {
    this.buttons.push(button);
  }

  setConfig(config: ScrollbarConfig) {
    config.value = validateBoundedNumber(config.value);
    if (config.step_size === 0) {
      config.step_size = 1;
    }
    config.step_size = Math.min(config.step_size, config.value.value_max - config.value.value_min);
    if (config.step_size < 0) {
      config.step_size = -1 * config.step_size;
    }
    if (config.scroll_pixel_constant && config.scroll_pixel_constant > 0) {
      this.scroll_pixel_constant = config.scroll_pixel_constant;
    }
    if (config.scroll_pages && config.scroll_pages > 0) {
      this.scroll_pages = config.scroll_pages;
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

  // returns how many steps the scrollbar can be scrolled
  totalSteps(): number {
    if (!this.config.step_size) {
      return 0;
    }
    const value_dif = this.maxValue() - this.minValue();
    return value_dif / this.config.step_size;
  }

  step(): number {
    if (!this.config.step_size) {
      return 0;
    }
    return (this.value() - this.minValue()) / this.config.step_size;
  }

  /** Returns the actual dif from this operation */
  _scroll(dif: number): number {
    const prev = this.value();
    this.setScroll(prev + dif);
    return this.value() - prev;
  }

  // Returns the dif that was scrolled
  scrollTo(value: number): number {
    const prev = this.value();
    this.setScroll(value);
    return this.value() - prev;
  }

  protected setScroll(v: number) {
    setBoundedNumber(this.config.value, v);
    this.scrollCallback(this.config.value.value);
  }

  isHovering() {
    return this.hovering;
  }

  setHovering(hovering: boolean) {
    this.hovering = hovering;
  }

  isClicking() {
    return this.clicking;
  }

  setClicking(clicking: boolean) {
    this.clicking = clicking;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number) {
    for (const button of this.buttons) {
      button.draw(ctx, transform, dt);
    }
  }

  scroll(dy: number, mode: number, _dx?: number): boolean {
    if (this.hovering) {
      if (mode === 0) {
        this._scroll((dy * this.config.step_size) / this.scroll_pixel_constant);
      } else if (mode === 1) {
        this._scroll(dy);
      } else if (mode === 2) {
        this._scroll(dy * this.scroll_pages);
      } else {
        console.error('Unknown scroll mode', mode);
        return false;
      }
      return true;
    }
    return false;
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    this.last_mousemove_m = m;
    this.last_mousemove_transform = transform;
    const button_hovering: boolean[] = [];
    for (const button of this.buttons) {
      button_hovering.push(button.mousemove(m, transform));
    }
    this.hovering = button_hovering.some((v) => !!v) || this.mouseOver(m, transform);
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    const button_clicking: boolean[] = [];
    for (const button of this.buttons) {
      button_clicking.push(button.mousedown(e));
    }
    this.clicking = button_clicking.some((v) => !!v);
    return this.clicking;
  }

  mouseup(e: MouseEvent): void {
    for (const button of this.buttons) {
      button.mouseup(e);
    }
  }

  abstract mouseOver(m: Point2D, transform: BoardTransformData): boolean;
  abstract scrollCallback(value: number): void;
  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;
}
