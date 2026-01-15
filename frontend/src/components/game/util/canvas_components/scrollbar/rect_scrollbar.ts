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
    this.scrollbar.scroll(this.scroll_amount);
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
    this.scrollbar.scroll(this.scroll_amount);
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
  private scrollbar_arrow_config: RectScrollbarBarButtonConfig;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarBarButtonConfig) {
    super(scrollbar, {
      button_config: {
        allow_nonleft_clicks: false,
      },
      ...config.rect_button_config,
    });
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {}
}

/** Config data for a rect scrollbar */
export declare interface RectScrollbarConfig {
  scrollbar_config: ScrollbarConfig;
  p: Point2D;
  w: number;
  h: number;
  vertical: boolean;
  draw_config: DrawConfig;
  space_draw_config: DrawConfig;
  background_color: string;
  min_bar_size?: number;
  bar_dif_size?: number;
}

export abstract class DwgRectScrollbar extends DwgScrollbar<DwgRectButton> {
  private rect_config: RectScrollbarConfig;
  private background_config: DrawConfig;

  constructor(config: RectScrollbarConfig) {
    super(config.scrollbar_config);
    this.rect_config = config;
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
    this.addButton(
      new DwgRectScrollbarArrowButton(this, {
        increase: true,
        rect_button_config: {
          ...arrow_button_config,
        },
      })
    );
    this.updateButtonPositions();
  }

  private updateButtonPositions() {
    const arrow_size = this.rect_config.vertical ? this.rect_config.w : this.rect_config.h;
    this.buttons[0].setPosition({ ...this.rect_config.p });
    this.buttons[0].setSize(arrow_size, arrow_size);
    this.buttons[1].setPosition({
      x: this.rect_config.p.x + this.rect_config.w - arrow_size,
      y: this.rect_config.p.y + this.rect_config.h - arrow_size,
    });
    this.buttons[1].setSize(arrow_size, arrow_size);
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

  override draw(ctx: CanvasRenderingContext2D, transform: BoardTransformData, dt: number): void {
    configDraw(ctx, transform, this.background_config, false, false, () => {
      drawRect(ctx, { x: this.xi(), y: this.yi() }, this.w(), this.h());
    });
    super.draw(ctx, transform, dt);
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
