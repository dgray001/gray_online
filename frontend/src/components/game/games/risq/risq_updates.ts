import type { GameRisqFromServer, RisqGatherPoint, RisqTargetCategory, RisqUnitStance } from './risq_data';

/** Data describing a start-turn update */
export declare interface StartTurnData {
  game: GameRisqFromServer;
}

/** Data describing a submitted-orders update */
export declare interface SubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

/** Data describing an unsubmitted-orders update */
export declare interface UnsubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

/** Data describing a unit-behavior-set update */
export declare interface UnitBehaviorSetData {
  internal_ids: number[];
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
}

/** Data describing a gather-point-set update */
export declare interface GatherPointSetData {
  building_id: number;
  gather_point?: RisqGatherPoint;
}
