import {GameBase, GamePlayer} from '../data_models';
import {StandardCard} from '../util/card_util';

/** Data describing a game of euchre */
export declare interface GameEuchre {
  game_base: GameBase;
  players: EuchrePlayer[];
  teams: EuchreTeam[];
  bidding: boolean;
  bidding_choosing_trump: boolean;
  dealer_substituting_card: boolean;
  player_bid: number;
  makers_team: number;
  defenders_team: number;
  going_alone: boolean;
  dealer: number;
  round: number;
  trick_leader: number;
  trick: StandardCard[];
  trump_suit: number;
  turn: number;
}

/** Data describing a euchre player */
export declare interface EuchrePlayer {
  player: GamePlayer;
  cards: StandardCard[];
  cards_played: number[];
}

/** Data describing a euchre team */
export declare interface EuchreTeam {
  score: number;
  tricks: number;
}