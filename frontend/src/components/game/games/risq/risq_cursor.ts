import type { DwgRisq } from './risq';
import { buildingImage } from './risq_buildings';
import { RisqOrderType } from './risq_data';

export const DEFAULT_CURSOR_IMAGE = 'cursor';

const ORDER_CURSOR_IMAGES: Partial<Record<RisqOrderType, string>> = {
  [RisqOrderType.OrderType_UnitMoveSpace]: 'move',
  [RisqOrderType.OrderType_UnitMoveZone]: 'move',
  [RisqOrderType.OrderType_UnitGather]: 'gather',
  [RisqOrderType.OrderType_UnitBuild]: 'build',
  [RisqOrderType.OrderType_UnitRepair]: 'repair',
  [RisqOrderType.OrderType_UnitAttackSpace]: 'attack',
  [RisqOrderType.OrderType_UnitAttackZone]: 'attack',
  [RisqOrderType.OrderType_UnitAttackUnit]: 'attack',
  [RisqOrderType.OrderType_UnitAttackBuilding]: 'attack',
  [RisqOrderType.OrderType_UnitGarrison]: 'garrison',
};

/** Returns the bare cursor name for the given order type (relative to the cursors/ folder), falling back to the default cursor */
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

export function resolveBuildCursorUrl(
  risq: DwgRisq,
  armed_building: { id: number; display_name: string },
  valid: boolean
): string | undefined {
  const building_icon = risq.getIcon(buildingImage(armed_building.id, false, true));
  const build_icon = risq.getIcon(`cursors/${cursorImageForOrderType(RisqOrderType.OrderType_UnitBuild)}`);
  return risq.getImageCache().getCursorUrl(
    buildOrderCursorKey(armed_building.id, valid),
    BUILD_CURSOR_SIZE,
    [building_icon, build_icon],
    (ctx) => drawBuildOrderCursor(ctx, build_icon, building_icon, valid)
  );
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
