import { GameBase, GamePlayer } from '../../data_models';
import { StandardCard } from '../../util/card_util';
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
export declare function getPlayersTeam(game: GameEuchre, player_id: number): EuchreTeam;
export declare interface EuchrePlayer {
    player: GamePlayer;
    cards: StandardCard[];
    cards_played: number[];
    order: number;
}
export declare interface EuchreTeam {
    player_ids: number[];
    team_id: number;
    score: number;
    tricks: number;
}
export declare interface DealRound {
    round: number;
    dealer: number;
    card_face_up: StandardCard;
    cards?: StandardCard[];
}
export declare interface PlayerPass {
    player_id: number;
}
export declare interface PlayerBid {
    player_id: number;
    going_alone: boolean;
}
export declare interface BidChooseTrump {
    player_id: number;
    going_alone: boolean;
    trump_suit: number;
}
export declare interface DealerSubstitutesCard {
    player_id: number;
    card_index: number;
}
export declare interface PlayCard {
    index: number;
    card: StandardCard;
    player_id: number;
}
