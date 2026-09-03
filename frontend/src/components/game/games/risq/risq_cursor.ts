import { RisqOrderType } from './risq_data';

export const DEFAULT_CURSOR_IMAGE = 'cursors/cursor';

const ORDER_CURSOR_IMAGES: Partial<Record<RisqOrderType, string>> = {
  [RisqOrderType.OrderType_UnitMoveSpace]: 'cursors/move',
  [RisqOrderType.OrderType_UnitMoveZone]: 'cursors/move',
  [RisqOrderType.OrderType_UnitGather]: 'cursors/gather',
  [RisqOrderType.OrderType_UnitBuild]: 'cursors/build',
  [RisqOrderType.OrderType_UnitRepair]: 'cursors/repair',
  [RisqOrderType.OrderType_UnitAttackSpace]: 'cursors/attack',
  [RisqOrderType.OrderType_UnitAttackZone]: 'cursors/attack',
  [RisqOrderType.OrderType_UnitAttackUnit]: 'cursors/attack',
  [RisqOrderType.OrderType_UnitAttackBuilding]: 'cursors/attack',
};

/** Returns the cursor image path for the given order type, falling back to the default cursor */
export function cursorImageForOrderType(order_type: RisqOrderType): string {
  return ORDER_CURSOR_IMAGES[order_type] ?? DEFAULT_CURSOR_IMAGE;
}

export const BUILD_CURSOR_SIZE = 48;
export const BUILD_CURSOR_ICON_SIZE = 24;
export const BUILD_PREVIEW_ALPHA_DEFAULT = 0.35;
export const BUILD_PREVIEW_ALPHA_VALID = 0.7;

export function buildOrderCursorKey(building_id: number, valid: boolean): string {
  return `build_${building_id}${valid ? '_valid' : ''}`;
}

export function drawBuildOrderCursor(
  ctx: CanvasRenderingContext2D,
  build_icon: HTMLImageElement,
  building_icon: HTMLImageElement,
  valid: boolean
) {
  ctx.globalAlpha = valid ? BUILD_PREVIEW_ALPHA_VALID : BUILD_PREVIEW_ALPHA_DEFAULT;
  ctx.drawImage(building_icon, 0, 0, BUILD_CURSOR_SIZE, BUILD_CURSOR_SIZE);
  ctx.globalAlpha = 1;
  ctx.drawImage(build_icon, 0, 0, BUILD_CURSOR_ICON_SIZE, BUILD_CURSOR_ICON_SIZE);
}
