export interface ControlGroupBoxLayout {
  x: number;
  y: number;
  cols: number;
  rows: number;
  width: number;
}
export const CONTROL_ICON_SIZE = 28;
export const CONTROL_GAP = 4;
export const CONTROL_LABEL = 14;

export function controlGroupLayout(counts: number[], width: number): ControlGroupBoxLayout[] {
  const rows = counts.length > 5 ? 2 : 4;
  const widths = counts.map((count) => Math.ceil(count / rows) * (CONTROL_ICON_SIZE + CONTROL_GAP) + CONTROL_GAP);
  const split = counts.length > 5 ? balancedSplit(widths) : counts.length;
  return layoutRow(widths.slice(0, split), width, rows, 0).concat(
    layoutRow(widths.slice(split), width, rows, 2 * (CONTROL_ICON_SIZE + CONTROL_GAP) + CONTROL_LABEL + CONTROL_GAP)
  );
}

function rowWidth(widths: number[]): number {
  return widths.reduce((sum, width) => sum + width, 0) + Math.max(0, widths.length - 1) * CONTROL_GAP;
}

function balancedSplit(widths: number[]): number {
  let split = 1;
  let best = Infinity;
  for (let i = 1; i < widths.length; i++) {
    const required = Math.max(rowWidth(widths.slice(0, i)), rowWidth(widths.slice(i)));
    if (required < best) {
      best = required;
      split = i;
    }
  }
  return split;
}

function layoutRow(widths: number[], available: number, rows: number, y: number): ControlGroupBoxLayout[] {
  const scale = Math.min(1, available / Math.max(1, rowWidth(widths)));
  let x = 0;
  return widths.map((natural) => {
    const cols = Math.max(1, Math.floor((natural * scale - CONTROL_GAP) / (CONTROL_ICON_SIZE + CONTROL_GAP)));
    const width = cols * (CONTROL_ICON_SIZE + CONTROL_GAP) + CONTROL_GAP;
    const box = { x, y, cols, rows, width };
    x += width + CONTROL_GAP;
    return box;
  });
}
