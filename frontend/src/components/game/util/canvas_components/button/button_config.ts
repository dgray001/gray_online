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
export function mouseEventToClickSource(e: MouseEvent): ClickSource {
  switch (e.button) {
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
  /** Config for when user holds a mouse button down */
  hold_config?: HoldButtonConfig;
  /** Only recognize a specific click source */
  allow_nonleft_clicks?: boolean;
}

/** Data describing hold click config */
export declare interface HoldButtonConfig {
  /** initial delay before the first hold click */
  hold_click_delay: number;
  /** time between subsequent hold clicks */
  hold_click_time: number;
  /** whether user must be hovering button for it to fire hold clicks */
  hold_click_hover?: boolean;
}
