import type { PartialBy } from '../../../../../scripts/types';
import type { BoardTransformData } from '../../canvas_board/canvas_board';
import { drawRect } from '../../canvas_util';
import type { Point2D } from '../../objects2d';
import type { RectButtonConfig } from '../button/rect_button';
import { DwgRectButton } from '../button/rect_button';
import { configDraw, type DrawConfig } from '../canvas_component';
import type { ScrollbarConfig } from './scrollbar';
import { DwgScrollbar } from './scrollbar';

abstract class DwgRectScrollbarButton extends DwgRectButton {
  protected scrollbar: DwgRectScrollbar;

  constructor(scrollbar: DwgRectScrollbar, config: RectButtonConfig) {
    super(config);
    this.scrollbar = scrollbar;
  }

  updateHovering(m: Point2D, transform: BoardTransformData) {
    this.setHovering(this.mouseOver(m, transform));
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected released(): void {}
}

interface RectScrollbarArrowButtonConfig {
  increase: boolean;
  /** scroll amount per click; defaults to 1 */
  amount?: number;
  rect_button_config: PartialBy<RectButtonConfig, 'button_config'>;
}

class DwgRectScrollbarArrowButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarArrowButtonConfig;
  private scroll_amount = 1;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarArrowButtonConfig) {
    super(scrollbar, {
      button_config: {
        hold_config: {
          hold_click_delay: 600,
          hold_click_time: 20,
          hold_click_hover: false,
        },
        allow_nonleft_clicks: false,
      },
      ...config.rect_button_config,
    });
    if (config.amount) {
      this.scroll_amount = Math.abs(config.amount);
    }
    if (!config.increase) {
      this.scroll_amount = -1 * this.scroll_amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar._scroll(this.scroll_amount);
  }
}

class DwgRectScrollbarSpaceButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarArrowButtonConfig;
  private scroll_amount = 10;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarArrowButtonConfig) {
    super(scrollbar, {
      button_config: {
        hold_config: {
          hold_click_delay: 600,
          hold_click_time: 50,
          hold_click_hover: true,
        },
        allow_nonleft_clicks: false,
      },
      ...config.rect_button_config,
    });
    if (config.amount) {
      this.scroll_amount = Math.abs(config.amount);
    }
    if (!config.increase) {
      this.scroll_amount = -1 * this.scroll_amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar._scroll(this.scroll_amount);
  }
}

interface RectScrollbarBarButtonConfig {
  /** minimum size of the bar button */
  min_bar_size: number;
  /** bar dif size per value of scrolling that is possible */
  bar_dif_size: number;
  rect_button_config: PartialBy<RectButtonConfig, 'button_config'>;
}

class DwgRectScrollbarBarButton extends DwgRectScrollbarButton {
  private scrollbar_bar_config: RectScrollbarBarButtonConfig;
  private click_m = 0;
  private click_v = 0;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarBarButtonConfig) {
    super(scrollbar, {
      button_config: {
        allow_nonleft_clicks: false,
      },
      ...config.rect_button_config,
    });
    this.scrollbar_bar_config = config;
  }

  override mousemove(m: Point2D, transform: BoardTransformData): boolean {
    super.mousemove(m, transform);
    if (this.scrollbar.isFixedPosition()) {
      m = {
        x: m.x * transform.scale - transform.view.x,
        y: m.y * transform.scale - transform.view.y,
      };
    }
    if (this.isClicking() && this.click_m && this.scrollbar.getBarDifSize()) {
      const mp = this.scrollbar.isVertical() ? m.y : m.x;
      this.scrollbar.scrollTo(this.click_v + (mp - this.click_m) / this.scrollbar.getBarDifSize());
    } else {
      this.click_m = this.scrollbar.isVertical() ? m.y : m.x;
      this.click_v = this.scrollbar.value();
    }
    return this.isHovering();
  }

  protected clicked(): void {}
}

