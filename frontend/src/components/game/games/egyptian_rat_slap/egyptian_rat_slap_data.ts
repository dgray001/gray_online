import type { GameBase, GamePlayer } from '../../data_models';

/** Data describing a game of egyptian rat slap */
export declare interface GameEgyptianRatSlap {
  game_base: GameBase;
  players: EgyptianRatSlapPlayer[];
}

/** Data describing a player in a game of egyptian rat slap */
export declare interface EgyptianRatSlapPlayer {
  player: GamePlayer;
}
