import type { RisqBuilding, RisqGatherPoint } from '../model/types';
import type { RisqSession } from './session';

interface GatherPointTarget {
  building_id: number;
  point?: RisqGatherPoint;
}

export class RisqGatherPointQueue {
  private targets = new Map<number, GatherPointTarget>();
  private in_flight?: GatherPointTarget;
  private waiters: ((success: boolean) => void)[] = [];

  constructor(
    private session: RisqSession,
    private send: (building_id: number, point?: RisqGatherPoint) => void,
    private changed: () => void
  ) {}

  point(building: RisqBuilding): RisqGatherPoint | undefined {
    return this.targets.has(building.internal_id)
      ? this.targets.get(building.internal_id)?.point
      : building.gather_point;
  }

  change(building_id: number, point?: RisqGatherPoint): void {
    this.targets.set(building_id, { building_id, point });
    this.changed();
    this.flush();
  }

  drain(): Promise<boolean> {
    return this.targets.size > 0
      ? new Promise<boolean>((resolve: (success: boolean) => void): void => {
          this.waiters.push(resolve);
        })
      : Promise.resolve(true);
  }

  acknowledge(building_id?: number, success = true): void {
    const sent = this.in_flight;
    if (!sent || (building_id !== undefined && building_id !== sent.building_id)) {
      return;
    }
    if (this.targets.get(sent.building_id) === sent) {
      this.targets.delete(sent.building_id);
    }
    this.in_flight = undefined;
    if (!success) {
      this.settle(false);
    }
    this.flush();
    this.changed();
  }

  private settle(success: boolean): void {
    this.waiters.splice(0).forEach((resolve: (success: boolean) => void): void => resolve(success));
  }

  reset(): void {
    this.targets.clear();
    this.in_flight = undefined;
    this.settle(false);
  }

  private flush(): void {
    if (!this.session.givingOrders() || this.session.getPlayer()?.orders_submitted) {
      this.reset();
      return;
    }
    if (this.in_flight) {
      return;
    }
    for (const target of this.targets.values()) {
      const building = this.session.getPlayer()?.buildings.get(target.building_id);
      if (building && samePoint(building.gather_point, target.point)) {
        this.targets.delete(target.building_id);
      } else {
        this.in_flight = target;
        break;
      }
    }
    if (this.in_flight) {
      this.send(this.in_flight.building_id, this.in_flight.point);
    } else {
      this.settle(true);
    }
  }

  reconcile(): void {
    const sent = this.in_flight;
    const building = sent && this.session.getPlayer()?.buildings.get(sent.building_id);
    if (sent && building && samePoint(building.gather_point, sent.point)) {
      this.acknowledge(sent.building_id);
    }
  }
}

function samePoint(a: RisqGatherPoint | undefined, b: RisqGatherPoint | undefined): boolean {
  return (
    a === b ||
    (!!a &&
      !!b &&
      a.location_kind === b.location_kind &&
      a.location_id === b.location_id &&
      a.object_type === b.object_type &&
      a.object_id === b.object_id)
  );
}
