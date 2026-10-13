import type { RisqCost } from '../model/types';
import { canAffordCost, RisqResourceType } from '../model/types';
import type { RisqSession } from './session';

interface AutoRenewTarget {
  building_id: number;
  count: number;
  cost: RisqCost;
}

export class RisqAutoRenewQueue {
  private targets = new Map<number, AutoRenewTarget>();
  private in_flight?: AutoRenewTarget;
  private waiters: ((success: boolean) => void)[] = [];

  constructor(
    private session: RisqSession,
    private send: (building_id: number, count: number) => void,
    private changed: () => void
  ) {}

  count(building_id: number): number {
    return this.targets.get(building_id)?.count ?? this.session.getPlayer()?.auto_renewals.get(building_id) ?? 0;
  }

  private reservedCount(building_id: number, count: number): number {
    const sent = this.in_flight?.building_id === building_id ? this.in_flight.count : 0;
    const paid = this.session.getPlayer()?.auto_renewals.get(building_id) ?? 0;
    return Math.max(0, Math.max(count, sent) - paid);
  }

  private scaledCost(cost: RisqCost, count: number): RisqCost {
    return { food: cost.food * count, wood: cost.wood * count, stone: cost.stone * count, gold: cost.gold * count };
  }

  reservations(): RisqCost[] {
    return [...this.targets.values()].map(
      (target: AutoRenewTarget): RisqCost =>
        this.scaledCost(target.cost, this.reservedCount(target.building_id, target.count))
    );
  }

  change(building_id: number, change: number, cost: RisqCost): boolean {
    const current = this.count(building_id);
    const count = Math.max(0, current + change);
    const increase = Math.max(0, this.reservedCount(building_id, count) - this.reservedCount(building_id, current));
    if (increase > 0 && !this.canAffordIncrease(this.scaledCost(cost, increase))) {
      return false;
    }
    this.targets.set(building_id, { building_id, count, cost });
    this.changed();
    this.flush();
    return true;
  }

  acknowledge(success: boolean = true): void {
    if (!this.in_flight) {
      return;
    }
    if (!success || !this.session.givingOrders() || this.session.getPlayer()?.orders_submitted) {
      this.reset();
      return;
    }
    this.in_flight = undefined;
    this.flush();
    this.changed();
  }

  drain(): Promise<boolean> {
    return this.in_flight || this.targets.size > 0
      ? new Promise<boolean>((resolve: (success: boolean) => void): void => {
          this.waiters.push(resolve);
        })
      : Promise.resolve(true);
  }

  reset(): void {
    if (!this.in_flight && this.targets.size === 0 && this.waiters.length === 0) {
      return;
    }
    this.targets.clear();
    this.in_flight = undefined;
    this.changed();
    for (const waiter of this.waiters.splice(0)) {
      waiter(false);
    }
  }

  private flush(): void {
    if (this.in_flight) {
      return;
    }
    for (const target of this.targets.values()) {
      if (target.count === (this.session.getPlayer()?.auto_renewals.get(target.building_id) ?? 0)) {
        this.targets.delete(target.building_id);
        continue;
      }
      this.in_flight = target;
      this.send(target.building_id, target.count);
      return;
    }
    for (const waiter of this.waiters.splice(0)) {
      waiter(true);
    }
  }

  private canAffordIncrease(cost: RisqCost): boolean {
    const player = this.session.getPlayer();
    if (!player || !canAffordCost(player, cost)) {
      return false;
    }
    const reserved = this.reservations();
    return (
      [
        ['food', RisqResourceType.FOOD],
        ['wood', RisqResourceType.WOOD],
        ['stone', RisqResourceType.STONE],
        ['gold', RisqResourceType.GOLD],
      ] as const
    ).every(([key, type]: readonly [keyof RisqCost, RisqResourceType]): boolean => {
      const amount = player.resources.get(type)?.amount ?? 0;
      return amount - reserved.reduce((total: number, item: RisqCost): number => total + item[key], 0) >= cost[key];
    });
  }
}