/** Config data for a rect scrollbar */
export declare interface RectScrollbarConfig {
  scrollbar_config: ScrollbarConfig;
  // Top left point of the scrollbar area
  p: Point2D;
  // width of the scrollbar area
  w: number;
  // height of the scrollbar area
  h: number;
  // width (or height for a horizontal scrollbar) of the actual scrollbar
  scrollbar_size: number;
  vertical: boolean;
  draw_config: DrawConfig;
  space_draw_config: DrawConfig;
  background_color: string;
  min_bar_size?: number;
  bar_dif_size?: number;
}

export abstract class DwgRectScrollbar extends DwgScrollbar<DwgRectScrollbarButton> {
  private rect_config: RectScrollbarConfig;
  private background_config: DrawConfig;
  private scrollbar_p: Point2D = { x: 0, y: 0 };
  private scrollbar_w = 0;
  private scrollbar_h = 0;
  private max_bar_dif_size = 10;
  private bar_dif_size = 0;
  private bar_button: DwgRectScrollbarBarButton;

  constructor(config: RectScrollbarConfig) {
    super(config.scrollbar_config);
    this.rect_config = config;
    this.max_bar_dif_size = config.bar_dif_size ?? this.max_bar_dif_size;
    this.background_config = {
      fill_style: config.background_color,
      stroke_style: 'transparent',
      stroke_width: 0,
      hover_fill_style: config.background_color,
      click_fill_style: config.background_color,
      fixed_position: true,
    };
    const arrow_button_config = {
      p: { x: 0, y: 0 },
      w: 0,
      h: 0,
      draw_config: config.draw_config,
    };
    this.addButton(
      new DwgRectScrollbarArrowButton(this, {
        increase: false,
        rect_button_config: {
          ...arrow_button_config,
        },
      })
    );
    this.addButton(
      new DwgRectScrollbarArrowButton(this, {
        increase: true,
        rect_button_config: {
          ...arrow_button_config,
        },
      })
    );
    this.bar_button = new DwgRectScrollbarBarButton(this, {
      min_bar_size: this.getMinBarSize(),
      bar_dif_size: this.bar_dif_size,
      rect_button_config: {
        ...arrow_button_config,
      },
    });
    this.addButton(this.bar_button);
    this.addButton(
      new DwgRectScrollbarSpaceButton(this, {
        increase: false,
        rect_button_config: {
          ...arrow_button_config,
          draw_config: config.space_draw_config,
        },
      })
    );
    this.addButton(
      new DwgRectScrollbarSpaceButton(this, {
        increase: true,
        rect_button_config: {
          ...arrow_button_config,
          draw_config: config.space_draw_config,
        },
      })
    );
    this.updateButtonPositions();
  }

  private getMinBarSize(): number {
    return this.rect_config.min_bar_size ?? 0.5 * this.rect_config.scrollbar_size;
  }

  getBarDifSize(): number {
    return this.bar_dif_size;
  }

