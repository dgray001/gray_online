import { defaultTransform, type BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';
import type { SliderConfig } from './slider_config';

export abstract class DwgSlider implements CanvasComponent {
  private hovering = false;
  private clicking = false;
  private disabled: boolean;
  private slider_value: number;
  private last_canvas: Point2D = { x: -1, y: -1 };
  private last_screen: Point2D = { x: -1, y: -1 };
  private last_transform: BoardTransformData = defaultTransform();
  private config: SliderConfig;

  constructor(config: SliderConfig) {
    this.disabled = !!config.start_disabled;
    this.config = config;
    const min = this.minValue();
    this.slider_value = config.value !== undefined ? this.snapValue(config.value) : min;
  }

  isHovering(): boolean {
    return this.hovering;
  }

  setHovering(hovering: boolean): void {
    this.hovering = hovering;
  }

  isClicking(): boolean {
    return this.clicking;
  }

  setClicking(clicking: boolean): void {
    this.clicking = clicking;
  }

  isPressed(): boolean {
    return this.clicking;
  }

  isDisabled(): boolean {
    return this.disabled;
  }

  disable(): void {
    this.disabled = true;
    this.hovering = false;
    this.clicking = false;
  }

  enable(): void {
    if (!this.disabled) {
      return;
    }
    this.disabled = false;
    this.mousemove(this.last_canvas, this.last_screen, this.last_transform);
  }

  value(): number {
    return this.slider_value;
  }

  setValue(value: number): void {
    const snapped = this.snapValue(value);
    if (snapped === this.slider_value) {
      return;
    }
    this.slider_value = snapped;
    this.changed(this.slider_value);
  }

  minValue(): number {
    if (this.config.fixed_values && this.config.fixed_values.length > 0) {
      return this.config.fixed_values[0];
    }
    return this.config.min ?? 0;
  }

  maxValue(): number {
    if (this.config.fixed_values && this.config.fixed_values.length > 0) {
      return this.config.fixed_values[this.config.fixed_values.length - 1];
    }
    return this.config.max ?? 1;
  }

  fixedValues(): number[] | undefined {
    return this.config.fixed_values;
  }

  showMarkers(): boolean {
    return this.config.show_markers ?? (!!this.config.fixed_values && this.config.fixed_values.length > 0);
  }

  valueFraction(): number {
    const min = this.minValue();
    const max = this.maxValue();
    if (max <= min) {
      return 0;
    }
    return Math.max(0, Math.min(1, (this.slider_value - min) / (max - min)));
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    this.last_transform = transform;
    this._draw(ctx, transform, dt);
  }

  mousemove(canvas: Point2D, screen: Point2D, transform: BoardTransformData): boolean {
    this.last_canvas = canvas;
    this.last_screen = screen;
    this.last_transform = transform;
    if (this.disabled) {
      this.hovering = false;
      return false;
    }
    if (this.clicking) {
      this.updateFromPoint(canvas, screen);
    }
    const previous_hovered = this.hovering;
    this.hovering = this.mouseOver(canvas, screen);
    if (!previous_hovered && this.hovering) {
      this.hovered();
    } else if (previous_hovered && !this.hovering) {
      this.unhovered();
    }
    return this.hovering || this.clicking;
  }

  mousedown(e: MouseEvent): boolean {
    if (this.disabled || e.button !== 0) {
      return false;
    }
    if (this.mouseOver(this.last_canvas, this.last_screen)) {
      this.clicking = true;
      this.updateFromPoint(this.last_canvas, this.last_screen);
      this.clicked();
      return true;
    }
    return false;
  }

  mouseup(_e: MouseEvent): void {
    if (this.clicking) {
      this.clicking = false;
      this.released();
    }
  }

  abstract xi(): number;
  abstract xf(): number;
  abstract yi(): number;
  abstract yf(): number;
  abstract w(): number;
  abstract h(): number;

  protected abstract _draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void;
  protected abstract mouseOver(canvas: Point2D, screen: Point2D): boolean;
  protected abstract fractionFromPoint(canvas: Point2D, screen: Point2D): number;
  protected abstract changed(value: number): void;

  protected hovered(): void {}
  protected unhovered(): void {}
  protected clicked(): void {}
  protected released(): void {}

  private snapValue(raw_value: number): number {
    const min = this.minValue();
    const max = this.maxValue();
    const clamped = Math.max(min, Math.min(max, raw_value));
    if (this.config.fixed_values && this.config.fixed_values.length > 0) {
      let closest = this.config.fixed_values[0];
      let min_dist = Math.abs(clamped - closest);
      for (const val of this.config.fixed_values) {
        const dist = Math.abs(clamped - val);
        if (dist < min_dist) {
          min_dist = dist;
          closest = val;
        }
      }
      return closest;
    }
    if (this.config.step && this.config.step > 0) {
      return min + Math.round((clamped - min) / this.config.step) * this.config.step;
    }
    return clamped;
  }

  private updateFromPoint(canvas: Point2D, screen: Point2D): void {
    const fraction = Math.max(0, Math.min(1, this.fractionFromPoint(canvas, screen)));
    const min = this.minValue();
    const max = this.maxValue();
    this.setValue(min + fraction * (max - min));
  }
}
