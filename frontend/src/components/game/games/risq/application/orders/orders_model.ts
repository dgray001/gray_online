import { invertPair } from '../../model/coordinates';
import type { RisqFrontendOrder, RisqOrder } from '../../model/types';
import { RisqOrderType } from '../../model/types';

/** Returns whether the order is for units */
export function isUnitOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_UnitMoveSpace && order <= RisqOrderType.OrderType_UnitDelete;
}

/** Returns whether the order is for buildings */
export function isBuildingOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_BuildingCreate && order <= RisqOrderType.OrderType_BuildingAutoAttackBuilding;
}

/** Returns whether the order survives a clear_previous_orders order on the same subject, mirroring the backend */
export function isClearImmune(order: RisqOrderType): boolean {
  return order === RisqOrderType.OrderType_BuildingCreate || order === RisqOrderType.OrderType_BuildingResearch;
}

/** Returns whether the order is a subject-less, player-level order */
export function isPlayerOrder(order: RisqOrderType): boolean {
  return order >= RisqOrderType.OrderType_CancelOrder && order <= RisqOrderType.OrderType_BuyMercenary;
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
    case RisqOrderType.OrderType_UnitAutoAttackUnit:
    case RisqOrderType.OrderType_UnitAutoAttackBuilding:
    case RisqOrderType.OrderType_BuildingAttackUnit:
    case RisqOrderType.OrderType_BuildingAttackBuilding:
    case RisqOrderType.OrderType_BuildingAutoAttackUnit:
    case RisqOrderType.OrderType_BuildingAutoAttackBuilding:
      return 'rgba(220, 30, 30, 0.9)';
    case RisqOrderType.OrderType_UnitBuild:
      return 'rgba(40, 110, 230, 0.9)';
    case RisqOrderType.OrderType_UnitRepair:
      return 'rgba(100, 170, 250, 0.9)';
    case RisqOrderType.OrderType_UnitRenew:
      return 'rgba(190, 200, 40, 0.9)';
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

export function collapseBuildingCreateOrders(
  orders: RisqFrontendOrder[],
  is_cancelled: (order: RisqFrontendOrder) => boolean
): RisqOrderRowEntry[] {
  let run_length = 0;
  const entries: RisqOrderRowEntry[] = [];
  for (const order of orders) {
    const previous = entries.at(-1);
    const same_run =
      previous &&
      order.order_type === RisqOrderType.OrderType_BuildingCreate &&
      previous.order.order_type === order.order_type &&
      previous.order.target_id === order.target_id &&
      previous.order.subjects.length === order.subjects.length &&
      previous.order.subjects.every((id: number, i: number): boolean => id === order.subjects[i]) &&
      is_cancelled(previous.order) === is_cancelled(order);
    run_length = same_run ? run_length + 1 : 1;
    if (previous && run_length > 2) {
      (previous.collapsed_orders ??= []).push(order);
    } else {
      entries.push({ order });
    }
  }
  return entries;
}

/** Owns this player's orders for the current turn: the server's already-active ones plus any queued locally */
export class RisqOrdersModel {
  private submitted: RisqFrontendOrder[] = [];
  private pending: RisqFrontendOrder[] = [];
  private on_change: () => void;
  private notify: () => void;
  private effective_cache = new Map<string, RisqFrontendOrder[]>();

  private batch_depth = 0;
  private batch_dirty = false;

  constructor(on_change: () => void) {
    this.on_change = (): void => {
      this.effective_cache.clear();
      if (this.batch_depth > 0) {
        this.batch_dirty = true;
      } else {
        on_change();
      }
    };
    this.notify = on_change;
  }

  batch(fn: () => void): void {
    this.batch_depth++;
    try {
      fn();
    } finally {
      this.batch_depth--;
    }
    if (this.batch_depth === 0 && this.batch_dirty) {
      this.batch_dirty = false;
      this.notify();
    }
  }

  triggerChange(): void {
    this.on_change();
  }

  setSubmitted(orders: RisqOrder[]): void {
    this.submitted = [...orders];
    this.effective_cache.clear();
  }

  revertSubmittedToPending(): void {
    this.pending.push(...this.submitted.filter((order) => order.internal_id === undefined));
    this.submitted = [];
    this.on_change();
  }

  all(): RisqFrontendOrder[] {
    return [...this.submitted, ...this.pending];
  }

  uncancelledOrders(): RisqFrontendOrder[] {
    return this.all().filter((order: RisqFrontendOrder): boolean => !this.isCancelling(order));
  }

  pendingOrders(): RisqFrontendOrder[] {
    return [...this.pending];
  }

  private stripPendingSubject(order: RisqFrontendOrder, subject_internal_id: number): boolean {
    order.subjects = order.subjects.filter((id) => id !== subject_internal_id);
    return order.subjects.length > 0;
  }

  add(order: RisqFrontendOrder): void {
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
        if (isClearImmune(o.order_type)) {
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

  isExplicitlyCancelling(order: RisqFrontendOrder, subject_internal_ids?: number[]): boolean {
    if (order.internal_id === undefined) {
      return false;
    }
    const explicit = this.pending.find(
      (o) => o.order_type === RisqOrderType.OrderType_CancelOrder && o.target_id === order.internal_id
    );
    if (!explicit) {
      return false;
    }
    if (explicit.subjects.length === 0) {
      return true;
    }
    const subs = subject_internal_ids ? order.subjects.filter((s) => subject_internal_ids.includes(s)) : order.subjects;
    return subs.length > 0 && subs.every((s) => explicit.subjects.includes(s));
  }

  isCancelling(order: RisqFrontendOrder, subject_internal_ids?: number[]): boolean {
    if (this.isExplicitlyCancelling(order, subject_internal_ids)) {
      return true;
    }
    if (order.order_type === RisqOrderType.OrderType_UnitBuild) {
      const subs = subject_internal_ids
        ? order.subjects.filter((s) => subject_internal_ids.includes(s))
        : order.subjects;
      if (subs.length > 0 || order.subjects.length === 0) {
        const zone_key = invertPair(order.target_id).y;
        if (
          this.pending.some(
            (p) => p.order_type === RisqOrderType.OrderType_CancelFoundation && p.target_id === zone_key
          )
        ) {
          return true;
        }
      }
    }
    if (order.internal_id === undefined || isClearImmune(order.order_type)) {
      return false;
    }
    const subs = subject_internal_ids ? order.subjects.filter((s) => subject_internal_ids.includes(s)) : order.subjects;
    const is_unit = isUnitOrder(order.order_type);
    const is_bldg = isBuildingOrder(order.order_type);
    return (
      subs.length > 0 &&
      subs.every((s) =>
        this.pending.some(
          (p) =>
            p.clear_previous_orders &&
            ((is_unit && isUnitOrder(p.order_type)) || (is_bldg && isBuildingOrder(p.order_type))) &&
            p.subjects.includes(s)
        )
      )
    );
  }

  cancel(order: RisqFrontendOrder, subject_internal_ids?: number[], behavior: 'toggle' | 'cancel' = 'toggle'): void {
    const selected = subject_internal_ids
      ? order.subjects.filter((id) => subject_internal_ids.includes(id))
      : order.subjects;
    if (subject_internal_ids !== undefined && selected.length === 0) {
      return;
    }
    if (order.internal_id === undefined) {
      if (subject_internal_ids !== undefined) {
        order.subjects = order.subjects.filter((id) => !selected.includes(id));
      }
      if (subject_internal_ids === undefined || order.subjects.length === 0) {
        this.pending = this.pending.filter((o) => o !== order);
      }
      this.on_change();
      return;
    }
    const existing = this.pending.find(
      (o) => o.order_type === RisqOrderType.OrderType_CancelOrder && o.target_id === order.internal_id
    );
    const restoring = behavior === 'toggle' && this.isExplicitlyCancelling(order, subject_internal_ids);
    if (behavior === 'toggle' && !restoring && this.isCancelling(order, subject_internal_ids)) {
      return;
    }
    if (restoring && order.order_type === RisqOrderType.OrderType_BuildingResearch) {
      this.pending = this.pending.filter(
        (o) => o.order_type !== RisqOrderType.OrderType_BuildingResearch || o.target_id !== order.target_id
      );
    }
    const cancelled = new Set(existing?.subjects.length === 0 ? order.subjects : existing?.subjects);
    for (const id of selected) {
      if (restoring) {
        cancelled.delete(id);
      } else {
        cancelled.add(id);
      }
    }
    this.pending = this.pending.filter((o) => o !== existing);
    if (cancelled.size === 0 && (restoring || subject_internal_ids !== undefined)) {
      this.on_change();
      return;
    }
    this.add({
      player_id: order.player_id,
      order_type: RisqOrderType.OrderType_CancelOrder,
      subjects: !restoring && subject_internal_ids === undefined ? [] : [...cancelled],
      target_id: order.internal_id,
      clear_previous_orders: false,
    });
  }

  cancelForSubject(subject_internal_id: number, kind: 'unit' | 'building' = 'unit'): void {
    for (const order of this.all()) {
      if (kind === 'unit' && !isUnitOrder(order.order_type)) {
        continue;
      }
      if (kind === 'building' && !isBuildingOrder(order.order_type)) {
        continue;
      }
      if (!order.subjects.includes(subject_internal_id)) {
        continue;
      }
      this.cancel(order, [subject_internal_id], 'cancel');
    }
  }

  clearPending(): void {
    this.pending = [];
    this.on_change();
  }

  effectiveForSubject(subject_internal_id: number, kind: 'unit' | 'building'): readonly RisqFrontendOrder[] {
    const key = `${kind}:${subject_internal_id}`;
    let effective = this.effective_cache.get(key);
    if (!effective) {
      effective = this.computeEffective(subject_internal_id, kind);
      this.effective_cache.set(key, effective);
    }
    return effective;
  }

  private computeEffective(subject_internal_id: number, kind: 'unit' | 'building'): RisqFrontendOrder[] {
    const all = this.all();
    const cancel_orders = all.filter((o) => o.order_type === RisqOrderType.OrderType_CancelOrder);
    const cancel_foundations = all.filter((o) => o.order_type === RisqOrderType.OrderType_CancelFoundation);
    const kind_matches = kind === 'unit' ? isUnitOrder : isBuildingOrder;
    let effective: RisqFrontendOrder[] = [];
    for (const order of all) {
      if (!kind_matches(order.order_type) || !order.subjects.includes(subject_internal_id)) {
        continue;
      }
      if (
        order.internal_id !== undefined &&
        cancel_orders.some(
          (c) =>
            c.target_id === order.internal_id && (c.subjects.length === 0 || c.subjects.includes(subject_internal_id))
        )
      ) {
        continue;
      }
      if (
        order.order_type === RisqOrderType.OrderType_UnitBuild &&
        cancel_foundations.some((c) => c.target_id === invertPair(order.target_id).y)
      ) {
        continue;
      }
      if (order.clear_previous_orders) {
        effective = effective.filter((o) => isClearImmune(o.order_type));
      }
      effective.push(order);
    }
    return effective;
  }
}
