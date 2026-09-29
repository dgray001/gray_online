import type { GameRisqFromServer } from './snapshot_types';
import type { RisqGatherPoint, RisqTargetCategory, RisqUnitStance } from '../model/types';

export declare interface StartTurnData {
  game: GameRisqFromServer;
}

export declare interface SubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

export declare interface UnsubmittedOrdersData {
  game: GameRisqFromServer;
  player_id: number;
}

export declare interface UnitBehaviorSetData {
  internal_ids: number[];
  stance?: RisqUnitStance;
  interrupt_current?: boolean;
  attack_back?: boolean;
  target_priority?: RisqTargetCategory[];
}

export declare interface GatherPointSetData {
  building_id: number;
  gather_point?: RisqGatherPoint;
}
