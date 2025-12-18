import type { GameBase, GamePlayer } from '../../data_models';
import type { StandardCard } from '../../util/card_util';

/** Data describing a game of fiddlesticks */
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
}

/** Data describing a fiddlesticks player */
export declare interface FiddlesticksPlayer {
  player: GamePlayer;
  cards: StandardCard[];
  cards_played: number[];
  score: number;
  bet: number;
  has_bet: boolean;
  tricks: number;
  order: number; // around table
}

/** Data describing a deal-round game-update */
interface BaseDealRound {
  round: number;
  dealer: number;
  trump: StandardCard;
}
export interface PlayerDealRound extends BaseDealRound {
  cards: StandardCard[];
}
export interface ViewerDealRound extends BaseDealRound {
  cards?: never;
}
export type DealRound = PlayerDealRound | ViewerDealRound;

/** Data describing a bet game-update */
export declare interface PlayerBet {
  amount: number;
  player_id: number;
}

/** Data describing a bet game-update */
export declare interface PlayCard {
  index: number;
  card: StandardCard;
  player_id: number;
}
