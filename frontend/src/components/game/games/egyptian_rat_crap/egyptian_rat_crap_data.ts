import type { GameBase, GamePlayer } from '../../data_models';
import type { StandardCard } from '../../util/card_util';

/** Data describing a game of egyptian rat crap */
export declare interface GameEgyptianRatCrap {
  game_base: GameBase;
  players: EgyptianRatCrapPlayer[];
  turn: number;
  challenge_active: boolean;
  challenger_id: number;
  chances_left: number;
  pile_pending: boolean;
  central_pile: StandardCard[];
}

/** Data describing a player in a game of egyptian rat crap */
export declare interface EgyptianRatCrapPlayer {
  player: GamePlayer;
  pile_count: number;
  slapped: boolean;
  order: number; // around table
}

/** Data describing a deal game-update */
export declare interface Deal {
  turn: number;
  pile_counts: number[];
}

/** Data describing a flip-card game-update */
export declare interface FlipCard {
  card: StandardCard;
  player_id: number;
  played_over_valid_slap: boolean;
  challenger_id?: number;
  chances_left?: number;
  challenge_busted?: boolean;
  turn: number;
}

/** Data describing a pile-awarded game-update (challenge resolution, after the grace window) */
export declare interface PileAwarded {
  pile_awarded_to: number;
  pile_size: number;
}

/** Data describing a turn-update game-update */
export declare interface TurnUpdate {
  turn: number;
}

/** Data describing a slap-result game-update */
export declare interface SlapResult {
  player_id: number;
  valid: boolean;
  pile_size?: number;
  pile_empty?: boolean;
}
