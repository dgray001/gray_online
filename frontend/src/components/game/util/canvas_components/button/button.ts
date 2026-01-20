import type { BoardTransformData } from '../../canvas_board/canvas_board';
import type { Point2D } from '../../objects2d';
import type { CanvasComponent } from '../canvas_component';
import { ClickSource, mouseEventToClickSource, type ButtonConfig } from './button_config';

export abstract class DwgButton implements CanvasComponent {
  private hovering = false;
  private clicking = false;
  private click_hold_timer = 0;
  private hold_clicks = 0;

  private config: ButtonConfig;

  constructor(config: ButtonConfig) {
    if (config.hold_config) {
      if (config.hold_config.hold_click_time < 1) {
        config.hold_config = undefined;
        console.error('Must set hold click time for hold click buttons');
      } else if (config.hold_config.hold_click_delay < 1) {
        config.hold_config.hold_click_delay = config.hold_config.hold_click_time;
      }
    }
    this.config = config;
  }

  isHovering() {
    return this.hovering;
  }

  protected setHovering(hovering: boolean) {
    this.hovering = hovering;
  }

  isClicking() {
    return this.clicking;
  }

  protected setClicking(clicking: boolean) {
    this.clicking = clicking;
  }

  draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    if (this.clicking && this.config.hold_config) {
      this.click_hold_timer -= dt;
      if (this.click_hold_timer < 0) {
        this.click_hold_timer += this.config.hold_config.hold_click_time;
        this.hold_clicks++;
        this.clicked(ClickSource.HOLD_CLICK);
      }
    }
    this._draw(ctx, transform, dt);
  }

  mousemove(m: Point2D, transform: BoardTransformData): boolean {
    const previous_hovered = this.hovering;
    this.hovering = this.mouseOver(m, transform);
    if (!previous_hovered && this.hovering) {
      this.hovered();
    } else if (previous_hovered && !this.hovering) {
      this.unhovered();
      if (this.clicking && this.config.hold_config?.hold_click_hover) {
        this.clicking = false;
        this.released(ClickSource.HOLD_CLICK);
      }
    }
    return this.hovering;
  }

  /** Returns whether this mousedown event triggered a click event on the button */
  mousedown(e: MouseEvent): boolean {
    const source = mouseEventToClickSource(e);
    if (source === ClickSource.UNKNOWN) {
      return false;
    }
    if (!this.config.allow_nonleft_clicks && source !== ClickSource.LEFT_MOUSE) {
      return false;
    }
    if (this.hovering && !this.clicking) {
      this.clicking = true;
      this.hold_clicks = 0;
      if (this.config.hold_config) {
        this.click_hold_timer = this.config.hold_config.hold_click_delay;
      }
      this.clicked(source);
      return true;
    }
    return false;
  }

  /** Returns whether this mouseup event triggered a release event on the button */
  mouseup(e: MouseEvent) {
    const source = mouseEventToClickSource(e);
    if (source === ClickSource.UNKNOWN) {
      return;
    }
    if (this.clicking) {
      this.clicking = false;
      this.released(source);
    }
  }

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
