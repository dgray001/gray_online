import type { RisqFrontendOrder } from './risq_data';
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

/**
 * Groups same-subject, same-target BuildingCreate orders together, keeping the first (the one actively
 * producing, since backend order queues are FIFO) as its own entry and collapsing the rest behind it.
 * Every other order type passes through as its own entry.
 */
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
