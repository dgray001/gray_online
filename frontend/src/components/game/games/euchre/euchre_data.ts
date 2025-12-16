import type { GameBase, GamePlayer } from '../../data_models';
import type { StandardCard } from '../../util/card_util';

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
  card_face_up: StandardCard;
  trick: StandardCard[];
  trump_suit: number;
  turn: number;
}

export function getPlayersTeam(game: GameEuchre, player_id: number): EuchreTeam {
  return game.teams[player_id % 2];
}

/** Data describing a euchre player */
export declare interface EuchrePlayer {
  player: GamePlayer;
  cards: StandardCard[];
  cards_played: number[];
  order: number; // around table
}

/** Data describing a euchre team */
export declare interface EuchreTeam {
  player_ids: number[];
  team_id: number;
  score: number;
  tricks: number;
}

/** Data describing a deal-round game-update */
export declare interface DealRound {
  round: number;
  dealer: number;
  card_face_up: StandardCard;
  cards?: StandardCard[];
}

/** Data describing a pass game-update */
export declare interface PlayerPass {
  player_id: number;
}

/** Data describing a bid game-update */
export declare interface PlayerBid {
  player_id: number;
  going_alone: boolean;
}

/** Data describing a bid-choose-trump game-update */
export declare interface BidChooseTrump {
  player_id: number;
  going_alone: boolean;
  trump_suit: number;
}

/** Data describing a dealer-substitute-card game-update */
export declare interface DealerSubstitutesCard {
  player_id: number;
  card_index: number;
}

/** Data describing a bet game-update */
export declare interface PlayCard {
  index: number;
  card: StandardCard;
  player_id: number;
}
