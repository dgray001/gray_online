import type { GameBase, GamePlayer } from '../../data_models';
import type { StandardCard } from '../../util/card_util';

export declare interface GameFiddlesticks {
  game_base: GameBase;
  players: FiddlesticksPlayer[];
  round: number;
  min_round: number;
  max_round: number;
  rounds_increasing: boolean;
  dealer: number;
  turn: number;
  betting: boolean;
  trump: StandardCard;
  trick_leader: number;
  trick: StandardCard[];
  round_points: number;
  trick_points: number;
  turn_duration: number;
}

export declare interface FiddlesticksPlayer {
  player: GamePlayer;
  cards: StandardCard[];
  cards_played: number[];
  score: number;
  bet: number;
  has_bet: boolean;
  tricks: number;
  order: number; // around table
  turn_start_time?: number; // only set by server for the current-turn human player
}

/** Data describing a deal-round game-update */
interface BaseDealRound {
  round: number;
  dealer: number;
  trump: StandardCard;
  turn_duration: number;
  turn_start_time: number;
}
export interface PlayerDealRound extends BaseDealRound {
  cards: StandardCard[];
}
export interface ViewerDealRound extends BaseDealRound {
  cards?: never;
}
export type DealRound = PlayerDealRound | ViewerDealRound;

/** Private comparison against the default AI, only present on the acting player's own update */
export declare interface AiComparison<Choice> {
  model_id: number;
  ai_choice: Choice;
  matched: boolean;
}

/** Data describing a bet game-update */
export declare interface PlayerBet {
  ai_comparison?: AiComparison<number>;
  amount: number;
  player_id: number;
  turn_duration: number;
  turn_start_time: number;
}

/** Data describing a play-card game-update */
export declare interface PlayCard {
  ai_comparison?: AiComparison<StandardCard>;
  index: number;
  card: StandardCard;
  player_id: number;
  turn_duration: number;
  turn_start_time: number;
}
