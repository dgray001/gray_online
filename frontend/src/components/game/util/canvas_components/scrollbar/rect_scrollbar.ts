import {Point2D} from "../../objects2d";
import {DwgRectButton} from "../button/rect_button";
import {DwgScrollbar, ScrollbarConfig} from "./scrollbar";

/*abstract class DwgRectScrollbarButton extends DwgRectButton {
  protected scrollbar: DwgRectScrollbar;

  constructor(scrollbar: DwgRectScrollbar) {
    super({});
    this.scrollbar = scrollbar;
  }

  protected hovered(): void {}
  protected unhovered(): void {}
  protected released(): void {}
}*/

interface RectScrollbarArrowButtonConfig {
  increase: boolean;
  /** scroll amount per click; defaults to 1 for arrows and 10 for space */
  amount?: number;
}
/*
class DwgRectScrollbarArrowButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarArrowButtonConfig;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarArrowButtonConfig) {
    super(scrollbar);
    config.amount = config.amount ?? 1;
    if ((config.increase && config.amount < 0) || (!config.increase && config.amount > 0)) {
      config.amount = -1 * config.amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar.scroll(this.scrollbar_arrow_config.amount);
  }
}
*/
/*class DwgRectScrollbarSpaceButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarArrowButtonConfig;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarArrowButtonConfig) {
    super(scrollbar);
    config.amount = config.amount ?? 10;
    if ((config.increase && config.amount < 0) || (!config.increase && config.amount > 0)) {
      config.amount = -1 * config.amount;
    }
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {
    this.scrollbar.scroll(this.scrollbar_arrow_config.amount);
  }
}
*/
interface RectScrollbarBarButtonConfig {
  /** minimum size of the bar button */
  min_bar_size: number;
}
/*
class DwgRectScrollbarBarButton extends DwgRectScrollbarButton {
  private scrollbar_arrow_config: RectScrollbarBarButtonConfig;

  constructor(scrollbar: DwgRectScrollbar, config: RectScrollbarBarButtonConfig) {
    super(scrollbar);
    this.scrollbar_arrow_config = config;
  }

  protected clicked(): void {}
}
*/
/** Config data for a rect scrollbar */
export declare interface RectScrollbarConfig {
  scrollbar_config: ScrollbarConfig;
  p: Point2D;
  w: number;
  h: number;
  vertical: boolean;
}
/*
export abstract class DwgRectScrollbar extends DwgScrollbar {
  private rect_config: RectScrollbarConfig;

  constructor(config: RectScrollbarConfig) {
    super(config.scrollbar_config);
    this.rect_config = config;
  }
}
*/