import {GameBase, GamePlayer} from '../../data_models';

/** Data describing a test game */
export declare interface GameTestGame {
  game_base: GameBase;
  players: TestGamePlayer[];
}

/** Data describing a player in a test game */
export declare interface TestGamePlayer {
  player: GamePlayer;
}
