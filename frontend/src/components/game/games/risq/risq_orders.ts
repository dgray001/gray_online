import type { RisqFrontendOrder, RisqOrder } from './risq_data';
import { RisqOrderType } from './risq_data';

/** Returns whether the order is for units */
export function isUnitOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_UnitMoveSpace && order <= RisqOrderType.OrderType_UnitDelete;
}

/** Returns whether the order is for buildings */
export function isBuildingOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_BuildingCreate && order <= RisqOrderType.OrderType_BuildingResearch;
}

/** Returns whether the order is a subject-less, player-level order */
export function isPlayerOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_CancelOrder && order <= RisqOrderType.OrderType_CancelFoundation;
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

  setSubmitted(orders: RisqOrder[]) {
    this.submitted = [...orders];
  }

  all(): RisqFrontendOrder[] {
    return [...this.submitted, ...this.pending];
  }

  pendingOrders(): RisqFrontendOrder[] {
    return [...this.pending];
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
        for (const new_id of order.subjects) {
          o.subjects = o.subjects.filter((id) => id !== new_id);
        }
        return o.subjects.length > 0;
      });
    }
    this.pending.push(order);
    this.on_change();
  }

  cancel(order: RisqFrontendOrder) {
    if (order.internal_id === undefined) {
      this.pending = this.pending.filter((o) => o !== order);
      this.on_change();
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
      if (order.subjects.includes(subject_internal_id)) {
        this.cancel(order);
      }
    }
  }

  clearPending() {
    this.pending = [];
    this.on_change();
  }

  effectiveForSubject(subject_internal_id: number): RisqFrontendOrder[] {
    const all = this.all();
    const cancelled = new Set(
      all.filter((o) => o.order_type === RisqOrderType.OrderType_CancelOrder).map((o) => o.target_id)
    );
    let effective: RisqFrontendOrder[] = [];
    for (const order of all) {
      if (!order.subjects.includes(subject_internal_id)) {
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
