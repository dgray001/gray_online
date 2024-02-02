import {DwgElement} from '../../../../dwg_element';
import {BoardTransformData} from '../../canvas_board/canvas_board';
import {Point2D} from '../../objects2d';
import {CanvasComponent} from '../canvas_component';

/** Possible ways one can click a button */
export enum ClickSource {
  UNKNOWN,
  LEFT_MOUSE,
  RIGHT_MOUSE,
  MIDDLE_MOUSE,
  ENTER_KEY,
  HOLD_CLICK,
}

/** Converts a javascript mouse event to a click source */
export function MouseEventToClickSource(e: MouseEvent): ClickSource {
  switch(e.button) {
    case 0:
      return ClickSource.LEFT_MOUSE;
    case 1:
      return ClickSource.MIDDLE_MOUSE;
    case 2:
      return ClickSource.RIGHT_MOUSE;
    default:
      return ClickSource.UNKNOWN;
  }
}

/** Data describing a button config */
export declare interface ButtonConfig {
  /** whether this button should continue firing clicks if user holds it */
  hold_click?: boolean;
  /** initial delay before the first hold click */
  hold_click_delay?: number;
  /** time between subsequent hold clicks */
  hold_click_time?: number;
  /** whether user must be hovering button for it to fire hold clicks */
  hold_click_hover?: boolean;
  /** only triggers click on left click */
  only_left_click?: boolean;
}

export abstract class DwgButton implements CanvasComponent {
  private hovering = false;
  private clicking = false;
  private click_hold_timer = 0;
  private hold_clicks = 0;

  private config: ButtonConfig;

  constructor(config: ButtonConfig) {
    if (config.hold_click) {
      if (config.hold_click_time < 1) {
        config.hold_click = false;
        console.error('Must set hold click time for hold click buttons');
      } else if (config.hold_click_delay < 1) {
        config.hold_click_delay = config.hold_click_time;
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
    if (this.clicking && this.config.hold_click) {
      this.click_hold_timer -= dt;
      if (this.click_hold_timer < 0) {
        this.click_hold_timer += this.config.hold_click_time;
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
    }
    return this.hovering;
  }

  mousedown(e: MouseEvent): boolean {
    const source = MouseEventToClickSource(e);
    if (source === ClickSource.UNKNOWN) {
      return false;
    }
    if (this.config.only_left_click && source !== ClickSource.LEFT_MOUSE) {
      return false;
    }
    if (this.hovering && !this.clicking) {
      this.clicking = true;
      this.click_hold_timer = this.config.hold_click_delay;
      this.hold_clicks = 0;
      this.clicked(source);
      return true;
    }
    return false;
  }

  mouseup(e: MouseEvent) {
    const source = MouseEventToClickSource(e);
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
