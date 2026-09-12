import type { RisqFrontendOrder, RisqOrder } from './risq_data';
import { RisqOrderType } from './risq_data';

/** Returns whether the order is for units */
export function isUnitOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_UnitMoveSpace && order <= RisqOrderType.OrderType_UnitDelete;
}

/** Returns whether the order is for buildings */
export function isBuildingOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_BuildingCreate && order <= RisqOrderType.OrderType_BuildingDelete;
}

/** Returns whether the order is a subject-less, player-level order */
export function isPlayerOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_CancelOrder && order <= RisqOrderType.OrderType_CancelFoundation;
}

/** Returns the arrow color used to draw the order on the map */
export function orderArrowColor(order_type: RisqOrderType): string {
  switch (order_type) {
    case RisqOrderType.OrderType_UnitGather:
      return 'rgba(255, 150, 0, 0.9)';
    case RisqOrderType.OrderType_UnitAttackSpace:
    case RisqOrderType.OrderType_UnitAttackZone:
    case RisqOrderType.OrderType_UnitAttackUnit:
    case RisqOrderType.OrderType_UnitAttackBuilding:
      return 'rgba(220, 30, 30, 0.9)';
    case RisqOrderType.OrderType_UnitBuild:
      return 'rgba(40, 110, 230, 0.9)';
    case RisqOrderType.OrderType_UnitRepair:
      return 'rgba(100, 170, 250, 0.9)';
    case RisqOrderType.OrderType_UnitGarrison:
      return 'rgba(255, 225, 0, 0.9)';
    case RisqOrderType.OrderType_UnitMoveSpace:
    case RisqOrderType.OrderType_UnitMoveZone:
    default:
      return 'rgba(20, 20, 20, 0.9)';
  }
}

/** One ledger row's worth of orders: a representative order plus any same-producible orders collapsed behind it */
export declare interface RisqOrderRowEntry {
  order: RisqFrontendOrder;
  collapsed_orders?: RisqFrontendOrder[];
}

/** Groups same-subject, same-target BuildingCreate orders together, keeping the first behind */
export function collapseBuildingCreateOrders(orders: RisqFrontendOrder[]): RisqOrderRowEntry[] {
  const clusters = new Map<string, RisqFrontendOrder[]>();
  for (const order of orders) {
    if (order.order_type !== RisqOrderType.OrderType_BuildingCreate) {
      continue;
    }
    const key = `${order.subjects[0]}:${order.target_id}`;
    const cluster = clusters.get(key);
    if (cluster) {
      cluster.push(order);
    } else {
      clusters.set(key, [order]);
    }
  }
  const consumed = new Set<RisqFrontendOrder>();
  const entries: RisqOrderRowEntry[] = [];
  for (const order of orders) {
    if (consumed.has(order)) {
      continue;
    }
    if (order.order_type !== RisqOrderType.OrderType_BuildingCreate) {
      consumed.add(order);
      entries.push({ order });
      continue;
    }
    const cluster = clusters.get(`${order.subjects[0]}:${order.target_id}`)!;
    consumed.add(cluster[0]);
    entries.push({ order: cluster[0] });
    if (cluster.length > 1) {
      for (const queued of cluster.slice(1)) {
        consumed.add(queued);
      }
      entries.push({ order: cluster[1], collapsed_orders: cluster.slice(2) });
    }
  }
  return entries;
}

/** Owns this player's orders for the current turn: the server's already-active ones plus any queued locally */
export class RisqOrdersModel {
  private submitted: RisqFrontendOrder[] = [];
  private pending: RisqFrontendOrder[] = [];
  private on_change: () => void;

  constructor(on_change: () => void) {
    this.on_change = on_change;
  }

  triggerChange() {
    this.on_change();
  }

  setSubmitted(orders: RisqOrder[]) {
    this.submitted = [...orders];
  }

  revertSubmittedToPending() {
    for (const order of this.submitted) {
      order.internal_id = undefined;
    }
    this.pending.push(...this.submitted);
    this.submitted = [];
    this.on_change();
  }

  all(): RisqFrontendOrder[] {
    return [...this.submitted, ...this.pending];
  }

  pendingOrders(): RisqFrontendOrder[] {
    return [...this.pending];
  }

  private stripPendingSubject(order: RisqFrontendOrder, subject_internal_id: number): boolean {
    order.subjects = order.subjects.filter((id) => id !== subject_internal_id);
    return order.subjects.length > 0;
  }

  add(order: RisqFrontendOrder) {
    if (order.clear_previous_orders) {
      this.pending = this.pending.filter((o) => {
        if (
          !(
            (isUnitOrder(order.order_type) && isUnitOrder(o.order_type)) ||
            (isBuildingOrder(order.order_type) && isBuildingOrder(o.order_type))
          )
        ) {
          return true;
        }
        let has_subjects = true;
        for (const new_id of order.subjects) {
          has_subjects = this.stripPendingSubject(o, new_id);
        }
        return has_subjects;
      });
    }
    this.pending.push(order);
    this.on_change();
  }

  isCancelling(internal_id: number): boolean {
    return this.pending.some(
      (o) => o.order_type === RisqOrderType.OrderType_CancelOrder && o.target_id === internal_id
    );
  }

  cancel(order: RisqFrontendOrder) {
    if (order.internal_id === undefined) {
      this.pending = this.pending.filter((o) => o !== order);
      this.on_change();
      return;
    }
    if (this.isCancelling(order.internal_id)) {
      return;
    }
    this.add({
      player_id: order.player_id,
      order_type: RisqOrderType.OrderType_CancelOrder,
      subjects: [],
      target_id: order.internal_id,
      clear_previous_orders: false,
    });
  }

  cancelForSubject(subject_internal_id: number) {
    for (const order of this.all()) {
      if (!order.subjects.includes(subject_internal_id)) {
        continue;
      }
      if (order.subjects.length === 1) {
        this.cancel(order);
        continue;
      }
      if (order.internal_id === undefined) {
        if (!this.stripPendingSubject(order, subject_internal_id)) {
          this.pending = this.pending.filter((o) => o !== order);
        }
        this.on_change();
        continue;
      }
      this.cancel(order);
      this.add({
        player_id: order.player_id,
        order_type: order.order_type,
        subjects: order.subjects.filter((id) => id !== subject_internal_id),
        target_id: order.target_id,
        clear_previous_orders: false,
      });
    }
  }

  clearPending() {
    this.pending = [];
    this.on_change();
  }

  effectiveForSubject(subject_internal_id: number, kind: 'unit' | 'building'): RisqFrontendOrder[] {
    const all = this.all();
    const cancelled = new Set(
      all.filter((o) => o.order_type === RisqOrderType.OrderType_CancelOrder).map((o) => o.target_id)
    );
    const kind_matches = kind === 'unit' ? isUnitOrder : isBuildingOrder;
    let effective: RisqFrontendOrder[] = [];
    for (const order of all) {
      if (!kind_matches(order.order_type) || !order.subjects.includes(subject_internal_id)) {
        continue;
      }
      if (order.internal_id !== undefined && cancelled.has(order.internal_id)) {
        continue;
      }
      if (order.clear_previous_orders) {
        effective = [];
      }
      effective.push(order);
    }
    return effective;
  }
}
