import { GameBase, GamePlayer } from '../../data_models';
import { StandardCard } from '../../util/card_util';
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
export declare interface FiddlesticksPlayer {
    player: GamePlayer;
    cards: StandardCard[];
    cards_played: number[];
    score: number;
    bet: number;
    has_bet: boolean;
    tricks: number;
    order: number;
}
export declare interface DealRound {
    round: number;
    dealer: number;
    trump: StandardCard;
    cards?: StandardCard[];
}
export declare interface PlayerBet {
    amount: number;
    player_id: number;
}
export declare interface PlayCard {
    index: number;
    card: StandardCard;
    player_id: number;
}
