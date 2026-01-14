import type { PartialBy } from '../../../../../scripts/types';
import type { Point2D } from '../../objects2d';
import type { RectButtonConfig } from '../button/rect_button';
import { DwgRectButton } from '../button/rect_button';
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
    config.amount = config.amount ?? 1;
    if ((config.increase && config.amount < 0) || (!config.increase && config.amount > 0)) {
      config.amount = -1 * config.amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar.scroll(this.scrollbar_arrow_config.amount ?? 1);
  }
}

class DwgRectScrollbarSpaceButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarArrowButtonConfig;

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
    config.amount = config.amount ?? 10;
    if ((config.increase && config.amount < 0) || (!config.increase && config.amount > 0)) {
      config.amount = -1 * config.amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar.scroll(this.scrollbar_arrow_config.amount ?? 10);
  }
}

interface RectScrollbarBarButtonConfig {
  /** minimum size of the bar button */
  min_bar_size: number;
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
}

export abstract class DwgRectScrollbar extends DwgScrollbar {
  private rect_config: RectScrollbarConfig;

  constructor(config: RectScrollbarConfig) {
    super(config.scrollbar_config);
    this.rect_config = config;
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

export class DwgTestScrollbar extends DwgRectScrollbar {
  constructor(config: RectScrollbarConfig) {
    super(config);
  }
}
