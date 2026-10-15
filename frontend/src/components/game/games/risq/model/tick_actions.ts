import type { Point2D } from '../../../util/objects2d';
import type { RisqAttackType, RisqCost, RisqOrderType, RisqProducibleKind } from './types';

export type RisqTickTargetKind = 'unit' | 'building' | 'resource';

export declare interface RisqTickTarget {
  kind: RisqTickTargetKind;
  internal_id: number;
  player_id: number;
}

export declare interface RisqTickLocation {
  space: Point2D;
  zone: Point2D;
  garrisoned_in?: number;
}

export declare interface RisqTickOrder {
  internal_id?: number;
  order_type: RisqOrderType;
  target_id: number;
  source: string;
  source_actor?: RisqTickTarget;
}

export declare interface RisqTickEffect {
  id: number;
  tick: number;
  kind: string;
  amount: number;
  damage_type: RisqAttackType;
  order?: RisqTickOrder;
  reason?: string;
  actor?: RisqTickTarget;
  target?: RisqTickTarget;
}

export declare interface RisqTickResolution {
  kind: string;
  reason: string;
  stamina_allocated: number;
  sunk_cost: number;
}

export declare interface RisqTickIntent {
  kind: string;
  location: RisqTickLocation;
  available_stamina: number;
  min_cost: number;
  max_cost: number;
  resolution: RisqTickResolution;
  destination?: RisqTickLocation;
  item_id?: number;
  producible_kind?: RisqProducibleKind;
  target?: RisqTickTarget;
  target_location?: RisqTickLocation;
}

export declare interface RisqTickExecution {
  kind: string;
  outcome: string;
  reason: string;
  stamina_spent: number;
  progress: number;
  gathered: number;
  healing: number;
  cost: RisqCost;
  target?: RisqTickTarget;
  target_location?: RisqTickLocation;
  effects: RisqTickEffect[];
}

export declare interface RisqTickAction {
  tick: number;
  sequence: number;
  order?: RisqTickOrder;
  intent: RisqTickIntent;
  execute: RisqTickExecution;
}
