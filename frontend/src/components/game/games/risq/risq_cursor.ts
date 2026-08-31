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
