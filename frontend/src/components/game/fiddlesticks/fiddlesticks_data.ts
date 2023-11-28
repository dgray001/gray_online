import {GameBase, GamePlayer} from '../data_models';
import {StandardCard} from '../util/card_util';

/** Data describing a game of fiddlesticks */
export declare interface GameFiddlesticks {
  game_base: GameBase;
  players: FiddlesticksPlayer[];
  round: number;
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
  tricks: number;
  order: number; // around table
}

/** Data describing a deal-round game-update */
export declare interface DealRound {
  round: number;
  dealer: number;
  trump: StandardCard;
  cards?: StandardCard[];
}

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
