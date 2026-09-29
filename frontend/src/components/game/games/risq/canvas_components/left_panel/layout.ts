export const PANEL_PADDING = 5;
export const ACTION_GRID_COLS = 5;
export const BUILDING_ACTION_GRID_ROWS = 3;
export const UNIT_ACTION_GRID_ROWS = 4;

/** Screen-space bounds of the left panel, as content renderers see it */
export declare interface PanelFrame {
  xi(): number;
  yi(): number;
  xf(): number;
  xc(): number;
  w(): number;
  h(): number;
}

/** Vertical bands below the stats section: optional garrison rows, then the action grid */
export declare interface LeftPanelLayout {
  separator_below_stats: number;
  garrison_rows: number;
  garrison_separator: number;
  grid_s: number;
  grid_x0: number;
  grid_y0: number;
  grid_bottom_separator: number;
}

export function emptyLayout(): LeftPanelLayout {
  return {
    separator_below_stats: 0,
    garrison_rows: 0,
    garrison_separator: 0,
    grid_s: 0,
    grid_x0: 0,
    grid_y0: 0,
    grid_bottom_separator: 0,
  };
}

/** Sizes square grid cells to fit both the panel width and the space between the stats and the bottom quarter */
export function computeLayout(
  frame: PanelFrame,
  stats_section_end: number,
  action_rows: number,
  garrison_capacity: number
): LeftPanelLayout {
  const P = PANEL_PADDING;
  const separator_below_stats = stats_section_end + P;
  const region_top = separator_below_stats + P;
  const region_bottom = frame.yi() + 0.75 * frame.h() - P;
  const garrison_rows = garrison_capacity > 0 ? Math.ceil(garrison_capacity / ACTION_GRID_COLS) : 0;
  const avail = region_bottom - region_top;
  const s = Math.min(
    (frame.w() - (ACTION_GRID_COLS + 1) * P) / ACTION_GRID_COLS,
    (avail - action_rows * P) / action_rows
  );
  const garrison_separator = region_top + garrison_rows * (s + P);
  const grid_y0 = garrison_rows > 0 ? garrison_separator + P : region_top;
  const grid_w = ACTION_GRID_COLS * s + (ACTION_GRID_COLS - 1) * P;
  return {
    separator_below_stats,
    garrison_rows,
    garrison_separator,
    grid_s: s,
    grid_x0: frame.xi() + 0.5 * (frame.w() - grid_w),
    grid_y0,
    grid_bottom_separator: grid_y0 + action_rows * (s + P),
  };
}

/** Top-left of an action-grid cell */
export function gridCell(layout: LeftPanelLayout, row: number, col: number): { x: number; y: number } {
  return {
    x: layout.grid_x0 + col * (layout.grid_s + PANEL_PADDING),
    y: layout.grid_y0 + row * (layout.grid_s + PANEL_PADDING),
  };
}
