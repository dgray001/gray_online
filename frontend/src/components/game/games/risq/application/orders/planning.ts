import { cantorPair, invertPair } from '../../model/coordinates';
import type {
  RisqBuilding,
  RisqCost,
  RisqFrontendOrder,
  RisqPlayer,
  RisqProducible,
  RisqResource,
  RisqUnit,
  RisqZone,
} from '../../model/types';
import { RisqOrderType, RisqProducibleKind, RisqResourceType } from '../../model/types';
import { canAffordCost } from '../../model/rules';
import type { RisqSession } from '../session';
import type { RisqOrdersModel } from './orders_model';

export declare interface LocalRisqFoundation {
  coordinate_key: number;
  building_id: number;
  display_name: string;
  order: RisqFrontendOrder;
}

/** Predictions derived from the snapshot plus pending orders: local foundations, spending, gather capacity, idleness */
export class RisqOrderPlanning {
  private local_foundations = new Map<number, LocalRisqFoundation>();
  private idle_units: RisqUnit[] = [];
  private last_idle_selected?: number;

  constructor(
    private session: RisqSession,
    private orders_model: RisqOrdersModel
  ) {}

  getLocalFoundation(key: number): LocalRisqFoundation | undefined {
    return this.local_foundations.get(key);
  }

  setLocalFoundation(foundation: LocalRisqFoundation) {
    this.local_foundations.set(foundation.coordinate_key, foundation);
  }

  hasPlannedFoundation(zone: RisqZone | undefined): boolean {
    if (!zone) {
      return false;
    }
    const has_local = this.local_foundations.has(zone.coordinate_key);
    const has_server = !!this.session.getPlayer()?.planned_foundations?.has(zone.coordinate_key);
    return has_local || has_server;
  }

  /** Recomputes everything derived from the orders model; call after any order or snapshot change */
  refresh() {
    this.syncLocalFoundations();
    this.updateResourceSpending();
    this.recomputeIdleUnits();
  }

  private syncLocalFoundations() {
    const all_orders = new Set(this.orders_model.all());
    for (const [key, f] of this.local_foundations.entries()) {
      if (!all_orders.has(f.order)) {
        this.local_foundations.delete(key);
      }
    }
  }

  private recomputeIdleUnits() {
    const player = this.session.getPlayer();
    if (!player) {
      this.idle_units = [];
      return;
    }
    this.idle_units = [...player.units.values()]
      .filter((u) => this.orders_model.effectiveForSubject(u.internal_id, 'unit').length === 0)
      .sort((a, b) => a.internal_id - b.internal_id);
  }

  idleUnitCount(): number {
    return this.idle_units.length;
  }

  idleBuildingCount(): number {
    const player = this.session.getPlayer();
    if (!player) {
      return 0;
    }
    return [...player.buildings.values()].filter(
      (b) =>
        !b.under_construction &&
        b.produces.length > 0 &&
        this.orders_model.effectiveForSubject(b.internal_id, 'building').length === 0
    ).length;
  }

  /** Cycles through idle units in internal-id order, continuing after the last one returned */
  nextIdleUnit(): RisqUnit | undefined {
    if (!this.session.getGame() || this.idle_units.length === 0) {
      return undefined;
    }
    let idx = 0;
    if (this.last_idle_selected !== undefined) {
      const found = this.idle_units.findIndex((u) => u.internal_id > this.last_idle_selected!);
      idx = found === -1 ? 0 : found;
    }
    const unit = this.idle_units[idx];
    this.last_idle_selected = unit.internal_id;
    return unit;
  }

  pendingMercenaryOrders(): RisqFrontendOrder[] {
    return this.orders_model.all().filter((o) => o.order_type === RisqOrderType.OrderType_BuyMercenary);
  }

  pendingMercenaryCount(): number {
    return this.pendingMercenaryOrders().length;
  }

  mercenaryUnavailableReason(mercenary: RisqProducible): string | undefined {
    const player = this.session.getPlayer();
    if (!player || !this.session.canGiveOrders()) {
      return 'Cannot give orders now';
    }
    if (!canAffordCost(player, mercenary.cost)) {
      return 'Not enough resources';
    }
    if (player.units.size + this.pendingMercenaryCount() >= player.population_limit) {
      return 'Population capped';
    }
    return undefined;
  }

  hasNegativeResources(): boolean {
    const player = this.session.getPlayer();
    return !!player && [...player.resources.values()].some((pr) => pr.amount - pr.spending < 0);
  }

