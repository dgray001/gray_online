import type { RisqFrontendOrder } from '../../model/types';
import { RisqOrderType } from '../../model/types';
import { isBuildingOrder, isPlayerOrder } from './orders_model';

export enum OrderTypeGroup {
  MOVE,
  GATHER,
  BUILD,
  REPAIR,
  RENEW,
  ATTACK,
  GARRISON,
  UNGARRISON,
  DELETE,
  CREATE,
  RESEARCH,
  HIRE,
  CANCEL,
}

export enum OrderSubjectClass {
  ECONOMIC_UNITS,
  MILITARY_UNITS,
  BUILDINGS,
  PLAYER,
}

export enum OrderHideOption {
  CANCELLED,
  OLD,
}

/** One checkbox row of a filter dropdown; the id is the enum value the filter state stores */
export declare interface OrderFilterOption {
  id: number;
  label: string;
  icon?: string;
}

export const ORDER_TYPE_OPTIONS: OrderFilterOption[] = [
  { id: OrderTypeGroup.MOVE, label: 'Move', icon: 'icons/move32' },
  { id: OrderTypeGroup.GATHER, label: 'Gather', icon: 'icons/gather32' },
  { id: OrderTypeGroup.BUILD, label: 'Build', icon: 'icons/build128' },
  { id: OrderTypeGroup.REPAIR, label: 'Repair', icon: 'icons/repair32' },
  { id: OrderTypeGroup.RENEW, label: 'Renew', icon: 'icons/wheat32' },
  { id: OrderTypeGroup.ATTACK, label: 'Attack', icon: 'icons/sword32' },
  { id: OrderTypeGroup.GARRISON, label: 'Garrison', icon: 'icons/garrison32' },
  { id: OrderTypeGroup.UNGARRISON, label: 'Ungarrison', icon: 'icons/ungarrison32' },
  { id: OrderTypeGroup.DELETE, label: 'Delete', icon: 'icons/skull32' },
  { id: OrderTypeGroup.CREATE, label: 'Create', icon: 'icons/unit64' },
  { id: OrderTypeGroup.RESEARCH, label: 'Research', icon: 'icons/research32' },
  { id: OrderTypeGroup.HIRE, label: 'Hire', icon: 'icons/swordsman' },
  { id: OrderTypeGroup.CANCEL, label: 'Cancel', icon: 'icons/close_gray32' },
];

export const ORDER_SUBJECT_OPTIONS: OrderFilterOption[] = [
  { id: OrderSubjectClass.ECONOMIC_UNITS, label: 'Economic units', icon: 'icons/villager64' },
  { id: OrderSubjectClass.MILITARY_UNITS, label: 'Military units', icon: 'icons/unit64' },
  { id: OrderSubjectClass.BUILDINGS, label: 'Buildings', icon: 'icons/building64' },
  { id: OrderSubjectClass.PLAYER, label: 'Player', icon: 'icons/person32' },
];

export const ORDER_HIDE_OPTIONS: OrderFilterOption[] = [
  { id: OrderHideOption.CANCELLED, label: 'Cancelled orders' },
  { id: OrderHideOption.OLD, label: 'Old orders' },
];

const ORDER_TYPE_GROUPS: Partial<Record<RisqOrderType, OrderTypeGroup>> = {
  [RisqOrderType.OrderType_UnitMoveSpace]: OrderTypeGroup.MOVE,
  [RisqOrderType.OrderType_UnitMoveZone]: OrderTypeGroup.MOVE,
  [RisqOrderType.OrderType_UnitGather]: OrderTypeGroup.GATHER,
  [RisqOrderType.OrderType_UnitBuild]: OrderTypeGroup.BUILD,
  [RisqOrderType.OrderType_UnitRepair]: OrderTypeGroup.REPAIR,
  [RisqOrderType.OrderType_UnitRenew]: OrderTypeGroup.RENEW,
  [RisqOrderType.OrderType_UnitAttackSpace]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitAttackZone]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitAttackUnit]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitAttackBuilding]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitAutoAttackUnit]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitAutoAttackBuilding]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_BuildingAttackUnit]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_BuildingAttackBuilding]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_BuildingAutoAttackUnit]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_BuildingAutoAttackBuilding]: OrderTypeGroup.ATTACK,
  [RisqOrderType.OrderType_UnitGarrison]: OrderTypeGroup.GARRISON,
  [RisqOrderType.OrderType_UnitUngarrison]: OrderTypeGroup.UNGARRISON,
  [RisqOrderType.OrderType_UnitDelete]: OrderTypeGroup.DELETE,
  [RisqOrderType.OrderType_BuildingDelete]: OrderTypeGroup.DELETE,
  [RisqOrderType.OrderType_BuildingCreate]: OrderTypeGroup.CREATE,
  [RisqOrderType.OrderType_BuildingResearch]: OrderTypeGroup.RESEARCH,
  [RisqOrderType.OrderType_BuyMercenary]: OrderTypeGroup.HIRE,
  [RisqOrderType.OrderType_CancelOrder]: OrderTypeGroup.CANCEL,
  [RisqOrderType.OrderType_CancelFoundation]: OrderTypeGroup.CANCEL,
};

export function orderTypeGroup(order_type: RisqOrderType): OrderTypeGroup | undefined {
  return ORDER_TYPE_GROUPS[order_type];
}

export declare interface OrderFilterContext {
  isCancelled(order: RisqFrontendOrder): boolean;
  unitSubjectClass(unit_internal_id: number): OrderSubjectClass | undefined;
}

function orderSubjectClasses(order: RisqFrontendOrder, context: OrderFilterContext): OrderSubjectClass[] {
  if (isPlayerOrder(order.order_type)) {
    return [OrderSubjectClass.PLAYER];
  }
  if (isBuildingOrder(order.order_type)) {
    return [OrderSubjectClass.BUILDINGS];
  }
  return order.subjects
    .map((id) => context.unitSubjectClass(id))
    .filter((c): c is OrderSubjectClass => c !== undefined);
}

/** Selected ids are stored as plain numbers so a dropdown can edit them without knowing which enum they are */
export class RisqOrderFilter {
  readonly subject_classes = new Set<number>();
  readonly type_groups = new Set<number>();
  readonly hidden = new Set<number>();

  matches(order: RisqFrontendOrder, context: OrderFilterContext): boolean {
    if (this.hidden.has(OrderHideOption.OLD) && order.internal_id !== undefined) {
      return false;
    }
    if (this.hidden.has(OrderHideOption.CANCELLED) && context.isCancelled(order)) {
      return false;
    }
    const group = orderTypeGroup(order.order_type);
    if (this.type_groups.size > 0 && (group === undefined || !this.type_groups.has(group))) {
      return false;
    }
    const classes = orderSubjectClasses(order, context);
    return this.subject_classes.size === 0 || classes.some((c) => this.subject_classes.has(c));
  }
}
