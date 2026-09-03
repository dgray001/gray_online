import type { GameRisqFromServer } from './risq_data';

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