  /** Own units gathering at the target per the server, and the set gathering there once pending orders apply */
  gatherers(target: RisqBuilding | RisqResource): { current: number; predicted: Set<number> } {
    const zone_key = cantorPair(
      cantorPair(target.space_coordinate.x, target.space_coordinate.y),
      cantorPair(target.zone_coordinate.x, target.zone_coordinate.y)
    );
    const gathers = (orders: RisqFrontendOrder[]) =>
      orders.some((o) => o.order_type === RisqOrderType.OrderType_UnitGather && o.target_id === zone_key);
    let current = 0;
    const predicted = new Set<number>();
    for (const unit of this.session.getPlayer()?.units.values() ?? []) {
      current += gathers(unit.active_orders) ? 1 : 0;
      if (gathers(this.orders_model.effectiveForSubject(unit.internal_id, 'unit'))) {
        predicted.add(unit.internal_id);
      }
    }
    return { current, predicted };
  }

  /** Keeps already-assigned gatherers plus as many new ones as the target has room for */
  fitGatherCapacity(
    target: RisqBuilding | RisqResource | undefined,
    subjects: number[]
  ): { subjects: number[]; at_capacity: boolean } {
    if (!target || target.gather_capacity === undefined) {
      return { subjects, at_capacity: false };
    }
    const { predicted } = this.gatherers(target);
    const joining = subjects.filter((id) => !predicted.has(id));
    const free = Math.max(0, target.gather_capacity - predicted.size);
    return {
      subjects: [...subjects.filter((id) => predicted.has(id)), ...joining.slice(0, free)],
      at_capacity: joining.length > free,
    };
  }

  private updateResourceSpending() {
    const player = this.session.getPlayer();
    if (!player) {
      return;
    }
    for (const pr of player.resources.values()) {
      pr.spending = 0;
    }
    for (const order of this.orders_model.pendingOrders()) {
      let kind: RisqProducibleKind;
      if (order.order_type === RisqOrderType.OrderType_BuildingCreate) {
        kind = RisqProducibleKind.UNIT;
      } else if (order.order_type === RisqOrderType.OrderType_BuildingResearch) {
        kind = RisqProducibleKind.TECH;
      } else {
        continue;
      }
      for (const subject_id of order.subjects) {
        this.addSpending(
          player,
          player.buildings.get(subject_id)?.produces.find((p) => p.kind === kind && p.id === order.target_id)?.cost
        );
      }
    }
    for (const foundation of this.local_foundations.values()) {
      this.addSpending(player, this.plannedFoundationCost(player, foundation));
    }
    for (const order of this.orders_model.pendingOrders()) {
      if (order.order_type === RisqOrderType.OrderType_BuyMercenary) {
        const unit_id = invertPair(order.target_id).x;
        this.addSpending(player, player.available_mercenaries.find((m) => m.id === unit_id)?.cost);
      } else if (order.order_type === RisqOrderType.OrderType_CancelOrder) {
        this.addSpending(player, this.paidProductionCost(player, order.target_id), -1);
      }
    }
    const renew_targets = new Set(
      this.orders_model
        .pendingOrders()
        .filter((o) => o.order_type === RisqOrderType.OrderType_UnitRenew)
        .map((o) => o.target_id)
    );
    for (const target_id of renew_targets) {
      this.addSpending(player, this.session.findBuildingById(target_id)?.renew_cost);
    }
  }

  /** Cost the server already charged for a production order still in a building's queue, refunded if it's cancelled */
  private paidProductionCost(player: RisqPlayer, order_internal_id: number): RisqCost | undefined {
    for (const building of player.buildings.values()) {
      const item = building.production_queue.find((i) => i.order_internal_id === order_internal_id);
      if (item) {
        return building.produces.find((p) => p.kind === item.kind && p.id === item.item_id)?.cost;
      }
    }
    return undefined;
  }

  private plannedFoundationCost(player: RisqPlayer, foundation: LocalRisqFoundation): RisqCost | undefined {
    for (const subject_id of foundation.order.subjects) {
      const cost = player.units.get(subject_id)?.builds.find((p) => p.id === foundation.building_id)?.cost;
      if (cost) {
        return cost;
      }
    }
    return undefined;
  }

  private addSpending(player: RisqPlayer, cost: RisqCost | undefined, multiplier = 1) {
    if (!cost) {
      return;
    }
    player.resources.get(RisqResourceType.FOOD)!.spending += multiplier * cost.food;
    player.resources.get(RisqResourceType.WOOD)!.spending += multiplier * cost.wood;
    player.resources.get(RisqResourceType.STONE)!.spending += multiplier * cost.stone;
    player.resources.get(RisqResourceType.GOLD)!.spending += multiplier * cost.gold;
  }
}