  protected updateButtonPositions() {
    const size = this.rect_config.scrollbar_size;
    const vertical = this.rect_config.vertical;
    const p: Point2D = {
      x: vertical ? this.xf() - size : this.xi(),
      y: vertical ? this.yi() : this.yf() - size,
    };
    this.buttons[0].setPosition({ ...p });
    this.buttons[0].setSize(size, size);
    this.buttons[1].setPosition({
      x: this.xf() - size,
      y: this.yf() - size,
    });
    this.buttons[1].setSize(size, size);
    this.scrollbar_p = p;
    this.scrollbar_w = vertical ? size : this.w();
    this.scrollbar_h = vertical ? this.h() : size;
    const total_bar_area_size = (vertical ? this.h() : this.w()) - 2 * this.rect_config.scrollbar_size;
    const total_steps = this.totalSteps();
    const bar_size = Math.max(total_bar_area_size - this.max_bar_dif_size * total_steps, this.getMinBarSize());
    if (total_bar_area_size <= 0 || bar_size <= 0) {
      this.bar_dif_size = 0;
      this.buttons[2].setSize(0, 0);
      this.buttons[3].setSize(0, 0);
      this.buttons[4].setSize(0, 0);
      return;
    }
    this.bar_dif_size = total_steps < 1 ? 0 : (total_bar_area_size - bar_size) / total_steps;
    const bar_area_offset = this.step() * this.bar_dif_size;
    const bar_offset = this.rect_config.scrollbar_size + bar_area_offset;
    this.buttons[2].setPosition({
      x: p.x + (vertical ? 0 : bar_offset),
      y: p.y + (vertical ? bar_offset : 0),
    });
    this.buttons[2].setSize(vertical ? this.scrollbar_w : bar_size, vertical ? bar_size : this.scrollbar_h);
    this.buttons[3].setPosition({
      x: p.x + (vertical ? 0 : this.rect_config.scrollbar_size),
      y: p.y + (vertical ? this.rect_config.scrollbar_size : 0),
    });
    this.buttons[3].setSize(
      vertical ? this.scrollbar_w : bar_area_offset,
      vertical ? bar_area_offset : this.scrollbar_h
    );
    this.buttons[4].setPosition({
      x: p.x + (vertical ? 0 : bar_offset + bar_size),
      y: p.y + (vertical ? bar_offset + bar_size : 0),
    });
    this.buttons[4].setSize(
      vertical ? this.scrollbar_w : total_bar_area_size - bar_area_offset - bar_size,
      vertical ? total_bar_area_size - bar_area_offset - bar_size : this.scrollbar_h
    );
  }

  protected override setScroll(v: number): void {
    super.setScroll(v);
    this.updateButtonPositions();
    if (!this.last_mousemove_m || !this.last_mousemove_transform) {
      return;
    }
    this.buttons[2].updateHovering(this.last_mousemove_m, this.last_mousemove_transform);
    this.buttons[3].updateHovering(this.last_mousemove_m, this.last_mousemove_transform);
    this.buttons[4].updateHovering(this.last_mousemove_m, this.last_mousemove_transform);
  }

  isFixedPosition(): boolean {
    return this.rect_config.draw_config.fixed_position ?? false;
  }

  isVertical(): boolean {
    return this.rect_config.vertical;
  }

  setScrollbarSize(size: number) {
    this.rect_config.scrollbar_size = size;
    this.updateButtonPositions();
  }

  getScrollbarSize(): number {
    return this.rect_config.scrollbar_size;
  }

  setPosition(p: Point2D) {
    this.rect_config.p = { ...p };
    this.updateButtonPositions();
  }

  setW(w: number) {
    this.rect_config.w = w;
    this.updateButtonPositions();
  }

  setH(h: number) {
    this.rect_config.h = h;
    this.updateButtonPositions();
  }

  setSize(w: number, h: number) {
    this.rect_config.w = w;
    this.rect_config.h = h;
    this.updateButtonPositions();
  }

  setAllSizes(size: number, p: Point2D, w: number, h: number) {
    this.rect_config.scrollbar_size = size;
    this.rect_config.p = { ...p };
    this.rect_config.w = w;
    this.rect_config.h = h;
    this.updateButtonPositions();
  }

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(ctx, transform, this.background_config, false, false, () => {
      drawRect(ctx, { ...this.scrollbar_p }, this.scrollbar_w, this.scrollbar_h);
    });
    super.draw(ctx, transform, dt);
  }

  mouseOver(m: Point2D, transform: BoardTransformData): boolean {
    if (this.rect_config.draw_config.fixed_position) {
      m = {
        x: m.x * transform.scale - transform.view.x,
        y: m.y * transform.scale - transform.view.y,
      };
    }
    if (m.x < this.xi() || m.y < this.yi() || m.x > this.xf() || m.y > this.yf()) {
      return false;
    }
    return true;
  }

  xi(): number {
    return this.rect_config.p.x;
  }
  xf(): number {
    return this.rect_config.p.x + this.rect_config.w;
  }
  yi(): number {
    return this.rect_config.p.y;
  }
  yf(): number {
    return this.rect_config.p.y + this.rect_config.h;
  }
  w(): number {
    return this.rect_config.w;
  }
  h(): number {
    return this.rect_config.h;
  }
}
